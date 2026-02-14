package testhelper

import (
	"context"
	"testing"
)

func TestSetupTestDB_Smoke(t *testing.T) {
	pool := SetupTestDB(t)

	user := SeedUser(t, pool)

	// Verify user exists in DB via SELECT.
	var email string
	err := pool.QueryRow(
		context.Background(),
		`SELECT email FROM users WHERE id = $1`,
		user.ID,
	).Scan(&email)
	if err != nil {
		t.Fatalf("expected user in DB, got error: %v", err)
	}

	if email != user.Email {
		t.Fatalf("expected email %q, got %q", user.Email, email)
	}
}
