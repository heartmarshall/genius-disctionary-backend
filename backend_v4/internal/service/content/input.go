package content

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// ---------------------------------------------------------------------------
// AddSenseInput
// ---------------------------------------------------------------------------

// AddSenseInput holds the parameters for adding a new sense to an entry.
type AddSenseInput struct {
	EntryID      uuid.UUID
	Definition   *string
	PartOfSpeech *domain.PartOfSpeech
	CEFRLevel    *string
	Translations []string
}

// Validate checks all fields and collects all errors.
func (i AddSenseInput) Validate() error {
	var errs []domain.FieldError

	if i.EntryID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "entry_id", Message: "required"})
	}

	if i.Definition != nil && len(*i.Definition) > 2000 {
		errs = append(errs, domain.FieldError{Field: "definition", Message: "too long (max 2000)"})
	}

	if i.CEFRLevel != nil && len(*i.CEFRLevel) > 10 {
		errs = append(errs, domain.FieldError{Field: "cefr_level", Message: "too long"})
	}

	if len(i.Translations) > 20 {
		errs = append(errs, domain.FieldError{Field: "translations", Message: "too many (max 20)"})
	}

	for idx, tr := range i.Translations {
		trimmed := strings.TrimSpace(tr)
		if trimmed == "" {
			errs = append(errs, domain.FieldError{
				Field:   fmt.Sprintf("translations[%d]", idx),
				Message: "required",
			})
		}
		if len(tr) > 500 {
			errs = append(errs, domain.FieldError{
				Field:   fmt.Sprintf("translations[%d]", idx),
				Message: "too long (max 500)",
			})
		}
	}

	if len(errs) > 0 {
		return domain.NewValidationErrors(errs)
	}
	return nil
}

// ---------------------------------------------------------------------------
// UpdateSenseInput
// ---------------------------------------------------------------------------

// UpdateSenseInput holds the parameters for updating a sense.
type UpdateSenseInput struct {
	SenseID      uuid.UUID
	Definition   *string
	PartOfSpeech *domain.PartOfSpeech
	CEFRLevel    *string
}

// Validate checks all fields and collects all errors.
func (i UpdateSenseInput) Validate() error {
	var errs []domain.FieldError

	if i.SenseID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "sense_id", Message: "required"})
	}

	if i.Definition != nil && len(*i.Definition) > 2000 {
		errs = append(errs, domain.FieldError{Field: "definition", Message: "too long"})
	}

	if i.CEFRLevel != nil && len(*i.CEFRLevel) > 10 {
		errs = append(errs, domain.FieldError{Field: "cefr_level", Message: "too long"})
	}

	if len(errs) > 0 {
		return domain.NewValidationErrors(errs)
	}
	return nil
}

// ---------------------------------------------------------------------------
// ReorderSensesInput
// ---------------------------------------------------------------------------

// ReorderSensesInput holds the parameters for reordering senses.
type ReorderSensesInput struct {
	EntryID uuid.UUID
	Items   []ReorderItem
}

// Validate checks all fields and collects all errors.
func (i ReorderSensesInput) Validate() error {
	var errs []domain.FieldError

	if i.EntryID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "entry_id", Message: "required"})
	}

	if len(i.Items) == 0 {
		errs = append(errs, domain.FieldError{Field: "items", Message: "required"})
	}

	if len(i.Items) > 50 {
		errs = append(errs, domain.FieldError{Field: "items", Message: "too many"})
	}

	for idx, item := range i.Items {
		if item.ID == uuid.Nil {
			errs = append(errs, domain.FieldError{
				Field:   fieldIndex("items", idx, "id"),
				Message: "required",
			})
		}
		if item.Position < 0 {
			errs = append(errs, domain.FieldError{
				Field:   fieldIndex("items", idx, "position"),
				Message: "must be >= 0",
			})
		}
	}

	if len(errs) > 0 {
		return domain.NewValidationErrors(errs)
	}
	return nil
}

