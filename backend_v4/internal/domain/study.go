package domain

import (
	"time"
)

// SRSConfig holds FSRS-5 spaced-repetition algorithm parameters (pure domain type).
type SRSConfig struct {
	DefaultRetention  float64
	MaxIntervalDays   int
	EnableFuzz        bool
	LearningSteps     []time.Duration
	RelearningSteps   []time.Duration
	NewCardsPerDay    int
	ReviewsPerDay     int // Not enforced in study queue. Due cards are always shown regardless of this limit.
	UndoWindowMinutes int
}

// SRSUpdateParams holds the fields to update on a card after FSRS calculation.
type SRSUpdateParams struct {
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

// CardStatusCounts holds the count of cards per state.
type CardStatusCounts struct {
	New        int
	Learning   int
	Review     int
	Relearning int
	Total      int
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
	ActiveSession *StudySession
}

// DayReviewCount holds the review count for a specific date.
type DayReviewCount struct {
	Date  time.Time
	Count int
}

// ReviewLogAggregation holds aggregated review stats computed in SQL.
type ReviewLogAggregation struct {
	TotalReviews  int
	AgainCount    int
	HardCount     int
	GoodCount     int
	EasyCount     int
	AvgDurationMs *int
}

// CardStats holds statistics for a single card.
type CardStats struct {
	TotalReviews      int
	AccuracyRate      float64
	AverageTimeMs     *int
	CurrentState      CardState
	Stability         float64
	Difficulty        float64
	ScheduledDays     int
	GradeDistribution *GradeCounts
}
