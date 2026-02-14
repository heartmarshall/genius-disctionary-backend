package testhelper

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver for database/sql
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	once      sync.Once
	sharedDSN string
	initErr   error
)

// SetupTestDB starts a shared PostgreSQL container (once for the entire test run),
// applies goose migrations, and returns a new pgxpool.Pool connected to it.
// The pool is closed via t.Cleanup; the container lives until the process exits.
func SetupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	once.Do(func() {
		sharedDSN, initErr = startContainerAndMigrate()
	})
	if initErr != nil {
		t.Fatalf("testhelper: failed to setup test DB: %v", initErr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, sharedDSN)
	if err != nil {
		t.Fatalf("testhelper: failed to create pgxpool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

func startContainerAndMigrate() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:17-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return "", fmt.Errorf("start container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return "", fmt.Errorf("get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return "", fmt.Errorf("get mapped port: %w", err)
	}

	dsn := fmt.Sprintf("postgres://testuser:testpass@%s:%s/testdb?sslmode=disable", host, port.Port())

	// Apply goose migrations using database/sql (goose requires *sql.DB).
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return "", fmt.Errorf("sql.Open: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return "", fmt.Errorf("db ping: %w", err)
	}

	migrationsDir := migrationsPath()

	// Use goose.NewProvider with os.DirFS — it correctly handles $$-delimited
	// PL/pgSQL functions, unlike the legacy goose.Up which splits on semicolons.
	provider, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS(migrationsDir))
	if err != nil {
		return "", fmt.Errorf("goose new provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return "", fmt.Errorf("goose up: %w", err)
	}

	return dsn, nil
}

// migrationsPath resolves the absolute path to backend_v4/migrations/ relative
// to the current source file using runtime.Caller.
func migrationsPath() string {
	_, currentFile, _, _ := runtime.Caller(0)
	// currentFile is .../backend_v4/internal/adapter/postgres/testhelper/db.go
	// Navigate up 5 levels to backend_v4/, then into migrations/
	return filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "..", "migrations")
}
