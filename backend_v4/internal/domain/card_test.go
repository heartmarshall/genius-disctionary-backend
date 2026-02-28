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
		name string
		card Card
		want bool
	}{
		{
			name: "NEW card is always due",
			card: Card{State: CardStateNew, Due: time.Time{}},
			want: true,
		},
		{
			name: "REVIEW card due when Due in past",
			card: Card{State: CardStateReview, Due: past},
			want: true,
		},
		{
			name: "REVIEW card not due when Due in future",
			card: Card{State: CardStateReview, Due: future},
			want: false,
		},
		{
			name: "REVIEW card due when Due equals now",
			card: Card{State: CardStateReview, Due: now},
			want: true,
		},
		{
			name: "LEARNING card due when Due in past",
			card: Card{State: CardStateLearning, Due: past},
			want: true,
		},
		{
			name: "LEARNING card not due when Due in future",
			card: Card{State: CardStateLearning, Due: future},
			want: false,
		},
		{
			name: "RELEARNING card due when Due in past",
			card: Card{State: CardStateRelearning, Due: past},
			want: true,
		},
		{
			name: "RELEARNING card not due when Due in future",
			card: Card{State: CardStateRelearning, Due: future},
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
