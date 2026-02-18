package dictionary

import (
	"strconv"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// CreateFromCatalogInput holds the parameters for creating an entry from the reference catalog.
type CreateFromCatalogInput struct {
	RefEntryID uuid.UUID
	SenseIDs   []uuid.UUID
	CreateCard bool
	Notes      *string
}

// Validate checks all fields and collects all errors.
func (i *CreateFromCatalogInput) Validate() error {
	var errs []domain.FieldError

	if i.RefEntryID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "ref_entry_id", Message: "required"})
	}
	if len(i.SenseIDs) > 20 {
		errs = append(errs, domain.FieldError{Field: "sense_ids", Message: "too many (max 20)"})
	}
	if i.Notes != nil && len(*i.Notes) > 5000 {
		errs = append(errs, domain.FieldError{Field: "notes", Message: "too long (max 5000)"})
	}

	if len(errs) > 0 {
		return domain.NewValidationErrors(errs)
	}
	return nil
}

// CreateCustomInput holds the parameters for creating a custom entry.
type CreateCustomInput struct {
	Text       string
	Senses     []SenseInput
	CreateCard bool
	Notes      *string
	TopicID    *uuid.UUID
}

// SenseInput holds the parameters for a single sense in a custom entry.
type SenseInput struct {
	Definition   *string
	PartOfSpeech *domain.PartOfSpeech
	Translations []string
	Examples     []ExampleInput
}

// ExampleInput holds the parameters for a single example in a custom sense.
type ExampleInput struct {
	Sentence    string
	Translation *string
}

// Validate checks all fields and collects all errors.
func (i *CreateCustomInput) Validate() error {
	var errs []domain.FieldError

	if i.Text == "" {
		errs = append(errs, domain.FieldError{Field: "text", Message: "required"})
	} else if len(i.Text) > 500 {
		errs = append(errs, domain.FieldError{Field: "text", Message: "too long (max 500)"})
	}

	if len(i.Senses) > 20 {
		errs = append(errs, domain.FieldError{Field: "senses", Message: "too many (max 20)"})
	}

	for si, sense := range i.Senses {
		if sense.Definition != nil && len(*sense.Definition) > 2000 {
			errs = append(errs, domain.FieldError{
				Field:   fieldIndex("senses", si, "definition"),
				Message: "too long (max 2000)",
			})
		}
		if sense.PartOfSpeech != nil && !sense.PartOfSpeech.IsValid() {
			errs = append(errs, domain.FieldError{
				Field:   fieldIndex("senses", si, "part_of_speech"),
				Message: "invalid value",
			})
		}
		if len(sense.Translations) > 20 {
			errs = append(errs, domain.FieldError{
				Field:   fieldIndex("senses", si, "translations"),
				Message: "too many (max 20)",
			})
		}
		for ti, tr := range sense.Translations {
			if tr == "" {
				errs = append(errs, domain.FieldError{
					Field:   fieldIndex2("senses", si, "translations", ti),
					Message: "required",
				})
			} else if len(tr) > 500 {
				errs = append(errs, domain.FieldError{
					Field:   fieldIndex2("senses", si, "translations", ti),
					Message: "too long (max 500)",
				})
			}
		}
		if len(sense.Examples) > 20 {
			errs = append(errs, domain.FieldError{
				Field:   fieldIndex("senses", si, "examples"),
				Message: "too many (max 20)",
			})
		}
		for ei, ex := range sense.Examples {
			if ex.Sentence == "" {
				errs = append(errs, domain.FieldError{
					Field:   fieldIndex2("senses", si, "examples", ei),
					Message: "sentence required",
				})
			} else if len(ex.Sentence) > 2000 {
				errs = append(errs, domain.FieldError{
					Field:   fieldIndex2("senses", si, "examples", ei),
					Message: "sentence too long (max 2000)",
				})
			}
			if ex.Translation != nil && len(*ex.Translation) > 2000 {
				errs = append(errs, domain.FieldError{
					Field:   fieldIndex2("senses", si, "examples", ei) + ".translation",
					Message: "too long (max 2000)",
				})
			}
		}
	}

	if i.Notes != nil && len(*i.Notes) > 5000 {
		errs = append(errs, domain.FieldError{Field: "notes", Message: "too long (max 5000)"})
	}

	if len(errs) > 0 {
		return domain.NewValidationErrors(errs)
	}
	return nil
}

