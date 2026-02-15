package token_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/testhelper"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/token"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// newRepo is a test helper that sets up the DB and returns a ready Repo.
func newRepo(t *testing.T) (*token.Repo, *pgxpool.Pool) {
	t.Helper()
	pool := testhelper.SetupTestDB(t)
	return token.New(pool), pool
}

// ---------------------------------------------------------------------------
// Create + GetByHash
// ---------------------------------------------------------------------------

func TestRepo_Create_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	hash := "testhash-" + uuid.New().String()[:8]
	expiresAt := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Microsecond)

	got, err := repo.Create(ctx, user.ID, hash, expiresAt)
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	if got.ID == uuid.Nil {
		t.Error("ID should not be nil")
	}
	if got.UserID != user.ID {
		t.Errorf("UserID mismatch: got %s, want %s", got.UserID, user.ID)
	}
	if got.TokenHash != hash {
		t.Errorf("TokenHash mismatch: got %q, want %q", got.TokenHash, hash)
	}
	if !got.ExpiresAt.Equal(expiresAt) {
		t.Errorf("ExpiresAt mismatch: got %v, want %v", got.ExpiresAt, expiresAt)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if got.RevokedAt != nil {
		t.Errorf("RevokedAt should be nil, got %v", got.RevokedAt)
	}
}

func TestRepo_Create_InvalidUserID(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	hash := "testhash-invalid-user-" + uuid.New().String()[:8]
	expiresAt := time.Now().UTC().Add(24 * time.Hour)

	// Non-existent user_id should trigger foreign key violation -> ErrNotFound.
	_, err := repo.Create(ctx, uuid.New(), hash, expiresAt)
	assertIsDomainError(t, err, domain.ErrNotFound)
}

func TestRepo_GetByHash_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	hash := "gethash-" + uuid.New().String()[:8]
	expiresAt := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Microsecond)

	created, err := repo.Create(ctx, user.ID, hash, expiresAt)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByHash(ctx, hash)
	if err != nil {
		t.Fatalf("GetByHash: unexpected error: %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, created.ID)
	}
	if got.UserID != user.ID {
		t.Errorf("UserID mismatch: got %s, want %s", got.UserID, user.ID)
	}
	if got.TokenHash != hash {
		t.Errorf("TokenHash mismatch: got %q, want %q", got.TokenHash, hash)
	}
}

func TestRepo_GetByHash_NotFound(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	_, err := repo.GetByHash(ctx, "nonexistent-hash-"+uuid.New().String()[:8])
	assertIsDomainError(t, err, domain.ErrNotFound)
}

func TestRepo_GetByHash_Expired(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	hash := "expired-hash-" + uuid.New().String()[:8]
	// Token already expired 1 hour ago.
	expiresAt := time.Now().UTC().Add(-1 * time.Hour)

	_, err := repo.Create(ctx, user.ID, hash, expiresAt)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err = repo.GetByHash(ctx, hash)
	assertIsDomainError(t, err, domain.ErrNotFound)
}

func TestRepo_GetByHash_Revoked(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	hash := "revoked-hash-" + uuid.New().String()[:8]
	expiresAt := time.Now().UTC().Add(24 * time.Hour)

	created, err := repo.Create(ctx, user.ID, hash, expiresAt)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.RevokeByID(ctx, created.ID); err != nil {
		t.Fatalf("RevokeByID: %v", err)
	}

	_, err = repo.GetByHash(ctx, hash)
	assertIsDomainError(t, err, domain.ErrNotFound)
}

// ---------------------------------------------------------------------------
// RevokeByID
// ---------------------------------------------------------------------------

func TestRepo_RevokeByID_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	hash := "revoke-id-" + uuid.New().String()[:8]
	expiresAt := time.Now().UTC().Add(24 * time.Hour)

	created, err := repo.Create(ctx, user.ID, hash, expiresAt)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.RevokeByID(ctx, created.ID); err != nil {
		t.Fatalf("RevokeByID: unexpected error: %v", err)
	}

	// After revocation, GetByHash should return ErrNotFound.
	_, err = repo.GetByHash(ctx, hash)
	assertIsDomainError(t, err, domain.ErrNotFound)
}

