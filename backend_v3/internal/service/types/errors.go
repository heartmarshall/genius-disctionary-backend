package types

import (
	"errors"
	"fmt"
)

var (
	// ErrNotFound возвращается, когда сущность не найдена.
	ErrNotFound = errors.New("entity not found")

	// ErrAlreadyExists возвращается, когда сущность уже существует.
	ErrAlreadyExists = errors.New("entity already exists")

	// ErrInvalidInput возвращается при невалидных входных данных.
	ErrInvalidInput = errors.New("invalid input data")

	// ErrInternal возвращается при внутренних ошибках сервиса.
	ErrInternal = errors.New("internal error")
)

// ValidationError представляет ошибку валидации с указанием поля.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error: field %q: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// NewValidationError создает новую ошибку валидации.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// IsValidationError проверяет, является ли ошибка ошибкой валидации.
func IsValidationError(err error) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
}
