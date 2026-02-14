package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TxManager manages database transactions using the context pattern.
// Nested RunInTx calls are NOT supported — calling RunInTx inside a RunInTx
// callback will create a second independent transaction, which is a bug.
type TxManager struct {
	pool *pgxpool.Pool
}

// NewTxManager creates a new TxManager.
func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

// RunInTx executes fn within a database transaction.
// Isolation level: Read Committed (PostgreSQL default).
// On success: commits.
// On error from fn: rolls back and returns the error.
// On panic from fn: rolls back and re-panics.
func (m *TxManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback(ctx)
			panic(r)
		}
	}()

	txCtx := withTx(ctx, tx)

	if err := fn(txCtx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("rollback failed: %w (original error: %v)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
