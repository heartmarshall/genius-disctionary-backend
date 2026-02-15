package topic

import (
	"strings"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// CreateTopicInput holds the parameters for creating a topic.
type CreateTopicInput struct {
	Name        string
	Description *string
}

// Validate checks all fields and collects all errors.
func (i CreateTopicInput) Validate() error {
	var errs []domain.FieldError

	name := strings.TrimSpace(i.Name)
	if name == "" {
		errs = append(errs, domain.FieldError{Field: "name", Message: "required"})
	}
	if len(name) > 100 {
		errs = append(errs, domain.FieldError{Field: "name", Message: "max 100 characters"})
	}

	if i.Description != nil && len(strings.TrimSpace(*i.Description)) > 500 {
		errs = append(errs, domain.FieldError{Field: "description", Message: "max 500 characters"})
	}

	if len(errs) > 0 {
		return &domain.ValidationError{Errors: errs}
	}
	return nil
}

// UpdateTopicInput holds the parameters for updating a topic.
type UpdateTopicInput struct {
	TopicID     uuid.UUID
	Name        *string
	Description *string // nil = don't change; ptr("") = clear
}

// Validate checks all fields and collects all errors.
func (i UpdateTopicInput) Validate() error {
	var errs []domain.FieldError

	if i.TopicID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "topic_id", Message: "required"})
	}
	if i.Name == nil && i.Description == nil {
		errs = append(errs, domain.FieldError{Field: "input", Message: "at least one field must be provided"})
	}
	if i.Name != nil {
		name := strings.TrimSpace(*i.Name)
		if name == "" {
			errs = append(errs, domain.FieldError{Field: "name", Message: "required"})
		}
		if len(name) > 100 {
			errs = append(errs, domain.FieldError{Field: "name", Message: "max 100 characters"})
		}
	}
	if i.Description != nil && len(*i.Description) > 500 {
		errs = append(errs, domain.FieldError{Field: "description", Message: "max 500 characters"})
	}

	if len(errs) > 0 {
		return &domain.ValidationError{Errors: errs}
	}
	return nil
}

// DeleteTopicInput holds the parameters for deleting a topic.
type DeleteTopicInput struct {
	TopicID uuid.UUID
}

// Validate checks all fields and collects all errors.
func (i DeleteTopicInput) Validate() error {
	if i.TopicID == uuid.Nil {
		return domain.NewValidationError("topic_id", "required")
	}
	return nil
}

// LinkEntryInput holds the parameters for linking an entry to a topic.
type LinkEntryInput struct {
	TopicID uuid.UUID
	EntryID uuid.UUID
}

// Validate checks all fields and collects all errors.
func (i LinkEntryInput) Validate() error {
	var errs []domain.FieldError
	if i.TopicID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "topic_id", Message: "required"})
	}
	if i.EntryID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "entry_id", Message: "required"})
	}
	if len(errs) > 0 {
		return &domain.ValidationError{Errors: errs}
	}
	return nil
}

// UnlinkEntryInput holds the parameters for unlinking an entry from a topic.
type UnlinkEntryInput struct {
	TopicID uuid.UUID
	EntryID uuid.UUID
}

// Validate checks all fields and collects all errors.
func (i UnlinkEntryInput) Validate() error {
	var errs []domain.FieldError
	if i.TopicID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "topic_id", Message: "required"})
	}
	if i.EntryID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "entry_id", Message: "required"})
	}
	if len(errs) > 0 {
		return &domain.ValidationError{Errors: errs}
	}
	return nil
}

// BatchLinkEntriesInput holds the parameters for batch linking entries to a topic.
type BatchLinkEntriesInput struct {
	TopicID  uuid.UUID
	EntryIDs []uuid.UUID
}

// Validate checks all fields and collects all errors.
func (i BatchLinkEntriesInput) Validate() error {
	var errs []domain.FieldError
	if i.TopicID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "topic_id", Message: "required"})
	}
	if len(i.EntryIDs) == 0 {
		errs = append(errs, domain.FieldError{Field: "entry_ids", Message: "at least one entry required"})
	}
	if len(i.EntryIDs) > 200 {
		errs = append(errs, domain.FieldError{Field: "entry_ids", Message: "max 200 entries per batch"})
	}
	if len(errs) > 0 {
		return &domain.ValidationError{Errors: errs}
	}
	return nil
}
