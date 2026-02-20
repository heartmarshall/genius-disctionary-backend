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
	FrequencyRank  *int
	CEFRLevel      *string
	IsCoreLexicon  bool
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

// RefWordRelation represents a semantic relationship between two reference entries.
type RefWordRelation struct {
	ID            uuid.UUID
	SourceEntryID uuid.UUID
	TargetEntryID uuid.UUID
	RelationType  string // synonym, hypernym, antonym, derived
	SourceSlug    string
	CreatedAt     time.Time
}

// RefDataSource describes an external data source used to populate the reference catalog.
type RefDataSource struct {
	Slug           string
	Name           string
	Description    string
	SourceType     string // definitions, translations, pronunciations, examples, relations, metadata
	IsActive       bool
	DatasetVersion string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// RefEntrySourceCoverage tracks which data sources have been fetched for a given entry.
type RefEntrySourceCoverage struct {
	RefEntryID     uuid.UUID
	SourceSlug     string
	Status         string // fetched, no_data, failed
	DatasetVersion string
	FetchedAt      time.Time
}

// EntryMetadataUpdate holds metadata fields to update on a ref_entry.
type EntryMetadataUpdate struct {
	TextNormalized string
	FrequencyRank  *int
	CEFRLevel      *string
	IsCoreLexicon  *bool
}

// Int32PtrToIntPtr converts *int32 (sqlc) to *int (domain).
func Int32PtrToIntPtr(v *int32) *int {
	if v == nil {
		return nil
	}
	i := int(*v)
	return &i
}

// IntPtrToInt32Ptr converts *int (domain) to *int32 (sqlc).
func IntPtrToInt32Ptr(v *int) *int32 {
	if v == nil {
		return nil
	}
	i := int32(*v)
	return &i
}
