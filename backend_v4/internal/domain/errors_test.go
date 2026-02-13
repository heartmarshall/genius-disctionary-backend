package domain

import (
	"errors"
	"testing"
)

func TestValidationError_SingleField(t *testing.T) {
	t.Parallel()

	err := NewValidationError("text", "required")

	if got := err.Error(); got != "validation: text — required" {
		t.Fatalf("unexpected Error(): %q", got)
	}
	if !errors.Is(err, ErrValidation) {
		t.Fatal("errors.Is(err, ErrValidation) = false")
	}
}

func TestValidationError_MultipleFields(t *testing.T) {
	t.Parallel()

	err := NewValidationErrors([]FieldError{
		{Field: "text", Message: "required"},
		{Field: "senses", Message: "at least one required"},
	})

	if got := err.Error(); got != "validation: 2 errors" {
		t.Fatalf("unexpected Error(): %q", got)
	}
	if !errors.Is(err, ErrValidation) {
		t.Fatal("errors.Is(err, ErrValidation) = false")
	}
	if len(err.Errors) != 2 {
		t.Fatalf("expected 2 field errors, got %d", len(err.Errors))
	}
}

func TestValidationError_Unwrap(t *testing.T) {
	t.Parallel()

	err := NewValidationError("email", "invalid format")
	if !errors.Is(err, ErrValidation) {
		t.Fatal("Unwrap should return ErrValidation")
	}
}

func TestSentinelErrors_AreDistinct(t *testing.T) {
	t.Parallel()

	sentinels := []error{
		ErrNotFound, ErrAlreadyExists, ErrValidation,
		ErrUnauthorized, ErrForbidden, ErrConflict,
	}
	for i, a := range sentinels {
		for j, b := range sentinels {
			if i != j && errors.Is(a, b) {
				t.Errorf("sentinel errors %d and %d should not match", i, j)
			}
		}
	}
}
