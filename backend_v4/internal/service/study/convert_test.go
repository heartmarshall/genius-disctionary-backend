package study

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/service/study/fsrs"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

func TestCardToFSRS(t *testing.T) {
	t.Parallel()

	now := time.Now()
	lastReview := now.Add(-24 * time.Hour)

	card := &domain.Card{
		ID:            uuid.New(),
		UserID:        uuid.New(),
		EntryID:       uuid.New(),
		State:         domain.CardStateReview,
		Step:          2,
		Stability:     15.5,
		Difficulty:    5.3,
		Due:           now,
		LastReview:    &lastReview,
		Reps:          10,
		Lapses:        2,
		ScheduledDays: 7,
		ElapsedDays:   3,
	}

	result := cardToFSRS(card)

	if result.State != card.State {
		t.Errorf("State: got %v, want %v", result.State, card.State)
	}
	if result.Step != card.Step {
		t.Errorf("Step: got %d, want %d", result.Step, card.Step)
	}
	if result.Stability != card.Stability {
		t.Errorf("Stability: got %f, want %f", result.Stability, card.Stability)
	}
	if result.Difficulty != card.Difficulty {
		t.Errorf("Difficulty: got %f, want %f", result.Difficulty, card.Difficulty)
	}
	if !result.Due.Equal(card.Due) {
		t.Errorf("Due: got %v, want %v", result.Due, card.Due)
	}
	if result.LastReview == nil || !result.LastReview.Equal(*card.LastReview) {
		t.Errorf("LastReview: got %v, want %v", result.LastReview, card.LastReview)
	}
	if result.Reps != card.Reps {
		t.Errorf("Reps: got %d, want %d", result.Reps, card.Reps)
	}
	if result.Lapses != card.Lapses {
		t.Errorf("Lapses: got %d, want %d", result.Lapses, card.Lapses)
	}
	if result.ScheduledDays != card.ScheduledDays {
		t.Errorf("ScheduledDays: got %d, want %d", result.ScheduledDays, card.ScheduledDays)
	}
	if result.ElapsedDays != card.ElapsedDays {
		t.Errorf("ElapsedDays: got %d, want %d", result.ElapsedDays, card.ElapsedDays)
	}
}

func TestCardToFSRS_NilLastReview(t *testing.T) {
	t.Parallel()

	card := &domain.Card{
		State: domain.CardStateNew,
	}
	result := cardToFSRS(card)
	if result.LastReview != nil {
		t.Errorf("LastReview should be nil for new card")
	}
}

func TestFsrsResultToUpdateParams(t *testing.T) {
	t.Parallel()

	now := time.Now()
	lastReview := now

	result := fsrs.Card{
		State:         domain.CardStateReview,
		Step:          0,
		Stability:     20.0,
		Difficulty:    4.5,
		Due:           now.Add(7 * 24 * time.Hour),
		LastReview:    &lastReview,
		Reps:          11,
		Lapses:        2,
		ScheduledDays: 7,
		ElapsedDays:   3,
	}

	params := fsrsResultToUpdateParams(result)

	if params.State != result.State {
		t.Errorf("State: got %v, want %v", params.State, result.State)
	}
	if params.Stability != result.Stability {
		t.Errorf("Stability: got %f, want %f", params.Stability, result.Stability)
	}
	if params.Due != result.Due {
		t.Errorf("Due: got %v, want %v", params.Due, result.Due)
	}
	if params.LastReview == nil || !params.LastReview.Equal(*result.LastReview) {
		t.Errorf("LastReview mismatch")
	}
}

func TestSnapshotFromCard(t *testing.T) {
	t.Parallel()

	now := time.Now()
	lastReview := now.Add(-48 * time.Hour)

	card := &domain.Card{
		State:         domain.CardStateLearning,
		Step:          1,
		Stability:     3.0,
		Difficulty:    6.0,
		Due:           now,
		LastReview:    &lastReview,
		Reps:          5,
		Lapses:        1,
		ScheduledDays: 2,
		ElapsedDays:   2,
	}

	snap := snapshotFromCard(card)

	if snap.State != card.State {
		t.Errorf("State: got %v, want %v", snap.State, card.State)
	}
	if snap.Step != card.Step {
		t.Errorf("Step: got %d, want %d", snap.Step, card.Step)
	}
	if snap.Stability != card.Stability {
		t.Errorf("Stability: got %f, want %f", snap.Stability, card.Stability)
	}
	if snap.LastReview == nil || !snap.LastReview.Equal(*card.LastReview) {
		t.Errorf("LastReview mismatch")
	}
}

