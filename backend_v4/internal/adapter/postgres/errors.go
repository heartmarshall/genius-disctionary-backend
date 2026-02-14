package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// mapError converts pgx/pgconn errors to domain errors.
// context.DeadlineExceeded and context.Canceled are NOT mapped — they pass through.
func mapError(err error, entity string, id uuid.UUID) error {
	if err == nil {
		return nil
	}

	// context errors pass through as-is
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s %s: %w", entity, id, err)
	}

	// pgx.ErrNoRows → domain.ErrNotFound
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
