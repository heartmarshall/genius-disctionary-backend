package inbox

import (
	"strings"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// CreateItemInput holds the parameters for creating an inbox item.
type CreateItemInput struct {
	Text    string
	Context *string
}

// Validate checks all fields and collects all errors.
func (i CreateItemInput) Validate() error {
	var errs []domain.FieldError

	text := strings.TrimSpace(i.Text)
	if text == "" {
		errs = append(errs, domain.FieldError{Field: "text", Message: "required"})
	}
	if len(text) > 500 {
		errs = append(errs, domain.FieldError{Field: "text", Message: "max 500 characters"})
	}

	if i.Context != nil {
		ctx := strings.TrimSpace(*i.Context)
		if len(ctx) > 2000 {
			errs = append(errs, domain.FieldError{Field: "context", Message: "max 2000 characters"})
		}
	}

	if len(errs) > 0 {
		return &domain.ValidationError{Errors: errs}
	}
	return nil
}

// ListItemsInput holds the parameters for listing inbox items.
type ListItemsInput struct {
	Limit  int
	Offset int
}

// Validate checks all fields and collects all errors.
func (i ListItemsInput) Validate() error {
	var errs []domain.FieldError
	if i.Limit < 0 {
		errs = append(errs, domain.FieldError{Field: "limit", Message: "must be non-negative"})
	}
	if i.Limit > 200 {
		errs = append(errs, domain.FieldError{Field: "limit", Message: "max 200"})
	}
	if i.Offset < 0 {
		errs = append(errs, domain.FieldError{Field: "offset", Message: "must be non-negative"})
	}
	if len(errs) > 0 {
		return &domain.ValidationError{Errors: errs}
	}
	return nil
}

// DeleteItemInput holds the parameters for deleting an inbox item.
type DeleteItemInput struct {
	ItemID uuid.UUID
}

// Validate checks all fields and collects all errors.
func (i DeleteItemInput) Validate() error {
	if i.ItemID == uuid.Nil {
		return domain.NewValidationError("item_id", "required")
	}
	return nil
}