func TestRepo_RevokeByID_Idempotent(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	hash := "revoke-idempotent-" + uuid.New().String()[:8]
	expiresAt := time.Now().UTC().Add(24 * time.Hour)

	created, err := repo.Create(ctx, user.ID, hash, expiresAt)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// First revocation.
	if err := repo.RevokeByID(ctx, created.ID); err != nil {
		t.Fatalf("RevokeByID (first): %v", err)
	}

	// Second revocation — should be idempotent, no error.
	if err := repo.RevokeByID(ctx, created.ID); err != nil {
		t.Fatalf("RevokeByID (second): expected no error, got %v", err)
	}
}

func TestRepo_RevokeByID_NonExistent(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	// Revoking a non-existent token should not produce an error (exec, no rows affected).
	if err := repo.RevokeByID(ctx, uuid.New()); err != nil {
		t.Fatalf("RevokeByID non-existent: expected no error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// RevokeAllByUser
// ---------------------------------------------------------------------------

func TestRepo_RevokeAllByUser_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)

	// Create multiple tokens for this user.
	hashes := make([]string, 3)
	for i := range hashes {
		hashes[i] = "revoke-all-" + uuid.New().String()[:8]
		_, err := repo.Create(ctx, user.ID, hashes[i], time.Now().UTC().Add(24*time.Hour))
		if err != nil {
			t.Fatalf("Create token %d: %v", i, err)
		}
	}

	// Verify all tokens are active.
	for i, hash := range hashes {
		if _, err := repo.GetByHash(ctx, hash); err != nil {
			t.Fatalf("GetByHash token %d before revoke: %v", i, err)
		}
	}

	// Revoke all.
	if err := repo.RevokeAllByUser(ctx, user.ID); err != nil {
		t.Fatalf("RevokeAllByUser: unexpected error: %v", err)
	}

	// All tokens should now be not found.
	for i, hash := range hashes {
		_, err := repo.GetByHash(ctx, hash)
		assertIsDomainError(t, err, domain.ErrNotFound)
		_ = i // ensure loop variable used
	}
}

func TestRepo_RevokeAllByUser_OnlyAffectsActiveTokens(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)

	// Create and immediately revoke a token.
	hash1 := "revoke-all-active-1-" + uuid.New().String()[:8]
	created, err := repo.Create(ctx, user.ID, hash1, time.Now().UTC().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("Create token 1: %v", err)
	}
	if err := repo.RevokeByID(ctx, created.ID); err != nil {
		t.Fatalf("RevokeByID: %v", err)
	}

	// Create an active token.
	hash2 := "revoke-all-active-2-" + uuid.New().String()[:8]
	_, err = repo.Create(ctx, user.ID, hash2, time.Now().UTC().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("Create token 2: %v", err)
	}

	// RevokeAllByUser should succeed.
	if err := repo.RevokeAllByUser(ctx, user.ID); err != nil {
		t.Fatalf("RevokeAllByUser: %v", err)
	}

	// Active token should be revoked.
	_, err = repo.GetByHash(ctx, hash2)
	assertIsDomainError(t, err, domain.ErrNotFound)
}

