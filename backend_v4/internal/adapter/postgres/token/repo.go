// Package token implements the RefreshToken repository using PostgreSQL.
package token

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	postgres "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/token/sqlc"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// Repo provides refresh-token persistence backed by PostgreSQL.
type Repo struct {
	pool *pgxpool.Pool
}

// New creates a new token repository.
func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// Create inserts a new refresh token.
func (r *Repo) Create(ctx context.Context, token *domain.RefreshToken) error {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	_, err := q.CreateRefreshToken(ctx, sqlc.CreateRefreshTokenParams{
		UserID:    token.UserID,
		TokenHash: token.TokenHash,
		ExpiresAt: token.ExpiresAt,
	})
	if err != nil {
		return mapError(err, "refresh_token", uuid.Nil)
	}

	return nil
}

// GetByHash returns an active (non-revoked, non-expired) refresh token by its hash.
// Returns domain.ErrNotFound if the token does not exist, is revoked, or is expired.
func (r *Repo) GetByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, mapError(err, "refresh_token", uuid.Nil)
	}

	t := toDomain(row)
	return &t, nil
}

// RevokeByID revokes a specific refresh token by setting revoked_at.
// Idempotent: revoking an already-revoked token is not an error.
func (r *Repo) RevokeByID(ctx context.Context, id uuid.UUID) error {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	err := q.RevokeRefreshTokenByID(ctx, id)
	if err != nil {
		return mapError(err, "refresh_token", id)
	}

	return nil
}

// RevokeAllByUser revokes all active refresh tokens for the given user.
func (r *Repo) RevokeAllByUser(ctx context.Context, userID uuid.UUID) error {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	err := q.RevokeAllRefreshTokensByUser(ctx, userID)
	if err != nil {
		return mapError(err, "refresh_token", uuid.Nil)
	}

	return nil
}

// DeleteExpired removes all expired or revoked tokens from the database.
// Returns the count of deleted tokens.
// May delete many records; does not use a transaction.
func (r *Repo) DeleteExpired(ctx context.Context) (int, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	tag, err := q.DeleteExpiredRefreshTokens(ctx)
	if err != nil {
		return 0, mapError(err, "refresh_token", uuid.Nil)
	}

	return int(tag.RowsAffected()), nil
}

// ---------------------------------------------------------------------------
// Error mapping
// ---------------------------------------------------------------------------

// mapError converts pgx/pgconn errors into domain errors.
func mapError(err error, entity string, id uuid.UUID) error {
	if err == nil {
		return nil
	}

	// context errors pass through as-is
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s %s: %w", entity, id, err)
	}

	// pgx.ErrNoRows -> domain.ErrNotFound
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s %s: %w", entity, id, domain.ErrNotFound)
	}

	// PgError codes
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return fmt.Errorf("%s %s: %w", entity, id, domain.ErrAlreadyExists)
		case "23503": // foreign_key_violation
			return fmt.Errorf("%s %s: %w", entity, id, domain.ErrNotFound)
		case "23514": // check_violation
			return fmt.Errorf("%s %s: %w", entity, id, domain.ErrValidation)
		}
	}

	// Everything else: wrap with context
	return fmt.Errorf("%s %s: %w", entity, id, err)
}

// ---------------------------------------------------------------------------
// Mapping helpers: sqlc → domain
// ---------------------------------------------------------------------------

// toDomain converts a sqlc.RefreshToken row into a domain.RefreshToken.
func toDomain(row sqlc.RefreshToken) domain.RefreshToken {
	return domain.RefreshToken{
		ID:        row.ID,
		UserID:    row.UserID,
		TokenHash: row.TokenHash,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
		RevokedAt: row.RevokedAt,
	}
}
