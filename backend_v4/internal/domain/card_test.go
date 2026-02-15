package domain

import (
	"testing"
	"time"
)

func TestCard_IsDue(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name   string
		card   Card
		want   bool
	}{
		{
			name: "MASTERED card due when next_review_at in past",
			card: Card{Status: LearningStatusMastered, NextReviewAt: &past},
			want: true,
		},
		{
			name: "MASTERED card not due when next_review_at in future",
			card: Card{Status: LearningStatusMastered, NextReviewAt: &future},
			want: false,
		},
		{
			name: "MASTERED with nil NextReviewAt is not due",
			card: Card{Status: LearningStatusMastered, NextReviewAt: nil},
			want: false,
		},
		{
			name: "NEW with nil NextReviewAt is due",
			card: Card{Status: LearningStatusNew, NextReviewAt: nil},
			want: true,
		},
		{
			name: "NEW with past NextReviewAt is due",
			card: Card{Status: LearningStatusNew, NextReviewAt: &past},
			want: true,
		},
		{
			name: "NEW with future NextReviewAt is not due",
			card: Card{Status: LearningStatusNew, NextReviewAt: &future},
			want: false,
		},
		{
			name: "LEARNING with past NextReviewAt is due",
			card: Card{Status: LearningStatusLearning, NextReviewAt: &past},
			want: true,
		},
		{
			name: "LEARNING with now NextReviewAt is due",
			card: Card{Status: LearningStatusLearning, NextReviewAt: &now},
			want: true,
		},
		{
			name: "LEARNING with future NextReviewAt is not due",
			card: Card{Status: LearningStatusLearning, NextReviewAt: &future},
			want: false,
		},
		{
			name: "REVIEW with past NextReviewAt is due",
			card: Card{Status: LearningStatusReview, NextReviewAt: &past},
			want: true,
		},
		{
			name: "REVIEW with future NextReviewAt is not due",
			card: Card{Status: LearningStatusReview, NextReviewAt: &future},
			want: false,
		},
		{
			name: "LEARNING with nil NextReviewAt is not due",
			card: Card{Status: LearningStatusLearning, NextReviewAt: nil},
			want: false,
		},
		{
			name: "REVIEW with nil NextReviewAt is not due",
			card: Card{Status: LearningStatusReview, NextReviewAt: nil},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.card.IsDue(now); got != tt.want {
				t.Errorf("Card.IsDue() = %v, want %v", got, tt.want)
			}
		})
	}
}