func TestComputeElapsedDays(t *testing.T) {
	t.Parallel()

	now := time.Now()

	t.Run("nil last review returns 0", func(t *testing.T) {
		if got := computeElapsedDays(nil, now); got != 0 {
			t.Errorf("got %d, want 0", got)
		}
	})

	t.Run("same day returns 0", func(t *testing.T) {
		lr := now.Add(-1 * time.Hour)
		if got := computeElapsedDays(&lr, now); got != 0 {
			t.Errorf("got %d, want 0", got)
		}
	})

	t.Run("3 days ago returns 3", func(t *testing.T) {
		lr := now.Add(-72 * time.Hour)
		if got := computeElapsedDays(&lr, now); got != 3 {
			t.Errorf("got %d, want 3", got)
		}
	})

	t.Run("future last review returns 0 (clamped)", func(t *testing.T) {
		lr := now.Add(24 * time.Hour)
		if got := computeElapsedDays(&lr, now); got != 0 {
			t.Errorf("got %d, want 0", got)
		}
	})
}

func TestUserID(t *testing.T) {
	t.Parallel()

	svc := &Service{}

	t.Run("extracts user ID from context", func(t *testing.T) {
		id := uuid.New()
		ctx := ctxutil.WithUserID(context.Background(), id)
		got, err := svc.userID(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != id {
			t.Errorf("got %v, want %v", got, id)
		}
	})

	t.Run("returns ErrUnauthorized when no user in context", func(t *testing.T) {
		_, err := svc.userID(context.Background())
		if err != domain.ErrUnauthorized {
			t.Errorf("got %v, want ErrUnauthorized", err)
		}
	})
}

func TestBuildFSRSParams(t *testing.T) {
	t.Parallel()

	svc := &Service{
		fsrsWeights: [19]float64{0.4, 0.6, 2.4, 5.8, 4.93, 0.94, 0.86, 0.01, 1.49, 0.14, 0.94, 2.18, 0.05, 0.34, 1.26, 0.29, 2.61, 0.0, 0.0},
		srsConfig: domain.SRSConfig{
			MaxIntervalDays: 365,
			EnableFuzz:      true,
			LearningSteps:   []time.Duration{1 * time.Minute, 10 * time.Minute},
			RelearningSteps: []time.Duration{10 * time.Minute},
		},
	}

	settings := &domain.UserSettings{
		DesiredRetention: 0.9,
		MaxIntervalDays:  180, // user's limit is lower than global
	}

	params := svc.buildFSRSParams(settings)

	if params.DesiredRetention != 0.9 {
		t.Errorf("DesiredRetention: got %f, want 0.9", params.DesiredRetention)
	}
	if params.MaxIntervalDays != 180 {
		t.Errorf("MaxIntervalDays: got %d, want 180 (min of 365 and 180)", params.MaxIntervalDays)
	}
	if !params.EnableFuzz {
		t.Errorf("EnableFuzz: got false, want true")
	}
	if len(params.LearningSteps) != 2 {
		t.Errorf("LearningSteps: got %d, want 2", len(params.LearningSteps))
	}
}

func TestAggregateSessionResult(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)
	now := time.Date(2026, 2, 28, 10, 15, 0, 0, time.UTC) // 15 min session

	logs := []*domain.ReviewLog{
		{Grade: domain.ReviewGradeGood, PrevState: &domain.CardSnapshot{State: domain.CardStateNew}},
		{Grade: domain.ReviewGradeAgain, PrevState: &domain.CardSnapshot{State: domain.CardStateReview}},
		{Grade: domain.ReviewGradeEasy, PrevState: &domain.CardSnapshot{State: domain.CardStateNew}},
		{Grade: domain.ReviewGradeHard, PrevState: &domain.CardSnapshot{State: domain.CardStateLearning}},
		{Grade: domain.ReviewGradeGood, PrevState: &domain.CardSnapshot{State: domain.CardStateReview}},
	}

	result := aggregateSessionResult(logs, startedAt, now)

	if result.TotalReviewed != 5 {
		t.Errorf("TotalReviewed: got %d, want 5", result.TotalReviewed)
	}
	if result.NewReviewed != 2 {
		t.Errorf("NewReviewed: got %d, want 2", result.NewReviewed)
	}
	if result.DueReviewed != 3 {
		t.Errorf("DueReviewed: got %d, want 3", result.DueReviewed)
	}
	if result.GradeCounts.Again != 1 {
		t.Errorf("Again: got %d, want 1", result.GradeCounts.Again)
	}
	if result.GradeCounts.Hard != 1 {
		t.Errorf("Hard: got %d, want 1", result.GradeCounts.Hard)
	}
	if result.GradeCounts.Good != 2 {
		t.Errorf("Good: got %d, want 2", result.GradeCounts.Good)
	}
	if result.GradeCounts.Easy != 1 {
		t.Errorf("Easy: got %d, want 1", result.GradeCounts.Easy)
	}
	// AccuracyRate = (2 Good + 1 Easy) / 5 * 100 = 60%
	if result.AccuracyRate != 60.0 {
		t.Errorf("AccuracyRate: got %f, want 60.0", result.AccuracyRate)
	}
	// DurationMs = 15 minutes = 900000ms
	if result.DurationMs != 900000 {
		t.Errorf("DurationMs: got %d, want 900000", result.DurationMs)
	}
}

