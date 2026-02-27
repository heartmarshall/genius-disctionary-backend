// Command cleanup-tokens deletes expired and revoked refresh tokens.
//
// Usage:
//
//	cleanup-tokens
//
// Requires DATABASE_DSN environment variable to be set.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		log.Fatal("DATABASE_DSN environment variable is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	tag, err := pool.Exec(ctx,
		"DELETE FROM refresh_tokens WHERE expires_at < now() OR revoked_at IS NOT NULL",
	)
	if err != nil {
		log.Fatalf("cleanup tokens: %v", err)
	}

	fmt.Printf("Deleted %d expired/revoked refresh tokens.\n", tag.RowsAffected())
}
