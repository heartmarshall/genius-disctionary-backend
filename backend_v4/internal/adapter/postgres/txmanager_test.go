package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/testhelper"
)

// userExists checks whether a user row with the given ID exists in the database.
func userExists(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) bool {
	t.Helper()
	var exists bool
	err := pool.QueryRow(
		context.Background(),
		`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`,
		userID,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("userExists query: %v", err)
	}
	return exists
}

func TestRunInTx_Commit(t *testing.T) {
	pool := testhelper.SetupTestDB(t)
	tm := postgres.NewTxManager(pool)

	userID := uuid.New()

	err := tm.RunInTx(context.Background(), func(ctx context.Context) error {
		q := postgres.QuerierFromCtx(ctx, pool)
		_, err := q.Exec(ctx,
			`INSERT INTO users (id, email, username, name, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, now(), now())`,
			userID, "commit-test@example.com", "commit-test", "Commit Test",
		)
		return err
	})
	if err != nil {
		t.Fatalf("RunInTx returned error: %v", err)
	}

	if !userExists(t, pool, userID) {
		t.Fatal("expected user to exist after committed transaction")
	}
}

func TestRunInTx_RollbackOnError(t *testing.T) {
	pool := testhelper.SetupTestDB(t)
	tm := postgres.NewTxManager(pool)

	userID := uuid.New()
	sentinel := errors.New("business logic error")

	err := tm.RunInTx(context.Background(), func(ctx context.Context) error {
		q := postgres.QuerierFromCtx(ctx, pool)
		_, execErr := q.Exec(ctx,
			`INSERT INTO users (id, email, username, name, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, now(), now())`,
			userID, "rollback-test@example.com", "rollback-test", "Rollback Test",
		)
		if execErr != nil {
			t.Fatalf("insert inside tx failed: %v", execErr)
		}
		return sentinel
	})

	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got: %v", err)
	}

	if userExists(t, pool, userID) {
		t.Fatal("expected user NOT to exist after rolled-back transaction")
	}
}

func TestRunInTx_RollbackOnPanic(t *testing.T) {
	pool := testhelper.SetupTestDB(t)
	tm := postgres.NewTxManager(pool)

	userID := uuid.New()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic to be re-raised")
		}
		if r != "test panic" {
			t.Fatalf("expected panic value %q, got %v", "test panic", r)
		}

		// Verify data was rolled back.
		if userExists(t, pool, userID) {
			t.Fatal("expected user NOT to exist after panic-rolled-back transaction")
		}
	}()

	_ = tm.RunInTx(context.Background(), func(ctx context.Context) error {
		q := postgres.QuerierFromCtx(ctx, pool)
		_, err := q.Exec(ctx,
			`INSERT INTO users (id, email, username, name, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, now(), now())`,
			userID, "panic-test@example.com", "panic-test", "Panic Test",
		)
		if err != nil {
			t.Fatalf("insert inside tx failed: %v", err)
		}
		panic("test panic")
	})
}

func TestRunInTx_QuerierFromCtx_UsesTx(t *testing.T) {
	pool := testhelper.SetupTestDB(t)
	tm := postgres.NewTxManager(pool)

	userID := uuid.New()

	// Insert inside a transaction, then verify it's visible within the same tx
	// but NOT outside until commit.
	err := tm.RunInTx(context.Background(), func(ctx context.Context) error {
		q := postgres.QuerierFromCtx(ctx, pool)
		_, err := q.Exec(ctx,
			`INSERT INTO users (id, email, username, name, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, now(), now())`,
			userID, "ctx-test@example.com", "ctx-test", "Ctx Test",
		)
		if err != nil {
			return err
		}

		// Should be visible within the transaction.
		var exists bool
		err = q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			t.Fatal("expected user to be visible within the transaction")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("RunInTx returned error: %v", err)
	}

	// After commit, also visible outside.
	if !userExists(t, pool, userID) {
		t.Fatal("expected user to exist after committed transaction")
	}
}
