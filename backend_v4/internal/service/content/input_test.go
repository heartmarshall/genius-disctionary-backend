package content

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// ---------------------------------------------------------------------------
// AddSenseInput.Validate
// ---------------------------------------------------------------------------

func TestValidation_AddSenseInput(t *testing.T) {
	t.Parallel()

	invalidPOS := domain.PartOfSpeech("INVALID")

	tests := []struct {
		name    string
		input   AddSenseInput
		wantErr bool
	}{
		{
			name:    "valid minimal",
			input:   AddSenseInput{EntryID: uuid.New()},
			wantErr: false,
		},
		{
			name:    "nil entry_id",
			input:   AddSenseInput{EntryID: uuid.Nil},
			wantErr: true,
		},
		{
			name: "definition too long",
			input: AddSenseInput{
				EntryID:    uuid.New(),
				Definition: strPtr(strings.Repeat("a", 2001)),
			},
			wantErr: true,
		},
		{
			name: "definition at boundary (2000 runes)",
			input: AddSenseInput{
				EntryID:    uuid.New(),
				Definition: strPtr(strings.Repeat("a", 2000)),
			},
			wantErr: false,
		},
		{
			name: "invalid part_of_speech",
			input: AddSenseInput{
				EntryID:      uuid.New(),
				PartOfSpeech: &invalidPOS,
			},
			wantErr: true,
		},
		{
			name: "invalid CEFR level",
			input: AddSenseInput{
				EntryID:   uuid.New(),
				CEFRLevel: strPtr("Z9"),
			},
			wantErr: true,
		},
		{
			name: "valid CEFR level",
			input: AddSenseInput{
				EntryID:   uuid.New(),
				CEFRLevel: strPtr("B2"),
			},
			wantErr: false,
		},
		{
			name: "translation text too long",
			input: AddSenseInput{
				EntryID:      uuid.New(),
				Translations: []string{strings.Repeat("б", 501)},
			},
			wantErr: true,
		},
		{
			name: "translation at boundary (500 runes)",
			input: AddSenseInput{
				EntryID:      uuid.New(),
				Translations: []string{strings.Repeat("б", 500)},
			},
			wantErr: false,
		},
		{
			name: "too many translations",
			input: AddSenseInput{
				EntryID:      uuid.New(),
				Translations: make([]string, 21),
			},
			wantErr: true,
		},
		{
			name: "empty translation string",
			input: AddSenseInput{
				EntryID:      uuid.New(),
				Translations: []string{"valid", "   "},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.input.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if tt.wantErr && err != nil && !errors.Is(err, domain.ErrValidation) {
				t.Errorf("expected ErrValidation, got %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// UpdateSenseInput.Validate
// ---------------------------------------------------------------------------

func TestValidation_UpdateSenseInput(t *testing.T) {
	t.Parallel()

	invalidPOS := domain.PartOfSpeech("INVALID")

	tests := []struct {
		name    string
		input   UpdateSenseInput
		wantErr bool
	}{
		{
			name:    "valid minimal",
			input:   UpdateSenseInput{SenseID: uuid.New()},
			wantErr: false,
		},
		{
			name:    "nil sense_id",
			input:   UpdateSenseInput{SenseID: uuid.Nil},
			wantErr: true,
		},
		{
			name: "definition too long",
			input: UpdateSenseInput{
				SenseID:    uuid.New(),
				Definition: strPtr(strings.Repeat("a", 2001)),
			},
			wantErr: true,
		},
		{
			name: "invalid part_of_speech",
			input: UpdateSenseInput{
				SenseID:      uuid.New(),
				PartOfSpeech: &invalidPOS,
			},
			wantErr: true,
		},
		{
			name: "invalid CEFR level",
			input: UpdateSenseInput{
				SenseID:   uuid.New(),
				CEFRLevel: strPtr("D1"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.input.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if tt.wantErr && err != nil && !errors.Is(err, domain.ErrValidation) {
				t.Errorf("expected ErrValidation, got %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AddTranslationInput.Validate
// ---------------------------------------------------------------------------

func TestValidation_AddTranslationInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   AddTranslationInput
		wantErr bool
	}{
		{
			name:    "valid",
			input:   AddTranslationInput{SenseID: uuid.New(), Text: "перевод"},
			wantErr: false,
		},
		{
			name:    "nil sense_id",
			input:   AddTranslationInput{SenseID: uuid.Nil, Text: "перевод"},
			wantErr: true,
		},
		{
			name:    "empty text",
			input:   AddTranslationInput{SenseID: uuid.New(), Text: ""},
			wantErr: true,
		},
		{
			name:    "whitespace-only text",
			input:   AddTranslationInput{SenseID: uuid.New(), Text: "   "},
			wantErr: true,
		},
		{
			name:    "text too long",
			input:   AddTranslationInput{SenseID: uuid.New(), Text: strings.Repeat("a", 501)},
			wantErr: true,
		},
		{
			name:    "text at boundary (500 runes)",
			input:   AddTranslationInput{SenseID: uuid.New(), Text: strings.Repeat("a", 500)},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.input.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if tt.wantErr && err != nil && !errors.Is(err, domain.ErrValidation) {
				t.Errorf("expected ErrValidation, got %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// UpdateTranslationInput.Validate
// ---------------------------------------------------------------------------

func TestValidation_UpdateTranslationInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   UpdateTranslationInput
		wantErr bool
	}{
		{
			name:    "valid",
			input:   UpdateTranslationInput{TranslationID: uuid.New(), Text: "перевод"},
			wantErr: false,
		},
		{
			name:    "nil translation_id",
			input:   UpdateTranslationInput{TranslationID: uuid.Nil, Text: "перевод"},
			wantErr: true,
		},
		{
			name:    "empty text",
			input:   UpdateTranslationInput{TranslationID: uuid.New(), Text: ""},
			wantErr: true,
		},
		{
			name:    "text too long",
			input:   UpdateTranslationInput{TranslationID: uuid.New(), Text: strings.Repeat("б", 501)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.input.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if tt.wantErr && err != nil && !errors.Is(err, domain.ErrValidation) {
				t.Errorf("expected ErrValidation, got %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AddExampleInput.Validate
// ---------------------------------------------------------------------------

func TestValidation_AddExampleInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   AddExampleInput
		wantErr bool
	}{
		{
			name:    "valid with translation",
			input:   AddExampleInput{SenseID: uuid.New(), Sentence: "Hello.", Translation: strPtr("Привет.")},
			wantErr: false,
		},
		{
			name:    "valid without translation",
			input:   AddExampleInput{SenseID: uuid.New(), Sentence: "Hello."},
			wantErr: false,
		},
		{
			name:    "nil sense_id",
			input:   AddExampleInput{SenseID: uuid.Nil, Sentence: "Hello."},
			wantErr: true,
		},
		{
			name:    "empty sentence",
			input:   AddExampleInput{SenseID: uuid.New(), Sentence: ""},
			wantErr: true,
		},
		{
			name:    "whitespace-only sentence",
			input:   AddExampleInput{SenseID: uuid.New(), Sentence: "   "},
			wantErr: true,
		},
		{
			name:    "sentence too long",
			input:   AddExampleInput{SenseID: uuid.New(), Sentence: strings.Repeat("a", 2001)},
			wantErr: true,
		},
		{
			name:    "translation too long",
			input:   AddExampleInput{SenseID: uuid.New(), Sentence: "Hello.", Translation: strPtr(strings.Repeat("б", 2001))},
			wantErr: true,
		},
		{
			name:    "translation at boundary (2000 runes)",
			input:   AddExampleInput{SenseID: uuid.New(), Sentence: "Hello.", Translation: strPtr(strings.Repeat("б", 2000))},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.input.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if tt.wantErr && err != nil && !errors.Is(err, domain.ErrValidation) {
				t.Errorf("expected ErrValidation, got %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// UpdateExampleInput.Validate
// ---------------------------------------------------------------------------

func TestValidation_UpdateExampleInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   UpdateExampleInput
		wantErr bool
	}{
		{
			name:    "valid",
			input:   UpdateExampleInput{ExampleID: uuid.New(), Sentence: "Hello."},
			wantErr: false,
		},
		{
			name:    "nil example_id",
			input:   UpdateExampleInput{ExampleID: uuid.Nil, Sentence: "Hello."},
			wantErr: true,
		},
		{
			name:    "empty sentence",
			input:   UpdateExampleInput{ExampleID: uuid.New(), Sentence: ""},
			wantErr: true,
		},
		{
			name:    "sentence too long",
			input:   UpdateExampleInput{ExampleID: uuid.New(), Sentence: strings.Repeat("a", 2001)},
			wantErr: true,
		},
		{
			name:    "translation too long",
			input:   UpdateExampleInput{ExampleID: uuid.New(), Sentence: "Hello.", Translation: strPtr(strings.Repeat("б", 2001))},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.input.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if tt.wantErr && err != nil && !errors.Is(err, domain.ErrValidation) {
				t.Errorf("expected ErrValidation, got %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AddUserImageInput.Validate
// ---------------------------------------------------------------------------

func TestValidation_AddUserImageInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   AddUserImageInput
		wantErr bool
	}{
		{
			name:    "valid without caption",
			input:   AddUserImageInput{EntryID: uuid.New(), URL: "https://example.com/img.jpg"},
			wantErr: false,
		},
		{
			name:    "valid with caption",
			input:   AddUserImageInput{EntryID: uuid.New(), URL: "https://example.com/img.jpg", Caption: strPtr("A photo")},
			wantErr: false,
		},
		{
			name:    "nil entry_id",
			input:   AddUserImageInput{EntryID: uuid.Nil, URL: "https://example.com/img.jpg"},
			wantErr: true,
		},
		{
			name:    "empty URL",
			input:   AddUserImageInput{EntryID: uuid.New(), URL: ""},
			wantErr: true,
		},
		{
			name:    "non-HTTP URL",
			input:   AddUserImageInput{EntryID: uuid.New(), URL: "ftp://example.com/img.jpg"},
			wantErr: true,
		},
		{
			name:    "URL without scheme",
			input:   AddUserImageInput{EntryID: uuid.New(), URL: "example.com/img.jpg"},
			wantErr: true,
		},
		{
			name:    "caption too long",
			input:   AddUserImageInput{EntryID: uuid.New(), URL: "https://example.com/img.jpg", Caption: strPtr(strings.Repeat("a", 501))},
			wantErr: true,
		},
		{
			name:    "caption at boundary (500 runes)",
			input:   AddUserImageInput{EntryID: uuid.New(), URL: "https://example.com/img.jpg", Caption: strPtr(strings.Repeat("a", 500))},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.input.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if tt.wantErr && err != nil && !errors.Is(err, domain.ErrValidation) {
				t.Errorf("expected ErrValidation, got %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// UpdateUserImageInput.Validate
// ---------------------------------------------------------------------------

func TestValidation_UpdateUserImageInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   UpdateUserImageInput
		wantErr bool
	}{
		{
			name:    "valid with caption",
			input:   UpdateUserImageInput{ImageID: uuid.New(), Caption: strPtr("new caption")},
			wantErr: false,
		},
		{
			name:    "valid nil caption (remove)",
			input:   UpdateUserImageInput{ImageID: uuid.New(), Caption: nil},
			wantErr: false,
		},
		{
			name:    "nil image_id",
			input:   UpdateUserImageInput{ImageID: uuid.Nil},
			wantErr: true,
		},
		{
			name:    "caption too long",
			input:   UpdateUserImageInput{ImageID: uuid.New(), Caption: strPtr(strings.Repeat("б", 501))},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.input.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if tt.wantErr && err != nil && !errors.Is(err, domain.ErrValidation) {
				t.Errorf("expected ErrValidation, got %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// isValidHTTPURL
// ---------------------------------------------------------------------------

func TestIsValidHTTPURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"valid https", "https://example.com/img.jpg", true},
		{"valid http", "http://example.com/img.jpg", true},
		{"empty string", "", false},
		{"ftp scheme", "ftp://example.com", false},
		{"no scheme", "example.com/img.jpg", false},
		{"no host", "https://", false},
		{"invalid characters", "https://exam ple.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isValidHTTPURL(tt.url)
			if got != tt.want {
				t.Errorf("isValidHTTPURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// validateReorderItems
// ---------------------------------------------------------------------------

func TestValidateReorderItems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		parent  uuid.UUID
		items   []domain.ReorderItem
		wantErr bool
	}{
		{
			name:    "valid",
			parent:  uuid.New(),
			items:   []domain.ReorderItem{{ID: uuid.New(), Position: 0}},
			wantErr: false,
		},
		{
			name:    "nil parent",
			parent:  uuid.Nil,
			items:   []domain.ReorderItem{{ID: uuid.New(), Position: 0}},
			wantErr: true,
		},
		{
			name:    "empty items",
			parent:  uuid.New(),
			items:   []domain.ReorderItem{},
			wantErr: true,
		},
		{
			name:   "too many items (>50)",
			parent: uuid.New(),
			items: func() []domain.ReorderItem {
				items := make([]domain.ReorderItem, 51)
				for i := range items {
					items[i] = domain.ReorderItem{ID: uuid.New(), Position: i}
				}
				return items
			}(),
			wantErr: true,
		},
		{
			name:   "nil item ID",
			parent: uuid.New(),
			items: []domain.ReorderItem{
				{ID: uuid.Nil, Position: 0},
			},
			wantErr: true,
		},
		{
			name:   "duplicate item IDs",
			parent: uuid.New(),
			items: func() []domain.ReorderItem {
				id := uuid.New()
				return []domain.ReorderItem{
					{ID: id, Position: 0},
					{ID: id, Position: 1},
				}
			}(),
			wantErr: true,
		},
		{
			name:   "negative position",
			parent: uuid.New(),
			items: []domain.ReorderItem{
				{ID: uuid.New(), Position: -1},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := validateReorderItems("parent_id", tt.parent, tt.items)
			if tt.wantErr && len(errs) == 0 {
				t.Error("expected validation errors, got none")
			}
			if !tt.wantErr && len(errs) > 0 {
				t.Errorf("expected no errors, got %v", errs)
			}
		})
	}
}
