package domain

import (
	"time"

	"github.com/google/uuid"
)

// EnrichmentStatus represents the processing state of a queued word.
type EnrichmentStatus string

const (
	EnrichmentStatusPending    EnrichmentStatus = "pending"
	EnrichmentStatusProcessing EnrichmentStatus = "processing"
	EnrichmentStatusDone       EnrichmentStatus = "done"
	EnrichmentStatusFailed     EnrichmentStatus = "failed"
)

func (s EnrichmentStatus) IsValid() bool {
	switch s {
	case EnrichmentStatusPending, EnrichmentStatusProcessing, EnrichmentStatusDone, EnrichmentStatusFailed:
		return true
	}
	return false
}

// EnrichmentQueueItem represents a word queued for LLM enrichment.
type EnrichmentQueueItem struct {
	ID           uuid.UUID
	RefEntryID   uuid.UUID
	Status       EnrichmentStatus
	Priority     int
	ErrorMessage *string
	RequestedAt  time.Time
	ProcessedAt  *time.Time
	CreatedAt    time.Time
}

// EnrichmentQueueStats holds aggregate counts by status.
type EnrichmentQueueStats struct {
	Pending    int
	Processing int
	Done       int
	Failed     int
	Total      int
}