// ---------------------------------------------------------------------------
// AddTranslationInput
// ---------------------------------------------------------------------------

// AddTranslationInput holds the parameters for adding a translation to a sense.
type AddTranslationInput struct {
	SenseID uuid.UUID
	Text    string
}

// Validate checks all fields and collects all errors.
func (i AddTranslationInput) Validate() error {
	var errs []domain.FieldError

	if i.SenseID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "sense_id", Message: "required"})
	}

	trimmed := strings.TrimSpace(i.Text)
	if trimmed == "" {
		errs = append(errs, domain.FieldError{Field: "text", Message: "required"})
	}

	if len(i.Text) > 500 {
		errs = append(errs, domain.FieldError{Field: "text", Message: "too long (max 500)"})
	}

	if len(errs) > 0 {
		return domain.NewValidationErrors(errs)
	}
	return nil
}

// ---------------------------------------------------------------------------
// UpdateTranslationInput
// ---------------------------------------------------------------------------

// UpdateTranslationInput holds the parameters for updating a translation.
type UpdateTranslationInput struct {
	TranslationID uuid.UUID
	Text          string
}

// Validate checks all fields and collects all errors.
func (i UpdateTranslationInput) Validate() error {
	var errs []domain.FieldError

	if i.TranslationID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "translation_id", Message: "required"})
	}

	trimmed := strings.TrimSpace(i.Text)
	if trimmed == "" {
		errs = append(errs, domain.FieldError{Field: "text", Message: "required"})
	}

	if len(i.Text) > 500 {
		errs = append(errs, domain.FieldError{Field: "text", Message: "too long"})
	}

	if len(errs) > 0 {
		return domain.NewValidationErrors(errs)
	}
	return nil
}

// ---------------------------------------------------------------------------
// ReorderTranslationsInput
// ---------------------------------------------------------------------------

// ReorderTranslationsInput holds the parameters for reordering translations.
type ReorderTranslationsInput struct {
	SenseID uuid.UUID
	Items   []ReorderItem
}

// Validate checks all fields and collects all errors.
func (i ReorderTranslationsInput) Validate() error {
	var errs []domain.FieldError

	if i.SenseID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "sense_id", Message: "required"})
	}

	if len(i.Items) == 0 {
		errs = append(errs, domain.FieldError{Field: "items", Message: "required"})
	}

	if len(i.Items) > 50 {
		errs = append(errs, domain.FieldError{Field: "items", Message: "too many"})
	}

	for idx, item := range i.Items {
		if item.ID == uuid.Nil {
			errs = append(errs, domain.FieldError{
				Field:   fieldIndex("items", idx, "id"),
				Message: "required",
			})
		}
		if item.Position < 0 {
			errs = append(errs, domain.FieldError{
				Field:   fieldIndex("items", idx, "position"),
				Message: "must be >= 0",
			})
		}
	}

	if len(errs) > 0 {
		return domain.NewValidationErrors(errs)
	}
	return nil
}

// ---------------------------------------------------------------------------
// AddExampleInput
// ---------------------------------------------------------------------------

// AddExampleInput holds the parameters for adding an example to a sense.
type AddExampleInput struct {
	SenseID     uuid.UUID
	Sentence    string
	Translation *string
}

// Validate checks all fields and collects all errors.
func (i AddExampleInput) Validate() error {
	var errs []domain.FieldError

	if i.SenseID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "sense_id", Message: "required"})
	}

	trimmed := strings.TrimSpace(i.Sentence)
	if trimmed == "" {
		errs = append(errs, domain.FieldError{Field: "sentence", Message: "required"})
	}

	if len(i.Sentence) > 2000 {
		errs = append(errs, domain.FieldError{Field: "sentence", Message: "too long (max 2000)"})
	}

	if i.Translation != nil && len(*i.Translation) > 2000 {
		errs = append(errs, domain.FieldError{Field: "translation", Message: "too long"})
	}

	if len(errs) > 0 {
		return domain.NewValidationErrors(errs)
	}
	return nil
}

