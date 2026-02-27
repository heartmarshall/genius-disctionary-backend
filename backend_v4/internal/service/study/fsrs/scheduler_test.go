package fsrs

import (
	"testing"
	"time"
)

func newTestParams() Parameters {
	return Parameters{
		W:                DefaultWeights,
		DesiredRetention: 0.9,
		MaxIntervalDays:  365,
		EnableFuzz:       false, // disable fuzz for deterministic tests
		LearningSteps:    []time.Duration{time.Minute, 10 * time.Minute},
		RelearningSteps:  []time.Duration{10 * time.Minute},
	}
}

func TestReviewNew_Again(t *testing.T) {
	params := newTestParams()
	card := Card{State: StateNew}
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	result := ReviewCard(params, card, Again, now)

	if result.State != StateLearning {
		t.Errorf("state = %s, want LEARNING", result.State)
	}
	if result.Step != 0 {
		t.Errorf("step = %d, want 0", result.Step)
	}
	if result.ScheduledDays != 0 {
		t.Errorf("scheduledDays = %d, want 0", result.ScheduledDays)
	}
	if result.Stability <= 0 {
		t.Errorf("stability should be > 0, got %f", result.Stability)
	}
}

func TestReviewNew_Good_StepProgression(t *testing.T) {
	params := newTestParams()
	card := Card{State: StateNew}
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	result := ReviewCard(params, card, Good, now)

	if result.State != StateLearning {
		t.Errorf("state = %s, want LEARNING (should go to step 1)", result.State)
	}
	if result.Step != 1 {
		t.Errorf("step = %d, want 1", result.Step)
	}
}

func TestReviewNew_Easy_Graduate(t *testing.T) {
	params := newTestParams()
	card := Card{State: StateNew}
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	result := ReviewCard(params, card, Easy, now)

	if result.State != StateReview {
		t.Errorf("state = %s, want REVIEW (should graduate)", result.State)
	}
	if result.ScheduledDays < 1 {
		t.Errorf("scheduledDays = %d, want >= 1", result.ScheduledDays)
	}
}

func TestReviewLearning_Good_Graduate(t *testing.T) {
	params := newTestParams()
	card := Card{
		State:     StateLearning,
		Step:      1, // last learning step
		Stability: 3.0,
		Difficulty: 5.0,
	}
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	result := ReviewCard(params, card, Good, now)

	if result.State != StateReview {
		t.Errorf("state = %s, want REVIEW (should graduate from last step)", result.State)
	}
	if result.ScheduledDays < 1 {
		t.Errorf("scheduledDays = %d, want >= 1", result.ScheduledDays)
	}
}

func TestReviewLearning_Again_ResetStep(t *testing.T) {
	params := newTestParams()
	card := Card{
		State:      StateLearning,
		Step:       1,
		Stability:  3.0,
		Difficulty: 5.0,
	}
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	result := ReviewCard(params, card, Again, now)

	if result.State != StateLearning {
		t.Errorf("state = %s, want LEARNING", result.State)
	}
	if result.Step != 0 {
		t.Errorf("step = %d, want 0 (should reset)", result.Step)
	}
}

func TestReviewReview_IntervalOrdering(t *testing.T) {
	params := newTestParams()

	// Test with various S/D combinations
	testCases := []struct {
		name       string
		stability  float64
		difficulty float64
	}{
		{"low S low D", 5.0, 3.0},
		{"medium S medium D", 20.0, 5.0},
		{"high S high D", 100.0, 8.0},
		{"low S high D", 5.0, 9.0},
		{"high S low D", 100.0, 2.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			card := Card{
				State:       StateReview,
				Stability:   tc.stability,
				Difficulty:  tc.difficulty,
				ElapsedDays: int(tc.stability), // approximately due
				Reps:        5,
			}
			now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

			hardCard := ReviewCard(params, card, Hard, now)
			goodCard := ReviewCard(params, card, Good, now)
			easyCard := ReviewCard(params, card, Easy, now)

			if hardCard.ScheduledDays > goodCard.ScheduledDays {
				t.Errorf("Hard (%d) > Good (%d)", hardCard.ScheduledDays, goodCard.ScheduledDays)
			}
			if goodCard.ScheduledDays >= easyCard.ScheduledDays {
				t.Errorf("Good (%d) >= Easy (%d)", goodCard.ScheduledDays, easyCard.ScheduledDays)
			}
		})
	}
}

func TestReviewReview_Again_Lapse(t *testing.T) {
	params := newTestParams()
	card := Card{
		State:       StateReview,
		Stability:   20.0,
		Difficulty:  5.0,
		ElapsedDays: 20,
		Reps:        10,
		Lapses:      0,
	}
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	result := ReviewCard(params, card, Again, now)

	if result.State != StateRelearning {
		t.Errorf("state = %s, want RELEARNING", result.State)
	}
	if result.Lapses != 1 {
		t.Errorf("lapses = %d, want 1", result.Lapses)
	}
	if result.Step != 0 {
		t.Errorf("step = %d, want 0", result.Step)
	}
	if result.Stability >= card.Stability {
		t.Errorf("stability should decrease on lapse: got %f, was %f", result.Stability, card.Stability)
	}
}

