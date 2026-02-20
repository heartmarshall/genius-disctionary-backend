package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier is the common interface implemented by both *pgxpool.Pool and pgx.Tx.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

// unexported context key type for storing tx
type txCtxKey struct{}

// withTx puts a transaction into the context.
func withTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txCtxKey{}, tx)
}

// QuerierFromCtx returns the transaction from context if present,
// otherwise returns the pool.
func QuerierFromCtx(ctx context.Context, pool *pgxpool.Pool) Querier {
	if tx, ok := ctx.Value(txCtxKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}
