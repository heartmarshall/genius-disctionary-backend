package graphql

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

func TestErrorPresenter_NotFound(t *testing.T) {
	log := slog.Default()
	presenter := NewErrorPresenter(log)

	err := domain.ErrNotFound
	ctx := context.Background()

	gqlErr := presenter(ctx, err)

	if gqlErr.Extensions == nil {
		t.Fatal("expected extensions, got nil")
	}
	code, ok := gqlErr.Extensions["code"]
	if !ok {
		t.Fatal("expected code in extensions")
	}
	if code != "NOT_FOUND" {
		t.Errorf("expected code NOT_FOUND, got %v", code)
	}
}

func TestErrorPresenter_AlreadyExists(t *testing.T) {
	log := slog.Default()
	presenter := NewErrorPresenter(log)

	err := domain.ErrAlreadyExists
	ctx := context.Background()

	gqlErr := presenter(ctx, err)

	if gqlErr.Extensions == nil {
		t.Fatal("expected extensions, got nil")
	}
	code, ok := gqlErr.Extensions["code"]
	if !ok {
		t.Fatal("expected code in extensions")
	}
	if code != "ALREADY_EXISTS" {
		t.Errorf("expected code ALREADY_EXISTS, got %v", code)
	}
}

func TestErrorPresenter_Validation(t *testing.T) {
	log := slog.Default()
	presenter := NewErrorPresenter(log)

	err := domain.NewValidationErrors([]domain.FieldError{
		{Field: "text", Message: "required"},
		{Field: "senses", Message: "at least one sense required"},
	})
	ctx := context.Background()

	gqlErr := presenter(ctx, err)

	if gqlErr.Extensions == nil {
		t.Fatal("expected extensions, got nil")
	}
	code, ok := gqlErr.Extensions["code"]
	if !ok {
		t.Fatal("expected code in extensions")
	}
	if code != "VALIDATION" {
		t.Errorf("expected code VALIDATION, got %v", code)
	}

	fields, ok := gqlErr.Extensions["fields"]
	if !ok {
		t.Fatal("expected fields in extensions")
	}
	fieldErrors, ok := fields.([]domain.FieldError)
	if !ok {
		t.Fatalf("expected fields to be []FieldError, got %T", fields)
	}
	if len(fieldErrors) != 2 {
		t.Errorf("expected 2 field errors, got %d", len(fieldErrors))
	}
}

func TestErrorPresenter_ValidationSingleField(t *testing.T) {
	log := slog.Default()
	presenter := NewErrorPresenter(log)

	err := domain.NewValidationError("text", "required")
	ctx := context.Background()

	gqlErr := presenter(ctx, err)

	if gqlErr.Extensions == nil {
		t.Fatal("expected extensions, got nil")
	}
	code, ok := gqlErr.Extensions["code"]
	if !ok {
		t.Fatal("expected code in extensions")
	}
	if code != "VALIDATION" {
		t.Errorf("expected code VALIDATION, got %v", code)
	}

	fields, ok := gqlErr.Extensions["fields"]
	if !ok {
		t.Fatal("expected fields in extensions")
	}
	fieldErrors, ok := fields.([]domain.FieldError)
	if !ok {
		t.Fatalf("expected fields to be []FieldError, got %T", fields)
	}
	if len(fieldErrors) != 1 {
		t.Errorf("expected 1 field error, got %d", len(fieldErrors))
	}
	if fieldErrors[0].Field != "text" {
		t.Errorf("expected field 'text', got %s", fieldErrors[0].Field)
	}
}

func TestErrorPresenter_Unauthorized(t *testing.T) {
	log := slog.Default()
	presenter := NewErrorPresenter(log)

	err := domain.ErrUnauthorized
	ctx := context.Background()

	gqlErr := presenter(ctx, err)

	if gqlErr.Extensions == nil {
		t.Fatal("expected extensions, got nil")
	}
	code, ok := gqlErr.Extensions["code"]
	if !ok {
		t.Fatal("expected code in extensions")
	}
	if code != "UNAUTHENTICATED" {
		t.Errorf("expected code UNAUTHENTICATED, got %v", code)
	}
}

func TestErrorPresenter_Forbidden(t *testing.T) {
	log := slog.Default()
	presenter := NewErrorPresenter(log)

	err := domain.ErrForbidden
	ctx := context.Background()

	gqlErr := presenter(ctx, err)

	if gqlErr.Extensions == nil {
		t.Fatal("expected extensions, got nil")
	}
	code, ok := gqlErr.Extensions["code"]
	if !ok {
		t.Fatal("expected code in extensions")
	}
	if code != "FORBIDDEN" {
		t.Errorf("expected code FORBIDDEN, got %v", code)
	}
}

func TestErrorPresenter_Conflict(t *testing.T) {
	log := slog.Default()
	presenter := NewErrorPresenter(log)

	err := domain.ErrConflict
	ctx := context.Background()

	gqlErr := presenter(ctx, err)

	if gqlErr.Extensions == nil {
		t.Fatal("expected extensions, got nil")
	}
	code, ok := gqlErr.Extensions["code"]
	if !ok {
		t.Fatal("expected code in extensions")
	}
	if code != "CONFLICT" {
		t.Errorf("expected code CONFLICT, got %v", code)
	}
}

func TestErrorPresenter_WrappedError(t *testing.T) {
	log := slog.Default()
	presenter := NewErrorPresenter(log)

	err := fmt.Errorf("op: %w", domain.ErrNotFound)
	ctx := context.Background()

	gqlErr := presenter(ctx, err)

	if gqlErr.Extensions == nil {
		t.Fatal("expected extensions, got nil")
	}
	code, ok := gqlErr.Extensions["code"]
	if !ok {
		t.Fatal("expected code in extensions")
	}
	if code != "NOT_FOUND" {
		t.Errorf("expected code NOT_FOUND (unwrap should work), got %v", code)
	}
}

func TestErrorPresenter_UnexpectedError(t *testing.T) {
	log := slog.Default()
	presenter := NewErrorPresenter(log)

	err := errors.New("unexpected database error")
	ctx := ctxutil.WithRequestID(context.Background(), "test-request-123")

	gqlErr := presenter(ctx, err)

	if gqlErr.Extensions == nil {
		t.Fatal("expected extensions, got nil")
	}
	code, ok := gqlErr.Extensions["code"]
	if !ok {
		t.Fatal("expected code in extensions")
	}
	if code != "INTERNAL" {
		t.Errorf("expected code INTERNAL, got %v", code)
	}
	if gqlErr.Message != "internal error" {
		t.Errorf("expected message 'internal error', got %s", gqlErr.Message)
	}
}

func TestErrorPresenter_UnexpectedError_NoLeakDetails(t *testing.T) {
	log := slog.Default()
	presenter := NewErrorPresenter(log)

	// Error with sensitive details
	err := errors.New("database connection string: postgres://user:password@host/db failed")
	ctx := context.Background()

	gqlErr := presenter(ctx, err)

	// Client should only see "internal error", not the original message
	if gqlErr.Message != "internal error" {
		t.Errorf("expected generic 'internal error', but got: %s (details leaked!)", gqlErr.Message)
	}

	// Verify we don't leak details anywhere in extensions either
	if details, ok := gqlErr.Extensions["details"]; ok {
		t.Errorf("unexpected details in extensions: %v (should not leak error details)", details)
	}
}