func TestReviewReview_LapseCappedByNextSMin(t *testing.T) {
	params := newTestParams()
	card := Card{
		State:       StateReview,
		Stability:   50.0,
		Difficulty:  5.0,
		ElapsedDays: 50,
		Reps:        20,
	}
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	result := ReviewCard(params, card, Again, now)

	sMin := NextSMin(params.W, card.Stability)
	if result.Stability > sMin+0.001 {
		t.Errorf("post-lapse stability %f exceeds nextSMin %f", result.Stability, sMin)
	}
}

func TestReviewRelearning_Graduate(t *testing.T) {
	params := newTestParams()
	card := Card{
		State:      StateRelearning,
		Step:       0, // only one relearning step
		Stability:  5.0,
		Difficulty: 6.0,
		Lapses:     1,
	}
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	result := ReviewCard(params, card, Good, now)

	if result.State != StateReview {
		t.Errorf("state = %s, want REVIEW (should graduate from relearning)", result.State)
	}
	if result.ScheduledDays < 1 {
		t.Errorf("scheduledDays = %d, want >= 1", result.ScheduledDays)
	}
}

func TestReviewCard_FuzzEnabled(t *testing.T) {
	params := newTestParams()
	params.EnableFuzz = true

	card := Card{
		State:       StateReview,
		Stability:   30.0,
		Difficulty:  5.0,
		ElapsedDays: 30,
		Reps:        10,
	}
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	// With fuzz, should still maintain ordering
	hardCard := ReviewCard(params, card, Hard, now)
	goodCard := ReviewCard(params, card, Good, now)
	easyCard := ReviewCard(params, card, Easy, now)

	if hardCard.ScheduledDays > goodCard.ScheduledDays {
		t.Errorf("With fuzz: Hard (%d) > Good (%d)", hardCard.ScheduledDays, goodCard.ScheduledDays)
	}
	if goodCard.ScheduledDays >= easyCard.ScheduledDays {
		t.Errorf("With fuzz: Good (%d) >= Easy (%d)", goodCard.ScheduledDays, easyCard.ScheduledDays)
	}
}

func TestReviewNew_Hard_DelayBetweenSteps(t *testing.T) {
	params := newTestParams()
	card := Card{State: StateNew}
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	result := ReviewCard(params, card, Hard, now)

	if result.State != StateLearning {
		t.Errorf("state = %s, want LEARNING", result.State)
	}

	// Hard delay should be avg of step 0 and step 1: (1m + 10m) / 2 = 5.5m
	expectedDue := now.Add((time.Minute + 10*time.Minute) / 2)
	if !result.Due.Equal(expectedDue) {
		t.Errorf("due = %v, want %v", result.Due, expectedDue)
	}
}

func TestReviewCard_RepsIncrement(t *testing.T) {
	params := newTestParams()
	card := Card{State: StateNew, Reps: 0}
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	result := ReviewCard(params, card, Good, now)
	if result.Reps != 1 {
		t.Errorf("reps = %d, want 1", result.Reps)
	}

	// Review the learning card
	result = ReviewCard(params, result, Good, now.Add(10*time.Minute))
	if result.Reps != 2 {
		t.Errorf("reps = %d, want 2", result.Reps)
	}
}

func TestReviewCard_MaxIntervalCap(t *testing.T) {
	params := newTestParams()
	params.MaxIntervalDays = 30

	card := Card{
		State:       StateReview,
		Stability:   200.0, // very high stability
		Difficulty:  3.0,
		ElapsedDays: 200,
		Reps:        50,
	}
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	result := ReviewCard(params, card, Easy, now)

	if result.ScheduledDays > 30 {
		t.Errorf("scheduledDays = %d, exceeds maxIntervalDays 30", result.ScheduledDays)
	}
}

func TestReviewReview_ElapsedDaysAffectsStability(t *testing.T) {
	params := newTestParams()

	// Two cards with same S, D but different elapsed days.
	base := Card{
		State:      StateReview,
		Stability:  10.0,
		Difficulty: 5.0,
		Reps:       5,
	}
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	// Card reviewed on time (elapsed ≈ stability)
	onTime := base
	onTime.ElapsedDays = 10
	resultOnTime := ReviewCard(params, onTime, Good, now)

	// Card reviewed very late (elapsed >> stability)
	late := base
	late.ElapsedDays = 30
	resultLate := ReviewCard(params, late, Good, now)

	// A late successful review should increase stability MORE because R is lower.
	if resultLate.Stability <= resultOnTime.Stability {
		t.Errorf("late review should increase stability more: late=%f, onTime=%f",
			resultLate.Stability, resultOnTime.Stability)
	}
}

func TestReviewLearning_Easy_IntervalGteGood(t *testing.T) {
	params := newTestParams()
	card := Card{
		State:      StateLearning,
		Step:       0,
		Stability:  3.0,
		Difficulty: 5.0,
	}
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	goodCard := ReviewCard(params, card, Good, now)
	easyCard := ReviewCard(params, card, Easy, now)

	// If good graduated, easy should have >= good + 1 interval
	if goodCard.State == StateReview && easyCard.State == StateReview {
		if easyCard.ScheduledDays <= goodCard.ScheduledDays {
			t.Errorf("Easy interval (%d) should be > Good interval (%d)",
				easyCard.ScheduledDays, goodCard.ScheduledDays)
		}
	}
}
