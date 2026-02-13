package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors used across all layers.
var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrValidation    = errors.New("validation error")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrConflict      = errors.New("conflict")
)

// FieldError describes a validation error for a specific field.
type FieldError struct {
	Field   string
	Message string
}

// ValidationError contains a list of field-level validation errors.
type ValidationError struct {
	Errors []FieldError
}

func (e *ValidationError) Error() string {
	if len(e.Errors) == 1 {
		return fmt.Sprintf("validation: %s — %s", e.Errors[0].Field, e.Errors[0].Message)
	}
	return fmt.Sprintf("validation: %d errors", len(e.Errors))
}

func (e *ValidationError) Unwrap() error { return ErrValidation }

// NewValidationError creates a ValidationError for a single field.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Errors: []FieldError{{Field: field, Message: message}},
	}
}

// NewValidationErrors creates a ValidationError from multiple field errors.
func NewValidationErrors(errs []FieldError) *ValidationError {
	return &ValidationError{Errors: errs}
}
