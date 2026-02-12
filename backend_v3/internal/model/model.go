package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// CORE ENTITIES
// ============================================================================

type DictionaryEntry struct {
	ID             uuid.UUID `db:"id" json:"id"`
	Text           string    `db:"text" json:"text"`
	TextNormalized string    `db:"text_normalized" json:"text_normalized"`
	Notes          *string   `db:"notes" json:"notes"` // Пользовательские заметки о слове
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

type Sense struct {
	ID           uuid.UUID     `db:"id" json:"id"`
	EntryID      uuid.UUID     `db:"entry_id" json:"entry_id"`
	Definition   *string       `db:"definition" json:"definition"` // Nullable
	PartOfSpeech *PartOfSpeech `db:"part_of_speech" json:"part_of_speech"`
	SourceSlug   string        `db:"source_slug" json:"source_slug"`
	CefrLevel    *string       `db:"cefr_level" json:"cefr_level"`
	CreatedAt    time.Time     `db:"created_at" json:"created_at"`
}

type Translation struct {
	ID         uuid.UUID `db:"id" json:"id"`
	SenseID    uuid.UUID `db:"sense_id" json:"sense_id"`
	Text       string    `db:"text" json:"text"`
	SourceSlug string    `db:"source_slug" json:"source_slug"`
}

type Example struct {
	ID          uuid.UUID `db:"id" json:"id"`
	SenseID     uuid.UUID `db:"sense_id" json:"sense_id"`
	Sentence    string    `db:"sentence" json:"sentence"`
	Translation *string   `db:"translation" json:"translation"`
	SourceSlug  string    `db:"source_slug" json:"source_slug"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type Image struct {
	ID         uuid.UUID `db:"id" json:"id"`
	EntryID    uuid.UUID `db:"entry_id" json:"entry_id"`
	URL        string    `db:"url" json:"url"`
	Caption    *string   `db:"caption" json:"caption"`
	SourceSlug string    `db:"source_slug" json:"source_slug"`
}

type Pronunciation struct {
	ID            uuid.UUID `db:"id" json:"id"`
	EntryID       uuid.UUID `db:"entry_id" json:"entry_id"`
	AudioURL      *string   `db:"audio_url" json:"audio_url"`         // Опционально
	Transcription string    `db:"transcription" json:"transcription"` // Обязательно
	Region        *string   `db:"region" json:"region"`
	SourceSlug    string    `db:"source_slug" json:"source_slug"`
}

// ============================================================================
// STUDY / SRS ENTITIES
// ============================================================================

type Card struct {
	ID           uuid.UUID      `db:"id" json:"id"`
	EntryID      uuid.UUID      `db:"entry_id" json:"entry_id"`
	Status       LearningStatus `db:"status" json:"status"`
	NextReviewAt *time.Time     `db:"next_review_at" json:"next_review_at"`
	IntervalDays int            `db:"interval_days" json:"interval_days"`
	EaseFactor   float64        `db:"ease_factor" json:"ease_factor"`
	CreatedAt    time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at" json:"updated_at"`
}

type Hint struct {
	ID        uuid.UUID `db:"id" json:"id"`
	CardID    uuid.UUID `db:"card_id" json:"card_id"`
	Text      string    `db:"text" json:"text"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type ReviewLog struct {
	ID         uuid.UUID   `db:"id" json:"id"`
	CardID     uuid.UUID   `db:"card_id" json:"card_id"`
	Grade      ReviewGrade `db:"grade" json:"grade"`
	DurationMs *int        `db:"duration_ms" json:"duration_ms"`
	ReviewedAt time.Time   `db:"reviewed_at" json:"reviewed_at"`
}

// ============================================================================
// INBOX & AUDIT
// ============================================================================

type InboxItem struct {
	ID        uuid.UUID `db:"id" json:"id"`
	Text      string    `db:"text" json:"text"`
	Context   *string   `db:"context" json:"context"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type AuditRecord struct {
	ID         uuid.UUID   `db:"id" json:"id"`
	EntityType EntityType  `db:"entity_type" json:"entity_type"`
	EntityID   *uuid.UUID  `db:"entity_id" json:"entity_id"`
	Action     AuditAction `db:"action" json:"action"`
	Changes    JSON        `db:"changes" json:"changes"` // JSONB
	CreatedAt  time.Time   `db:"created_at" json:"created_at"`
}

// ============================================================================
// TYPES & HELPERS
// ============================================================================

// JSON is a helper for handling JSONB in Postgres.
type JSON map[string]any

// Make sure JSON implements Valuer and Scanner for database/sql
func (j JSON) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSON) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &j)
}

// ============================================================================
// TOPICS
// ============================================================================

type Topic struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description *string   `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
}