func TestRepo_RevokeAllByUser_DoesNotAffectOtherUsers(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user1 := testhelper.SeedUser(t, pool)
	user2 := testhelper.SeedUser(t, pool)

	hash1 := "other-user-1-" + uuid.New().String()[:8]
	hash2 := "other-user-2-" + uuid.New().String()[:8]

	_, err := repo.Create(ctx, user1.ID, hash1, time.Now().UTC().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("Create token for user1: %v", err)
	}
	_, err = repo.Create(ctx, user2.ID, hash2, time.Now().UTC().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("Create token for user2: %v", err)
	}

	// Revoke all for user1 only.
	if err := repo.RevokeAllByUser(ctx, user1.ID); err != nil {
		t.Fatalf("RevokeAllByUser: %v", err)
	}

	// user1's token is revoked.
	_, err = repo.GetByHash(ctx, hash1)
	assertIsDomainError(t, err, domain.ErrNotFound)

	// user2's token is still active.
	got, err := repo.GetByHash(ctx, hash2)
	if err != nil {
		t.Fatalf("GetByHash user2 token: %v", err)
	}
	if got.UserID != user2.ID {
		t.Errorf("UserID mismatch: got %s, want %s", got.UserID, user2.ID)
	}
}

// ---------------------------------------------------------------------------
// DeleteExpired
// ---------------------------------------------------------------------------

func TestRepo_DeleteExpired_RemovesExpiredTokens(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)

	// Create an expired token directly via SQL (since Create succeeds regardless of expiry).
	expiredHash := "delete-expired-" + uuid.New().String()[:8]
	_, err := repo.Create(ctx, user.ID, expiredHash, time.Now().UTC().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("Create expired token: %v", err)
	}

	// Create an active token.
	activeHash := "delete-active-" + uuid.New().String()[:8]
	_, err = repo.Create(ctx, user.ID, activeHash, time.Now().UTC().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("Create active token: %v", err)
	}

	// Run cleanup.
	_, err = repo.DeleteExpired(ctx)
	if err != nil {
		t.Fatalf("DeleteExpired: unexpected error: %v", err)
	}

	// Active token should still be available.
	got, err := repo.GetByHash(ctx, activeHash)
	if err != nil {
		t.Fatalf("GetByHash active token after cleanup: %v", err)
	}
	if got.TokenHash != activeHash {
		t.Errorf("TokenHash mismatch: got %q, want %q", got.TokenHash, activeHash)
	}

	// Expired token should be deleted (not just not-found by query, but actually gone).
	// Verify via raw query that the row doesn't exist at all.
	var rowCount int
	err = pool.QueryRow(ctx,
		`SELECT count(*) FROM refresh_tokens WHERE token_hash = $1`,
		expiredHash,
	).Scan(&rowCount)
	if err != nil {
		t.Fatalf("count query: %v", err)
	}
	if rowCount != 0 {
		t.Errorf("expected expired token to be deleted, but found %d rows", rowCount)
	}
}

func TestRepo_DeleteExpired_RemovesRevokedTokens(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)

	// Create and revoke a token.
	revokedHash := "delete-revoked-" + uuid.New().String()[:8]
	created, err := repo.Create(ctx, user.ID, revokedHash, time.Now().UTC().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.RevokeByID(ctx, created.ID); err != nil {
		t.Fatalf("RevokeByID: %v", err)
	}

	// Run cleanup.
	_, err = repo.DeleteExpired(ctx)
	if err != nil {
		t.Fatalf("DeleteExpired: %v", err)
	}

	// Revoked token should be physically deleted.
	var rowCount int
	err = pool.QueryRow(ctx,
		`SELECT count(*) FROM refresh_tokens WHERE token_hash = $1`,
		revokedHash,
	).Scan(&rowCount)
	if err != nil {
		t.Fatalf("count query: %v", err)
	}
	if rowCount != 0 {
		t.Errorf("expected revoked token to be deleted, but found %d rows", rowCount)
	}
}

func TestRepo_DeleteExpired_NoOp(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	// DeleteExpired on empty/clean table should not error.
	_, err := repo.DeleteExpired(ctx)
	if err != nil {
		t.Fatalf("DeleteExpired: expected no error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func assertIsDomainError(t *testing.T, err error, target error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error wrapping %v, got nil", target)
	}
	if !errors.Is(err, target) {
		t.Fatalf("expected error wrapping %v, got: %v", target, err)
	}
}
