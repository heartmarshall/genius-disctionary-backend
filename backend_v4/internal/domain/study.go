package domain

import (
	"time"

	"github.com/google/uuid"
)

// SRSConfig holds spaced-repetition algorithm parameters (pure domain type).
// This is a clean copy of config.SRSConfig without external tags.
type SRSConfig struct {
	DefaultEaseFactor    float64
	MinEaseFactor        float64
	MaxIntervalDays      int
	GraduatingInterval   int
	LearningSteps        []time.Duration
	NewCardsPerDay       int
	ReviewsPerDay        int
	EasyInterval         int
	RelearningSteps      []time.Duration
	IntervalModifier     float64
	HardIntervalModifier float64
	EasyBonus            float64
	LapseNewInterval     float64
	UndoWindowMinutes    int
}

// SRSUpdateParams holds the fields to update on a card after SRS calculation.
type SRSUpdateParams struct {
	Status       LearningStatus
	NextReviewAt *time.Time // Pointer to properly represent NULL for NEW cards
	IntervalDays int
	EaseFactor   float64
	LearningStep int
}

// CardStatusCounts holds the count of cards per learning status.
type CardStatusCounts struct {
	New      int
	Learning int
	Review   int
	Mastered int
	Total    int
}

// Dashboard holds aggregated study statistics for the user.
type Dashboard struct {
	DueCount      int
	NewCount      int
	ReviewedToday int
	NewToday      int
	Streak        int
	StatusCounts  CardStatusCounts
	OverdueCount  int
	ActiveSession *uuid.UUID
}

// DayReviewCount holds the review count for a specific date.
type DayReviewCount struct {
	Date  time.Time
	Count int
}

// CardStats holds statistics for a single card.
type CardStats struct {
	TotalReviews  int
	AccuracyRate  float64
	AverageTimeMs *int
	CurrentStatus LearningStatus
	IntervalDays  int
	EaseFactor    float64
}
