package domain

import (
	"time"

	"github.com/google/uuid"
)

// RefEntry is an immutable reference catalog entry (shared across users).
type RefEntry struct {
	ID             uuid.UUID
	Text           string
	TextNormalized string
	CreatedAt      time.Time

	Senses         []RefSense
	Pronunciations []RefPronunciation
	Images         []RefImage
}

// RefSense is a reference sense from an external source.
type RefSense struct {
	ID           uuid.UUID
	RefEntryID   uuid.UUID
	Definition   string
	PartOfSpeech *PartOfSpeech
	CEFRLevel    *string
	SourceSlug   string
	Position     int
	CreatedAt    time.Time

	Translations []RefTranslation
	Examples     []RefExample
}

// RefTranslation is a reference translation from an external source.
type RefTranslation struct {
	ID         uuid.UUID
	RefSenseID uuid.UUID
	Text       string
	SourceSlug string
	Position   int
}

// RefExample is a reference usage example from an external source.
type RefExample struct {
	ID          uuid.UUID
	RefSenseID  uuid.UUID
	Sentence    string
	Translation *string
	SourceSlug  string
	Position    int
}

// RefPronunciation is a reference pronunciation from an external source.
type RefPronunciation struct {
	ID            uuid.UUID
	RefEntryID    uuid.UUID
	Transcription *string
	AudioURL      *string
	Region        *string
	SourceSlug    string
}

// RefImage is a reference image from an external source.
type RefImage struct {
	ID         uuid.UUID
	RefEntryID uuid.UUID
	URL        string
	Caption    *string
	SourceSlug string
}
