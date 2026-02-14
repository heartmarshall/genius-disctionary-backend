package postgres

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

func TestMapError_Nil(t *testing.T) {
	t.Parallel()

	got := mapError(nil, "entry", uuid.New())
	if got != nil {
		t.Errorf("mapError(nil) = %v, want nil", got)
	}
}

func TestMapError_NoRows(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	got := mapError(pgx.ErrNoRows, "entry", id)

	if got == nil {
		t.Fatal("mapError(ErrNoRows) = nil, want error")
	}
	if !errors.Is(got, domain.ErrNotFound) {
		t.Errorf("mapError(ErrNoRows) does not wrap domain.ErrNotFound: %v", got)
	}
	if want := fmt.Sprintf("entry %s: not found", id); got.Error() != want {
		t.Errorf("mapError(ErrNoRows).Error() = %q, want %q", got.Error(), want)
	}
}

func TestMapError_WrappedNoRows(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	wrapped := fmt.Errorf("scan row: %w", pgx.ErrNoRows)
	got := mapError(wrapped, "card", id)

	if !errors.Is(got, domain.ErrNotFound) {
		t.Errorf("mapError(wrapped ErrNoRows) does not wrap domain.ErrNotFound: %v", got)
	}
}

func TestMapError_UniqueViolation(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	pgErr := &pgconn.PgError{Code: "23505", Message: "duplicate key"}
	got := mapError(pgErr, "entry", id)

	if !errors.Is(got, domain.ErrAlreadyExists) {
		t.Errorf("mapError(23505) does not wrap domain.ErrAlreadyExists: %v", got)
	}
}

func TestMapError_ForeignKeyViolation(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	pgErr := &pgconn.PgError{Code: "23503", Message: "foreign key violation"}
	got := mapError(pgErr, "entry", id)

	if !errors.Is(got, domain.ErrNotFound) {
		t.Errorf("mapError(23503) does not wrap domain.ErrNotFound: %v", got)
	}
}

func TestMapError_CheckViolation(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	pgErr := &pgconn.PgError{Code: "23514", Message: "check constraint"}
	got := mapError(pgErr, "entry", id)

	if !errors.Is(got, domain.ErrValidation) {
		t.Errorf("mapError(23514) does not wrap domain.ErrValidation: %v", got)
	}
}

func TestMapError_ContextDeadlineExceeded(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	got := mapError(context.DeadlineExceeded, "entry", id)

	if !errors.Is(got, context.DeadlineExceeded) {
		t.Errorf("mapError(DeadlineExceeded) does not wrap context.DeadlineExceeded: %v", got)
	}
	// Must NOT be mapped to a domain error
	if errors.Is(got, domain.ErrNotFound) {
		t.Error("mapError(DeadlineExceeded) should not wrap domain.ErrNotFound")
	}
}

func TestMapError_ContextCanceled(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	got := mapError(context.Canceled, "entry", id)

	if !errors.Is(got, context.Canceled) {
		t.Errorf("mapError(Canceled) does not wrap context.Canceled: %v", got)
	}
	// Must NOT be mapped to a domain error
	if errors.Is(got, domain.ErrNotFound) {
		t.Error("mapError(Canceled) should not wrap domain.ErrNotFound")
	}
}

func TestMapError_UnknownError(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	original := errors.New("something unexpected")
	got := mapError(original, "entry", id)

	if !errors.Is(got, original) {
		t.Errorf("mapError(unknown) does not wrap original error: %v", got)
	}
	if want := fmt.Sprintf("entry %s: something unexpected", id); got.Error() != want {
		t.Errorf("mapError(unknown).Error() = %q, want %q", got.Error(), want)
	}
}

func TestMapError_UnknownPgError(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	pgErr := &pgconn.PgError{Code: "42P01", Message: "relation does not exist"}
	got := mapError(pgErr, "entry", id)

	// Unknown PG codes should pass through, not be mapped to domain errors
	var unwrapped *pgconn.PgError
	if !errors.As(got, &unwrapped) {
		t.Errorf("mapError(unknown PgError) does not wrap *pgconn.PgError: %v", got)
	}
	if errors.Is(got, domain.ErrNotFound) || errors.Is(got, domain.ErrAlreadyExists) || errors.Is(got, domain.ErrValidation) {
		t.Error("mapError(unknown PgError) should not map to a domain error")
	}
}

func TestMapError_WrappedPgError(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	pgErr := &pgconn.PgError{Code: "23505", Message: "duplicate key"}
	wrapped := fmt.Errorf("insert row: %w", pgErr)
	got := mapError(wrapped, "entry", id)

	if !errors.Is(got, domain.ErrAlreadyExists) {
		t.Errorf("mapError(wrapped 23505) does not wrap domain.ErrAlreadyExists: %v", got)
	}
}

func TestMapError_EntityAndIDInMessage(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	got := mapError(pgx.ErrNoRows, "dictionary_entry", id)

	wantPrefix := fmt.Sprintf("dictionary_entry %s:", id)
	if len(got.Error()) < len(wantPrefix) || got.Error()[:len(wantPrefix)] != wantPrefix {
		t.Errorf("mapError message should start with %q, got %q", wantPrefix, got.Error())
	}
}

func TestMapError_AllPgCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		code     string
		wantErr  error
		wantName string
	}{
		{"unique_violation", "23505", domain.ErrAlreadyExists, "ErrAlreadyExists"},
		{"foreign_key_violation", "23503", domain.ErrNotFound, "ErrNotFound"},
		{"check_violation", "23514", domain.ErrValidation, "ErrValidation"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id := uuid.New()
			pgErr := &pgconn.PgError{Code: tt.code}
			got := mapError(pgErr, "entry", id)

			if !errors.Is(got, tt.wantErr) {
				t.Errorf("mapError(code %s) does not wrap %s: %v", tt.code, tt.wantName, got)
			}
		})
	}
}