func TestAggregateSessionResult_Empty(t *testing.T) {
	t.Parallel()

	now := time.Now()
	result := aggregateSessionResult(nil, now.Add(-10*time.Minute), now)

	if result.TotalReviewed != 0 {
		t.Errorf("TotalReviewed: got %d, want 0", result.TotalReviewed)
	}
	if result.AccuracyRate != 0 {
		t.Errorf("AccuracyRate: got %f, want 0", result.AccuracyRate)
	}
}

func TestFilterBatchEntries(t *testing.T) {
	t.Parallel()

	id1 := uuid.New() // exists, no card, has senses → create
	id2 := uuid.New() // exists, has card → skip existing
	id3 := uuid.New() // doesn't exist → error
	id4 := uuid.New() // exists, no card, no senses → skip no senses
	id5 := uuid.New() // exists, no card, has senses → create

	entryIDs := []uuid.UUID{id1, id2, id3, id4, id5}
	existMap := map[uuid.UUID]bool{id1: true, id2: true, id4: true, id5: true}
	cardExistsMap := map[uuid.UUID]bool{id2: true}
	senseCounts := map[uuid.UUID]int{id1: 2, id4: 0, id5: 1}

	toCreate, skippedExisting, skippedNoSenses, errs := filterBatchEntries(entryIDs, existMap, cardExistsMap, senseCounts)

	if len(toCreate) != 2 {
		t.Fatalf("toCreate: got %d, want 2", len(toCreate))
	}
	if toCreate[0] != id1 || toCreate[1] != id5 {
		t.Errorf("toCreate: got %v, want [%v, %v]", toCreate, id1, id5)
	}
	if skippedExisting != 1 {
		t.Errorf("skippedExisting: got %d, want 1", skippedExisting)
	}
	if skippedNoSenses != 1 {
		t.Errorf("skippedNoSenses: got %d, want 1", skippedNoSenses)
	}
	if len(errs) != 1 {
		t.Fatalf("errors: got %d, want 1", len(errs))
	}
	if errs[0].EntryID != id3 {
		t.Errorf("error entry: got %v, want %v", errs[0].EntryID, id3)
	}
}

func TestFilterBatchEntries_AllNotFound(t *testing.T) {
	t.Parallel()

	ids := []uuid.UUID{uuid.New(), uuid.New()}
	toCreate, _, _, errs := filterBatchEntries(ids, map[uuid.UUID]bool{}, nil, nil)

	if len(toCreate) != 0 {
		t.Errorf("toCreate: got %d, want 0", len(toCreate))
	}
	if len(errs) != 2 {
		t.Errorf("errors: got %d, want 2", len(errs))
	}
}
