package study

import (
	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// GetQueueInput holds the parameters for fetching the study queue.
type GetQueueInput struct {
	Limit int
}

// Validate checks all fields and collects all errors.
func (i *GetQueueInput) Validate() error {
	var errs []domain.FieldError

	if i.Limit < 0 || i.Limit > 200 {
		errs = append(errs, domain.FieldError{Field: "limit", Message: "must be between 0 and 200"})
	}

	if len(errs) > 0 {
		return domain.NewValidationErrors(errs)
	}
	return nil
}

// ReviewCardInput holds the parameters for reviewing a card.
type ReviewCardInput struct {
	CardID     uuid.UUID
	Grade      domain.ReviewGrade
	DurationMs *int
	SessionID  *uuid.UUID
}

// Validate checks all fields and collects all errors.
func (i *ReviewCardInput) Validate() error {
	var errs []domain.FieldError

	if i.CardID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "card_id", Message: "required"})
	}
	if !i.Grade.IsValid() {
		errs = append(errs, domain.FieldError{Field: "grade", Message: "must be AGAIN, HARD, GOOD, or EASY"})
	}
	// Only validate DurationMs if it's provided (not nil)
	if i.DurationMs != nil && *i.DurationMs < 0 {
		errs = append(errs, domain.FieldError{Field: "duration_ms", Message: "must be non-negative"})
	}
	if i.DurationMs != nil && *i.DurationMs > 600_000 {
		errs = append(errs, domain.FieldError{Field: "duration_ms", Message: "max 10 minutes"})
	}
	// No validation for SessionID - it's optional and can be nil

	if len(errs) > 0 {
		return domain.NewValidationErrors(errs)
	}
	return nil
}

// UndoReviewInput holds the parameters for undoing a review.
type UndoReviewInput struct {
	CardID uuid.UUID
}

// Validate checks all fields and collects all errors.
func (i *UndoReviewInput) Validate() error {
	var errs []domain.FieldError

	if i.CardID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "card_id", Message: "required"})
	}

	if len(errs) > 0 {
		return domain.NewValidationErrors(errs)
	}
	return nil
}

// CreateCardInput holds the parameters for creating a card.
type CreateCardInput struct {
	EntryID uuid.UUID
}

// Validate checks all fields and collects all errors.
func (i *CreateCardInput) Validate() error {
	var errs []domain.FieldError

	if i.EntryID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "entry_id", Message: "required"})
	}

	if len(errs) > 0 {
		return domain.NewValidationErrors(errs)
	}
	return nil
}

// DeleteCardInput holds the parameters for deleting a card.
type DeleteCardInput struct {
	CardID uuid.UUID
}

// Validate checks all fields and collects all errors.
func (i *DeleteCardInput) Validate() error {
	var errs []domain.FieldError

	if i.CardID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "card_id", Message: "required"})
	}

	if len(errs) > 0 {
		return domain.NewValidationErrors(errs)
	}
	return nil
}

// GetCardHistoryInput holds the parameters for fetching card review history.
type GetCardHistoryInput struct {
	CardID uuid.UUID
	Limit  int
	Offset int
}

// Validate checks all fields and collects all errors.
func (i *GetCardHistoryInput) Validate() error {
	var errs []domain.FieldError

	if i.CardID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "card_id", Message: "required"})
	}
	if i.Limit < 0 || i.Limit > 200 {
		errs = append(errs, domain.FieldError{Field: "limit", Message: "must be between 0 and 200"})
	}
	if i.Offset < 0 {
		errs = append(errs, domain.FieldError{Field: "offset", Message: "must be >= 0"})
	}

	if len(errs) > 0 {
		return domain.NewValidationErrors(errs)
	}
	return nil
}

// BatchCreateCardsInput holds the parameters for batch-creating cards.
type BatchCreateCardsInput struct {
	EntryIDs []uuid.UUID
}

// Validate checks all fields and collects all errors.
func (i *BatchCreateCardsInput) Validate() error {
	var errs []domain.FieldError

	if len(i.EntryIDs) == 0 {
		errs = append(errs, domain.FieldError{Field: "entry_ids", Message: "required (at least 1)"})
	} else if len(i.EntryIDs) > 100 {
		errs = append(errs, domain.FieldError{Field: "entry_ids", Message: "too many (max 100)"})
	}

	if len(errs) > 0 {
		return domain.NewValidationErrors(errs)
	}
	return nil
}

// FinishSessionInput holds the parameters for finishing a study session.
type FinishSessionInput struct {
	SessionID uuid.UUID
}

// Validate checks all fields and collects all errors.
func (i *FinishSessionInput) Validate() error {
	var errs []domain.FieldError

	if i.SessionID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "session_id", Message: "required"})
	}

	if len(errs) > 0 {
		return domain.NewValidationErrors(errs)
	}
	return nil
}
