// Package authmethod implements the AuthMethod repository using PostgreSQL.
package authmethod

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	postgres "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/authmethod/sqlc"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// Repo provides auth_methods persistence backed by PostgreSQL.
type Repo struct {
	pool *pgxpool.Pool
}

// New creates a new auth method repository.
func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// GetByOAuth returns the auth method for the given OAuth provider + provider ID.
func (r *Repo) GetByOAuth(ctx context.Context, method domain.AuthMethodType, providerID string) (*domain.AuthMethod, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.GetByOAuth(ctx, sqlc.GetByOAuthParams{
		Method:     string(method),
		ProviderID: pgtype.Text{String: providerID, Valid: true},
	})
	if err != nil {
		return nil, mapError(err, "auth_method")
	}

	am := toDomain(row)
	return &am, nil
}

// GetByUserAndMethod returns the auth method for a user with the given method type.
func (r *Repo) GetByUserAndMethod(ctx context.Context, userID uuid.UUID, method domain.AuthMethodType) (*domain.AuthMethod, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.GetByUserAndMethod(ctx, sqlc.GetByUserAndMethodParams{
		UserID: userID,
		Method: string(method),
	})
	if err != nil {
		return nil, mapError(err, "auth_method")
	}

	am := toDomain(row)
	return &am, nil
}

// Create inserts a new auth method row.
func (r *Repo) Create(ctx context.Context, am *domain.AuthMethod) (*domain.AuthMethod, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.CreateAuthMethod(ctx, sqlc.CreateAuthMethodParams{
		UserID:       am.UserID,
		Method:       string(am.Method),
		ProviderID:   ptrStringToPgText(am.ProviderID),
		PasswordHash: ptrStringToPgText(am.PasswordHash),
	})
	if err != nil {
		return nil, mapError(err, "auth_method")
	}

	result := toDomain(row)
	return &result, nil
}

// ListByUser returns all auth methods for a user.
func (r *Repo) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.AuthMethod, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	rows, err := q.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth_method list: %w", err)
	}

	result := make([]domain.AuthMethod, len(rows))
	for i, row := range rows {
		result[i] = toDomain(row)
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Error mapping
// ---------------------------------------------------------------------------

func mapError(err error, entity string) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s: %w", entity, err)
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: %w", entity, domain.ErrNotFound)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return fmt.Errorf("%s: %w", entity, domain.ErrAlreadyExists)
		case "23503": // foreign_key_violation
			return fmt.Errorf("%s: %w", entity, domain.ErrNotFound)
		case "23514": // check_violation
			return fmt.Errorf("%s: %w", entity, domain.ErrValidation)
		}
	}

	return fmt.Errorf("%s: %w", entity, err)
}

// ---------------------------------------------------------------------------
// Mapping helpers
// ---------------------------------------------------------------------------

func toDomain(row sqlc.AuthMethod) domain.AuthMethod {
	return domain.AuthMethod{
		ID:           row.ID,
		UserID:       row.UserID,
		Method:       domain.AuthMethodType(row.Method),
		ProviderID:   pgTextToPtr(row.ProviderID),
		PasswordHash: pgTextToPtr(row.PasswordHash),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

func pgTextToPtr(t pgtype.Text) *string {
	if t.Valid {
		return &t.String
	}
	return nil
}

func ptrStringToPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}
