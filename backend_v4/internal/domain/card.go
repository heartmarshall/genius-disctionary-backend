package domain

import (
	"time"

	"github.com/google/uuid"
)

// Card represents an SRS flashcard linked 1:1 with an Entry.
type Card struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	EntryID      uuid.UUID
	Status       LearningStatus
	LearningStep int
	NextReviewAt *time.Time
	IntervalDays int
	EaseFactor   float64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IsDue returns true if the card needs review at the given time.
//   - NEW cards with no NextReviewAt are always due.
//   - LEARNING / REVIEW / MASTERED cards are due when NextReviewAt <= now.
func (c *Card) IsDue(now time.Time) bool {
	if c.Status == LearningStatusNew && c.NextReviewAt == nil {
		return true
	}
	return c.NextReviewAt != nil && !c.NextReviewAt.After(now)
}

// ReviewLog records a single review event for a card.
type ReviewLog struct {
	ID         uuid.UUID
	CardID     uuid.UUID
	Grade      ReviewGrade
	PrevState  *CardSnapshot
	DurationMs *int
	ReviewedAt time.Time
}

// CardSnapshot captures the SRS state of a card before a review (for undo).
// No json/db tags — serialization is the responsibility of the repo layer.
type CardSnapshot struct {
	Status       LearningStatus
	LearningStep int
	IntervalDays int
	EaseFactor   float64
	NextReviewAt *time.Time
}

// SRSResult is the output of a pure SRS calculation.
type SRSResult struct {
	Status       LearningStatus
	LearningStep int
	NextReviewAt time.Time
	IntervalDays int
	EaseFactor   float64
}

// StudySession tracks a user's study session from start to finish.
type StudySession struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	Status     SessionStatus
	StartedAt  time.Time
	FinishedAt *time.Time
	Result     *SessionResult
	CreatedAt  time.Time
}

// GradeCounts holds per-grade counters for a study session.
type GradeCounts struct {
	Again int
	Hard  int
	Good  int
	Easy  int
}

// SessionResult holds aggregated results of a completed study session.
type SessionResult struct {
	TotalReviewed int
	NewReviewed   int
	DueReviewed   int
	GradeCounts   GradeCounts
	DurationMs    int64
	AccuracyRate  float64
}
