// Package database предоставляет базовые компоненты для работы с PostgreSQL.
package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ============================================================================
// QUERIER INTERFACE
// ============================================================================

// Querier — общий интерфейс для pgxpool.Pool и pgx.Tx.
// Позволяет использовать один и тот же код для обычных запросов и транзакций.
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Compile-time check что типы реализуют интерфейс.
var (
	_ Querier = (*pgxpool.Pool)(nil)
	_ Querier = (pgx.Tx)(nil)
)

// ============================================================================
// CLOCK
// ============================================================================

// Clock интерфейс для получения текущего времени.
// Позволяет мокать время в тестах.
type Clock interface {
	Now() time.Time
}

// RealClock возвращает реальное время.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }

// DefaultClock — дефолтная реализация часов.
var DefaultClock Clock = RealClock{}

// ============================================================================
// TRANSACTIONS
// ============================================================================

// TxFunc — функция, выполняемая в рамках транзакции.
type TxFunc func(ctx context.Context, q Querier) error

// WithTx выполняет функцию в транзакции.
// При ошибке или панике — автоматический rollback.
// При успехе — commit.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn TxFunc) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(ctx, tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}

// ============================================================================
// TX MANAGER
// ============================================================================

// TxManager управляет транзакциями и предоставляет доступ к пулу.
type TxManager struct {
	pool *pgxpool.Pool
}

// NewTxManager создаёт новый TxManager.
func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

// RunInTx выполняет функцию в рамках транзакции.
func (m *TxManager) RunInTx(ctx context.Context, fn TxFunc) error {
	return WithTx(ctx, m.pool, fn)
}

// Pool возвращает пул соединений с БД.
func (m *TxManager) Pool() *pgxpool.Pool {
	return m.pool
}

// Q возвращает Querier (пул) для обычных запросов.
func (m *TxManager) Q() Querier {
	return m.pool
}
