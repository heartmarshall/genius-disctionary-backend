package domain

import (
	"time"

	"github.com/google/uuid"
)

// Card represents an FSRS-5 flashcard linked 1:1 with an Entry.
type Card struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	EntryID       uuid.UUID
	State         CardState
	Step          int
	Stability     float64
	Difficulty    float64
	Due           time.Time
	LastReview    *time.Time
	Reps          int
	Lapses        int
	ScheduledDays int
	ElapsedDays   int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// IsDue returns true if the card needs review at the given time.
//   - NEW cards are always due.
//   - Other cards are due when Due <= now.
func (c *Card) IsDue(now time.Time) bool {
	if c.State == CardStateNew {
		return true
	}
	return !c.Due.After(now)
}

// ReviewLog records a single review event for a card.
type ReviewLog struct {
	ID         uuid.UUID
	CardID     uuid.UUID
	UserID     uuid.UUID
	Grade      ReviewGrade
	PrevState  *CardSnapshot
	DurationMs *int
	ReviewedAt time.Time
}

// CardSnapshot captures the FSRS state of a card before a review (for undo).
type CardSnapshot struct {
	State         CardState
	Step          int
	Stability     float64
	Difficulty    float64
	Due           time.Time
	LastReview    *time.Time
	Reps          int
	Lapses        int
	ScheduledDays int
	ElapsedDays   int
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
