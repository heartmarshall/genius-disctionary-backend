// Command promote sets a user's role to admin by email address.
// It is used to bootstrap the first admin user.
//
// Usage:
//
//	promote --email=user@example.com
//
// Requires DATABASE_DSN environment variable to be set.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	email := flag.String("email", "", "email of user to promote to admin")
	flag.Parse()

	if *email == "" {
		fmt.Fprintln(os.Stderr, "Usage: promote --email=user@example.com")
		os.Exit(1)
	}

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
		"UPDATE users SET role = 'admin', updated_at = now() WHERE email = $1 AND role != 'admin'",
		*email,
	)
	if err != nil {
		log.Fatalf("update role: %v", err)
	}

	if tag.RowsAffected() == 0 {
		fmt.Printf("No user found with email %q, or already admin.\n", *email)
		os.Exit(1)
	}

	fmt.Printf("User %q promoted to admin.\n", *email)
}
