package dictionary

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/service/types"
)

// parseEntryID парсит строку в UUID.
func parseEntryID(idStr string) (uuid.UUID, error) {
	if idStr == "" {
		return uuid.Nil, types.NewValidationError("id", "cannot be empty")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, types.NewValidationError("id", fmt.Sprintf("invalid UUID format: %v", err))
	}

	return id, nil
}

// wrapServiceError оборачивает ошибку с контекстом операции, сохраняя типизированные ошибки.
// Типизированные ошибки (ValidationError, ErrNotFound, ErrAlreadyExists) не оборачиваются.
func wrapServiceError(err error, operation string) error {
	if err == nil {
		return nil
	}

	// Сохраняем типизированные ошибки без обертки
	if errors.Is(err, types.ErrNotFound) ||
		errors.Is(err, types.ErrAlreadyExists) ||
		errors.Is(err, types.ErrInvalidInput) ||
		types.IsValidationError(err) {
		return err
	}

	return fmt.Errorf("%s: %w", operation, err)
}