// ---------------------------------------------------------------------------
// UpdateExampleInput
// ---------------------------------------------------------------------------

// UpdateExampleInput holds the parameters for updating an example.
type UpdateExampleInput struct {
	ExampleID   uuid.UUID
	Sentence    string
	Translation *string
}

// Validate checks all fields and collects all errors.
func (i UpdateExampleInput) Validate() error {
	var errs []domain.FieldError

	if i.ExampleID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "example_id", Message: "required"})
	}

	trimmed := strings.TrimSpace(i.Sentence)
	if trimmed == "" {
		errs = append(errs, domain.FieldError{Field: "sentence", Message: "required"})
	}

	if len(i.Sentence) > 2000 {
		errs = append(errs, domain.FieldError{Field: "sentence", Message: "too long"})
	}

	if i.Translation != nil && len(*i.Translation) > 2000 {
		errs = append(errs, domain.FieldError{Field: "translation", Message: "too long"})
	}

	if len(errs) > 0 {
		return domain.NewValidationErrors(errs)
	}
	return nil
}

// ---------------------------------------------------------------------------
// ReorderExamplesInput
// ---------------------------------------------------------------------------

// ReorderExamplesInput holds the parameters for reordering examples.
type ReorderExamplesInput struct {
	SenseID uuid.UUID
	Items   []ReorderItem
}

// Validate checks all fields and collects all errors.
func (i ReorderExamplesInput) Validate() error {
	var errs []domain.FieldError

	if i.SenseID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "sense_id", Message: "required"})
	}

	if len(i.Items) == 0 {
		errs = append(errs, domain.FieldError{Field: "items", Message: "required"})
	}

	if len(i.Items) > 50 {
		errs = append(errs, domain.FieldError{Field: "items", Message: "too many"})
	}

	for idx, item := range i.Items {
		if item.ID == uuid.Nil {
			errs = append(errs, domain.FieldError{
				Field:   fieldIndex("items", idx, "id"),
				Message: "required",
			})
		}
		if item.Position < 0 {
			errs = append(errs, domain.FieldError{
				Field:   fieldIndex("items", idx, "position"),
				Message: "must be >= 0",
			})
		}
	}

	if len(errs) > 0 {
		return domain.NewValidationErrors(errs)
	}
	return nil
}

// ---------------------------------------------------------------------------
// AddUserImageInput
// ---------------------------------------------------------------------------

// AddUserImageInput holds the parameters for adding a user image to an entry.
type AddUserImageInput struct {
	EntryID uuid.UUID
	URL     string
	Caption *string
}

// Validate checks all fields and collects all errors.
func (i AddUserImageInput) Validate() error {
	var errs []domain.FieldError

	if i.EntryID == uuid.Nil {
		errs = append(errs, domain.FieldError{Field: "entry_id", Message: "required"})
	}

	trimmed := strings.TrimSpace(i.URL)
	if trimmed == "" {
		errs = append(errs, domain.FieldError{Field: "url", Message: "required"})
	}

	if !isValidHTTPURL(i.URL) {
		errs = append(errs, domain.FieldError{Field: "url", Message: "must be a valid HTTP(S) URL"})
	}

	if len(i.URL) > 2000 {
		errs = append(errs, domain.FieldError{Field: "url", Message: "too long"})
	}

	if i.Caption != nil && len(*i.Caption) > 500 {
		errs = append(errs, domain.FieldError{Field: "caption", Message: "too long (max 500)"})
	}

	if len(errs) > 0 {
		return domain.NewValidationErrors(errs)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// isValidHTTPURL checks if the URL is a valid HTTP or HTTPS URL.
func isValidHTTPURL(rawURL string) bool {
	if rawURL == "" {
		return false
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	if u.Host == "" {
		return false
	}

	return true
}

// fieldIndex formats a nested field path like "items[0].id".
func fieldIndex(parent string, idx int, field string) string {
	return parent + "[" + strconv.Itoa(idx) + "]." + field
}
