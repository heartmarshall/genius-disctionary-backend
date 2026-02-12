package card

import (
	"fmt"

	"github.com/heartmarshall/my-english/internal/service/types"
)

const (
	// MinEaseFactor — минимальное значение ease factor.
	MinEaseFactor = 1.3
	// MaxEaseFactor — максимальное значение ease factor.
	MaxEaseFactor = 3.0
	// MinIntervalDays — минимальное значение интервала в днях.
	MinIntervalDays = 0
	// MaxIntervalDays — максимальное значение интервала в днях.
	MaxIntervalDays = 3650 // 10 лет
)

// validateCreateCardInput валидирует входные данные для создания карточки.
func validateCreateCardInput(input CreateCardInput) error {
	if input.EntryID == "" {
		return types.NewValidationError("entryID", "cannot be empty")
	}

	if input.Status != nil && !input.Status.IsValid() {
		return types.NewValidationError("status", fmt.Sprintf("invalid status: %s", *input.Status))
	}

	if input.IntervalDays != nil {
		if *input.IntervalDays < MinIntervalDays {
			return types.NewValidationError("intervalDays", fmt.Sprintf("cannot be less than %d", MinIntervalDays))
		}
		if *input.IntervalDays > MaxIntervalDays {
			return types.NewValidationError("intervalDays", fmt.Sprintf("cannot be greater than %d", MaxIntervalDays))
		}
	}

	if input.EaseFactor != nil {
		if *input.EaseFactor < MinEaseFactor {
			return types.NewValidationError("easeFactor", fmt.Sprintf("cannot be less than %.1f", MinEaseFactor))
		}
		if *input.EaseFactor > MaxEaseFactor {
			return types.NewValidationError("easeFactor", fmt.Sprintf("cannot be greater than %.1f", MaxEaseFactor))
		}
	}

	return nil
}

// validateUpdateCardInput валидирует входные данные для обновления карточки.
func validateUpdateCardInput(input UpdateCardInput) error {
	if input.ID == "" {
		return types.NewValidationError("id", "cannot be empty")
	}

	if input.Status != nil && !input.Status.IsValid() {
		return types.NewValidationError("status", fmt.Sprintf("invalid status: %s", *input.Status))
	}

	if input.IntervalDays != nil {
		if *input.IntervalDays < MinIntervalDays {
			return types.NewValidationError("intervalDays", fmt.Sprintf("cannot be less than %d", MinIntervalDays))
		}
		if *input.IntervalDays > MaxIntervalDays {
			return types.NewValidationError("intervalDays", fmt.Sprintf("cannot be greater than %d", MaxIntervalDays))
		}
	}

	if input.EaseFactor != nil {
		if *input.EaseFactor < MinEaseFactor {
			return types.NewValidationError("easeFactor", fmt.Sprintf("cannot be less than %.1f", MinEaseFactor))
		}
		if *input.EaseFactor > MaxEaseFactor {
			return types.NewValidationError("easeFactor", fmt.Sprintf("cannot be greater than %.1f", MaxEaseFactor))
		}
	}

	return nil
}
