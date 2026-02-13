package domain

import (
	"time"

	"github.com/google/uuid"
)

// Entry is a user's dictionary entry, optionally linked to a reference catalog entry.
type Entry struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	RefEntryID     *uuid.UUID
	Text           string
	TextNormalized string
	Notes          *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time

	Senses         []Sense
	Pronunciations []RefPronunciation
	CatalogImages  []RefImage
	UserImages     []UserImage
	Card           *Card
	Topics         []Topic
}

// IsDeleted returns true if the entry has been soft-deleted.
func (e *Entry) IsDeleted() bool {
	return e.DeletedAt != nil
}

// Sense is a user's sense, optionally inheriting data from a reference sense via COALESCE.
type Sense struct {
	ID           uuid.UUID
	EntryID      uuid.UUID
	RefSenseID   *uuid.UUID
	Definition   *string
	PartOfSpeech *PartOfSpeech
	CEFRLevel    *string
	SourceSlug   string
	Position     int
	CreatedAt    time.Time

	Translations []Translation
	Examples     []Example
}

// Translation is a user's translation, optionally inheriting from a reference translation.
type Translation struct {
	ID               uuid.UUID
	SenseID          uuid.UUID
	RefTranslationID *uuid.UUID
	Text             *string
	SourceSlug       string
	Position         int
}

// Example is a user's usage example, optionally inheriting from a reference example.
type Example struct {
	ID           uuid.UUID
	SenseID      uuid.UUID
	RefExampleID *uuid.UUID
	Sentence     *string
	Translation  *string
	SourceSlug   string
	Position     int
	CreatedAt    time.Time
}

// UserImage is an image uploaded by the user (not from the reference catalog).
type UserImage struct {
	ID        uuid.UUID
	EntryID   uuid.UUID
	URL       string
	Caption   *string
	CreatedAt time.Time
}