// FindInput holds the parameters for searching entries.
type FindInput struct {
	Search       *string
	HasCard      *bool
	PartOfSpeech *domain.PartOfSpeech
	TopicID      *uuid.UUID
	Status       *domain.LearningStatus
	SortBy       string
	SortOrder    string
	Limit        int
	Cursor       *string
	Offset       *int
}

// Validate checks all fields and collects all errors.
func (i *FindInput) Validate() error {
	var errs []domain.FieldError

	if i.SortBy != "" {
		switch i.SortBy {
		case "text", "created_at", "updated_at":
			// valid
		default:
			errs = append(errs, domain.FieldError{Field: "sort_by", Message: "invalid value (allowed: text, created_at, updated_at)"})
		}
	}

	if i.SortOrder != "" {
		switch i.SortOrder {
		case "ASC", "DESC":
			// valid
		default:
			errs = append(errs, domain.FieldError{Field: "sort_order", Message: "invalid value (allowed: ASC, DESC)"})
		}
	}

	if i.PartOfSpeech != nil && !i.PartOfSpeech.IsValid() {
		errs = append(errs, domain.FieldError{Field: "part_of_speech", Message: "invalid value"})
	}

	if i.Status != nil && !i.Status.IsValid() {
		errs = append(errs, domain.FieldError{Field: "status", Message: "invalid value"})
	}

	if len(errs) > 0 {
		return domain.NewValidationErrors(errs)
	}
	return nil
}

// UpdateNotesInput holds the parameters for updating entry notes.
type UpdateNotesInput struct {
	EntryID uuid.UUID
	Notes   *string
}

// Validate checks all fields and collects all errors.
func (i *UpdateNotesInput) Validate() error {
	var errs []domain.FieldError

	if i.EntryID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "entry_id", Message: "required"})
	}
	if i.Notes != nil && len(*i.Notes) > 5000 {
		errs = append(errs, domain.FieldError{Field: "notes", Message: "too long (max 5000)"})
	}

	if len(errs) > 0 {
		return domain.NewValidationErrors(errs)
	}
	return nil
}

// ImportInput holds the parameters for importing entries.
type ImportInput struct {
	Items []ImportItem
}

// ImportItem represents a single item to import.
type ImportItem struct {
	Text         string
	Translations []string
	Notes        *string
	TopicName    *string // ignored in MVP
}

// Validate checks all fields and collects all errors.
func (i *ImportInput) Validate() error {
	var errs []domain.FieldError

	if len(i.Items) == 0 {
		errs = append(errs, domain.FieldError{Field: "items", Message: "required (at least 1)"})
	} else if len(i.Items) > 5000 {
		errs = append(errs, domain.FieldError{Field: "items", Message: "too many (max 5000)"})
	}

	for idx, item := range i.Items {
		if item.Text == "" {
			errs = append(errs, domain.FieldError{
				Field:   fieldIdx("items", idx, "text"),
				Message: "required",
			})
		} else if len(item.Text) > 500 {
			errs = append(errs, domain.FieldError{
				Field:   fieldIdx("items", idx, "text"),
				Message: "too long (max 500)",
			})
		}
		if len(item.Translations) > 20 {
			errs = append(errs, domain.FieldError{
				Field:   fieldIdx("items", idx, "translations"),
				Message: "too many (max 20)",
			})
		}
	}

	if len(errs) > 0 {
		return domain.NewValidationErrors(errs)
	}
	return nil
}

// fieldIndex formats a nested field path like "senses[0].definition".
func fieldIndex(parent string, idx int, field string) string {
	return parent + "[" + strconv.Itoa(idx) + "]." + field
}

// fieldIndex2 formats a deeply nested field path like "senses[0].translations[1]".
func fieldIndex2(parent string, idx int, child string, childIdx int) string {
	return parent + "[" + strconv.Itoa(idx) + "]." + child + "[" + strconv.Itoa(childIdx) + "]"
}

// fieldIdx formats a nested field path like "items[0].text".
func fieldIdx(parent string, idx int, field string) string {
	return parent + "[" + strconv.Itoa(idx) + "]." + field
}
