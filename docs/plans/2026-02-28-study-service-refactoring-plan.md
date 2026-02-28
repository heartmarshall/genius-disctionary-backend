# Study Service Refactoring Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor study service to extract pure functions, add conversion layer, and eliminate boilerplate while preserving the external API.

**Architecture:** Single Service struct preserved. Business logic extracted into pure functions testable without mocks. Conversion layer centralizes Card/FSRS/Snapshot field mapping. BatchCreateCards collapsed into single transaction.

**Tech Stack:** Go, existing test patterns (manual mocks in mocks_test.go, t.Parallel, table-driven subtests)

**Design doc:** `docs/plans/2026-02-28-study-service-refactoring-design.md`

---

### Task 1: Create `convert.go` — Conversion Layer + Helpers

**Files:**
- Create: `backend_v4/internal/service/study/convert.go`
- Test: `backend_v4/internal/service/study/convert_test.go`

**Step 1: Write tests for conversion functions**

Create `convert_test.go` with tests for all 5 functions:

```go
package study

import (
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
```

**Step 2: Run tests to verify they fail**

Run: `cd backend_v4 && go test ./internal/service/study/ -run "TestCardToFSRS|TestFsrsResult|TestSnapshot|TestComputeElapsed|TestUserID|TestBuildFSRS" -v -count=1`
Expected: FAIL — functions not defined.

**Step 3: Write implementation**

Create `convert.go`:

```go
package study

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/service/study/fsrs"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// cardToFSRS converts a domain Card to an fsrs.Card for scheduling calculations.
func cardToFSRS(card *domain.Card) fsrs.Card {
	return fsrs.Card{
		State:         card.State,
		Step:          card.Step,
		Stability:     card.Stability,
		Difficulty:    card.Difficulty,
		Due:           card.Due,
		LastReview:    card.LastReview,
		Reps:          card.Reps,
		Lapses:        card.Lapses,
		ScheduledDays: card.ScheduledDays,
		ElapsedDays:   card.ElapsedDays,
	}
}

// fsrsResultToUpdateParams converts an FSRS scheduling result to domain update params.
func fsrsResultToUpdateParams(result fsrs.Card) domain.SRSUpdateParams {
	var lastReview *time.Time
	if result.LastReview != nil {
		t := *result.LastReview
		lastReview = &t
	}

	return domain.SRSUpdateParams{
		State:         result.State,
		Step:          result.Step,
		Stability:     result.Stability,
		Difficulty:    result.Difficulty,
		Due:           result.Due,
		LastReview:    lastReview,
		Reps:          result.Reps,
		Lapses:        result.Lapses,
		ScheduledDays: result.ScheduledDays,
		ElapsedDays:   result.ElapsedDays,
	}
}

// snapshotFromCard captures the current SRS state of a card before mutation.
func snapshotFromCard(card *domain.Card) *domain.CardSnapshot {
	return &domain.CardSnapshot{
		State:         card.State,
		Step:          card.Step,
		Stability:     card.Stability,
		Difficulty:    card.Difficulty,
		Due:           card.Due,
		LastReview:    card.LastReview,
		Reps:          card.Reps,
		Lapses:        card.Lapses,
		ScheduledDays: card.ScheduledDays,
		ElapsedDays:   card.ElapsedDays,
	}
}

// computeElapsedDays calculates whole days elapsed since the last review.
func computeElapsedDays(lastReview *time.Time, now time.Time) int {
	if lastReview == nil {
		return 0
	}
	return max(0, int(now.Sub(*lastReview).Hours()/24))
}

// userID extracts the authenticated user's ID from context.
func (s *Service) userID(ctx context.Context) (uuid.UUID, error) {
	uid, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return uuid.Nil, domain.ErrUnauthorized
	}
	return uid, nil
}

// buildFSRSParams merges global SRS config with per-user settings into FSRS parameters.
func (s *Service) buildFSRSParams(settings *domain.UserSettings) fsrs.Parameters {
	return fsrs.Parameters{
		W:                s.fsrsWeights,
		DesiredRetention: settings.DesiredRetention,
		MaxIntervalDays:  min(s.srsConfig.MaxIntervalDays, settings.MaxIntervalDays),
		EnableFuzz:       s.srsConfig.EnableFuzz,
		LearningSteps:    s.srsConfig.LearningSteps,
		RelearningSteps:  s.srsConfig.RelearningSteps,
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `cd backend_v4 && go test ./internal/service/study/ -run "TestCardToFSRS|TestFsrsResult|TestSnapshot|TestComputeElapsed|TestUserID|TestBuildFSRS" -v -count=1`
Expected: PASS (all 8 tests)

**Step 5: Run ALL existing tests to verify nothing broke**

Run: `cd backend_v4 && go test ./internal/service/study/... -race -count=1`
Expected: PASS (all 115+ tests). New file is additive — no existing code changed.

**Step 6: Commit**

```bash
git add backend_v4/internal/service/study/convert.go backend_v4/internal/service/study/convert_test.go
git commit -m "refactor(study): add conversion layer and helper functions

Extract cardToFSRS, fsrsResultToUpdateParams, snapshotFromCard,
computeElapsedDays, userID, and buildFSRSParams helpers.
Preparation for eliminating field-copy duplication across service methods."
```

---

### Task 2: Extract `aggregateSessionResult` Pure Function

**Files:**
- Modify: `backend_v4/internal/service/study/session.go:135-170`
- Test: `backend_v4/internal/service/study/convert_test.go` (add tests)

**Step 1: Write test for aggregateSessionResult**

Add to `convert_test.go`:

```go
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
```

**Step 2: Run tests to verify they fail**

Run: `cd backend_v4 && go test ./internal/service/study/ -run "TestAggregateSession" -v -count=1`
Expected: FAIL — `aggregateSessionResult` not defined.

**Step 3: Add function to `convert.go` and refactor `session.go`**

Add to `convert.go`:

```go
// aggregateSessionResult computes session statistics from review logs.
func aggregateSessionResult(logs []*domain.ReviewLog, startedAt, now time.Time) domain.SessionResult {
	totalReviewed := len(logs)
	newReviewed := 0
	gradeCounts := domain.GradeCounts{}

	for _, log := range logs {
		if log.PrevState != nil && log.PrevState.State == domain.CardStateNew {
			newReviewed++
		}
		switch log.Grade {
		case domain.ReviewGradeAgain:
			gradeCounts.Again++
		case domain.ReviewGradeHard:
			gradeCounts.Hard++
		case domain.ReviewGradeGood:
			gradeCounts.Good++
		case domain.ReviewGradeEasy:
			gradeCounts.Easy++
		}
	}

	accuracyRate := 0.0
	if totalReviewed > 0 {
		accuracyRate = float64(gradeCounts.Good+gradeCounts.Easy) / float64(totalReviewed) * 100
	}

	return domain.SessionResult{
		TotalReviewed: totalReviewed,
		NewReviewed:   newReviewed,
		DueReviewed:   totalReviewed - newReviewed,
		GradeCounts:   gradeCounts,
		DurationMs:    now.Sub(startedAt).Milliseconds(),
		AccuracyRate:  accuracyRate,
	}
}
```

Then replace lines 135-170 of `session.go` (inside finishSession's tx callback) with:

```go
	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		logs, logErr := s.reviews.GetByPeriod(txCtx, userID, session.StartedAt, now)
		if logErr != nil {
			return fmt.Errorf("get review logs: %w", logErr)
		}

		result := aggregateSessionResult(logs, session.StartedAt, now)

		var finErr error
		finishedSession, finErr = s.sessions.Finish(txCtx, userID, session.ID, result)
		return finErr
	})
```

**Step 4: Run ALL tests**

Run: `cd backend_v4 && go test ./internal/service/study/... -race -count=1`
Expected: PASS (all existing session tests + 2 new tests)

**Step 5: Commit**

```bash
git add backend_v4/internal/service/study/convert.go backend_v4/internal/service/study/convert_test.go backend_v4/internal/service/study/session.go
git commit -m "refactor(study): extract aggregateSessionResult as pure function

Move session statistics computation from finishSession tx callback
into a standalone pure function. Testable without mocks."
```

---

### Task 3: Extract `filterBatchEntries` Pure Function

**Files:**
- Modify: `backend_v4/internal/service/study/card_crud.go:148-206`
- Test: `backend_v4/internal/service/study/convert_test.go` (add tests)

**Step 1: Write test for filterBatchEntries**

Add to `convert_test.go`:

```go
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
```

**Step 2: Run tests to verify they fail**

Run: `cd backend_v4 && go test ./internal/service/study/ -run "TestFilterBatch" -v -count=1`
Expected: FAIL

**Step 3: Add function to `convert.go` and refactor `card_crud.go`**

Add to `convert.go`:

```go
// filterBatchEntries categorizes entry IDs for batch card creation.
// Returns entries ready for creation, skip counts, and errors for not-found entries.
func filterBatchEntries(
	entryIDs []uuid.UUID,
	existMap map[uuid.UUID]bool,
	cardExistsMap map[uuid.UUID]bool,
	senseCounts map[uuid.UUID]int,
) (toCreate []uuid.UUID, skippedExisting int, skippedNoSenses int, errors []BatchCreateError) {
	// Phase 1: filter to existing entries
	var existing []uuid.UUID
	for _, id := range entryIDs {
		if exists, ok := existMap[id]; !ok || !exists {
			errors = append(errors, BatchCreateError{EntryID: id, Reason: "entry not found"})
		} else {
			existing = append(existing, id)
		}
	}

	// Phase 2: filter out entries that already have cards
	var withoutCards []uuid.UUID
	for _, id := range existing {
		if has, ok := cardExistsMap[id]; ok && has {
			skippedExisting++
		} else {
			withoutCards = append(withoutCards, id)
		}
	}

	// Phase 3: filter out entries without senses
	for _, id := range withoutCards {
		if cnt, ok := senseCounts[id]; !ok || cnt == 0 {
			skippedNoSenses++
		} else {
			toCreate = append(toCreate, id)
		}
	}

	return toCreate, skippedExisting, skippedNoSenses, errors
}
```

Then replace the 3 filter loops in `BatchCreateCards` (lines 148-206 of `card_crud.go`) with:

```go
	// Check which entries exist
	existMap, err := s.entries.ExistByIDs(ctx, userID, input.EntryIDs)
	if err != nil {
		return result, fmt.Errorf("check entries exist: %w", err)
	}

	// Check which entries already have cards
	cardExistsMap, err := s.cards.ExistsByEntryIDs(ctx, userID, input.EntryIDs)
	if err != nil {
		return result, fmt.Errorf("check cards exist: %w", err)
	}

	// Batch count senses (eliminates N+1)
	senseCounts, err := s.senses.CountByEntryIDs(ctx, input.EntryIDs)
	if err != nil {
		return result, fmt.Errorf("count senses batch: %w", err)
	}

	toCreate, result.SkippedExisting, result.SkippedNoSenses, result.Errors = filterBatchEntries(
		input.EntryIDs, existMap, cardExistsMap, senseCounts,
	)

	if len(toCreate) == 0 {
		return result, nil
	}
```

Note: The 3 pre-checks (ExistByIDs, ExistsByEntryIDs, CountByEntryIDs) now all receive `input.EntryIDs` (the full list), not filtered subsets. This is slightly less optimal (counting senses for non-existent entries), but simplifies the code and the DB queries are cheap. The filtering logic handles it correctly.

**Step 4: Run ALL tests**

Run: `cd backend_v4 && go test ./internal/service/study/... -race -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add backend_v4/internal/service/study/convert.go backend_v4/internal/service/study/convert_test.go backend_v4/internal/service/study/card_crud.go
git commit -m "refactor(study): extract filterBatchEntries as pure function

Move batch entry categorization logic into standalone function.
Simplifies BatchCreateCards and enables direct unit testing
of edge cases without mocks."
```

---

### Task 4: Refactor `ReviewCard` to Use Conversion Layer

**Files:**
- Modify: `backend_v4/internal/service/study/review_card.go`

**Step 1: Refactor ReviewCard to use helpers**

Replace the current `ReviewCard` method with version using conversion helpers.
Key changes:
- `ctxutil.UserIDFromCtx` → `s.userID(ctx)`
- Inline FSRS params → `s.buildFSRSParams(settings)`
- Manual snapshot construction → `snapshotFromCard(card)`
- Manual fsrs.Card construction → `cardToFSRS(card)` + `computeElapsedDays`
- Manual SRSUpdateParams → `fsrsResultToUpdateParams(result)`

The full refactored method:

```go
func (s *Service) ReviewCard(ctx context.Context, input ReviewCardInput) (*domain.Card, error) {
	userID, err := s.userID(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	now := s.clock.Now()

	settings, err := s.settings.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}

	params := s.buildFSRSParams(settings)
	rating := mapGradeToRating(input.Grade)

	var updatedCard *domain.Card

	err = s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		card, cardErr := s.cards.GetByIDForUpdate(txCtx, userID, input.CardID)
		if cardErr != nil {
			return fmt.Errorf("get card: %w", cardErr)
		}

		snapshot := snapshotFromCard(card)

		fsrsCard := cardToFSRS(card)
		fsrsCard.ElapsedDays = computeElapsedDays(card.LastReview, now)

		result, fsrsErr := fsrs.ReviewCard(params, fsrsCard, rating, now)
		if fsrsErr != nil {
			return fmt.Errorf("fsrs review: %w", fsrsErr)
		}

		var updateErr error
		updatedCard, updateErr = s.cards.UpdateSRS(txCtx, userID, card.ID, fsrsResultToUpdateParams(result))
		if updateErr != nil {
			return fmt.Errorf("update card: %w", updateErr)
		}

		_, logErr := s.reviews.Create(txCtx, &domain.ReviewLog{
			ID:         uuid.New(),
			CardID:     card.ID,
			UserID:     userID,
			Grade:      input.Grade,
			PrevState:  snapshot,
			DurationMs: input.DurationMs,
			ReviewedAt: now,
		})
		if logErr != nil {
			return fmt.Errorf("create review log: %w", logErr)
		}

		auditErr := s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeCard,
			EntityID:   &card.ID,
			Action:     domain.AuditActionUpdate,
			Changes: map[string]any{
				"grade": map[string]any{"new": input.Grade},
				"state": map[string]any{
					"old": card.State,
					"new": updatedCard.State,
				},
				"stability": map[string]any{
					"old": card.Stability,
					"new": updatedCard.Stability,
				},
			},
		})
		if auditErr != nil {
			return fmt.Errorf("audit log: %w", auditErr)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if updatedCard == nil {
		return nil, fmt.Errorf("card update failed: no result returned")
	}

	s.log.InfoContext(ctx, "card reviewed",
		slog.String("user_id", userID.String()),
		slog.String("card_id", input.CardID.String()),
		slog.String("grade", string(input.Grade)),
		slog.String("new_state", string(updatedCard.State)),
		slog.Float64("stability", updatedCard.Stability),
	)

	return updatedCard, nil
}
```

**Step 2: Run ALL tests**

Run: `cd backend_v4 && go test ./internal/service/study/... -race -count=1`
Expected: PASS (all 115+ tests). Behavior is identical — only internal implementation changed.

**Step 3: Commit**

```bash
git add backend_v4/internal/service/study/review_card.go
git commit -m "refactor(study): simplify ReviewCard using conversion layer

Replace manual field-by-field copying with cardToFSRS,
snapshotFromCard, fsrsResultToUpdateParams, computeElapsedDays,
and buildFSRSParams helpers."
```

---

### Task 5: Refactor `UndoReview` to Use Conversion Layer

**Files:**
- Modify: `backend_v4/internal/service/study/undo_review.go`

**Step 1: Refactor UndoReview**

Key changes:
- `ctxutil.UserIDFromCtx` → `s.userID(ctx)`
- Manual `SRSUpdateParams{ps.State, ps.Step, ...}` → inline (keep as-is: UndoReview restores from snapshot, not from fsrs result, so `fsrsResultToUpdateParams` doesn't apply here; the snapshot→params mapping is inherently different)

Actually, UndoReview maps `CardSnapshot → SRSUpdateParams`. These have the same fields. Add a new helper to `convert.go`:

```go
// snapshotToUpdateParams converts a CardSnapshot back to SRSUpdateParams for restoration.
func snapshotToUpdateParams(ps *domain.CardSnapshot) domain.SRSUpdateParams {
	return domain.SRSUpdateParams{
		State:         ps.State,
		Step:          ps.Step,
		Stability:     ps.Stability,
		Difficulty:    ps.Difficulty,
		Due:           ps.Due,
		LastReview:    ps.LastReview,
		Reps:          ps.Reps,
		Lapses:        ps.Lapses,
		ScheduledDays: ps.ScheduledDays,
		ElapsedDays:   ps.ElapsedDays,
	}
}
```

Add test to `convert_test.go`:

```go
func TestSnapshotToUpdateParams(t *testing.T) {
	t.Parallel()

	now := time.Now()
	lastReview := now.Add(-24 * time.Hour)

	snap := &domain.CardSnapshot{
		State: domain.CardStateLearning, Step: 1,
		Stability: 3.0, Difficulty: 6.0,
		Due: now, LastReview: &lastReview,
		Reps: 5, Lapses: 1,
		ScheduledDays: 2, ElapsedDays: 1,
	}

	params := snapshotToUpdateParams(snap)

	if params.State != snap.State {
		t.Errorf("State: got %v, want %v", params.State, snap.State)
	}
	if params.Stability != snap.Stability {
		t.Errorf("Stability: got %f, want %f", params.Stability, snap.Stability)
	}
}
```

Then refactor `undo_review.go` to use `s.userID(ctx)` and `snapshotToUpdateParams(ps)`.

**Step 2: Run ALL tests**

Run: `cd backend_v4 && go test ./internal/service/study/... -race -count=1`
Expected: PASS

**Step 3: Commit**

```bash
git add backend_v4/internal/service/study/convert.go backend_v4/internal/service/study/convert_test.go backend_v4/internal/service/study/undo_review.go
git commit -m "refactor(study): simplify UndoReview using conversion helpers

Replace manual snapshot-to-params field copying with
snapshotToUpdateParams. Use s.userID() helper."
```

---

### Task 6: Apply `s.userID()` Helper to All Remaining Methods

**Files:**
- Modify: `backend_v4/internal/service/study/study_queue.go`
- Modify: `backend_v4/internal/service/study/study_queue_entries.go`
- Modify: `backend_v4/internal/service/study/session.go`
- Modify: `backend_v4/internal/service/study/card_crud.go`
- Modify: `backend_v4/internal/service/study/dashboard.go`

**Step 1: Replace all `ctxutil.UserIDFromCtx` calls with `s.userID(ctx)`**

In each file, replace:
```go
userID, ok := ctxutil.UserIDFromCtx(ctx)
if !ok {
    return ..., domain.ErrUnauthorized
}
```
with:
```go
userID, err := s.userID(ctx)
if err != nil {
    return ..., err
}
```

Methods to update (12 remaining after Tasks 4-5):
- `GetStudyQueue` (study_queue.go)
- `GetStudyQueueEntries` (study_queue_entries.go)
- `GetActiveSession`, `StartSession`, `FinishActiveSession`, `FinishSession`, `AbandonSession` (session.go)
- `CreateCard`, `DeleteCard`, `BatchCreateCards` (card_crud.go)
- `GetDashboard`, `GetCardHistory`, `GetCardStats` (dashboard.go)

After replacing, remove unused `ctxutil` imports from files that no longer use it directly. Keep the import in `convert.go` where `userID()` is defined.

**Step 2: Run ALL tests**

Run: `cd backend_v4 && go test ./internal/service/study/... -race -count=1`
Expected: PASS

**Step 3: Verify no unused imports**

Run: `cd backend_v4 && go vet ./internal/service/study/...`
Expected: No errors. If unused imports exist, remove them.

**Step 4: Commit**

```bash
git add backend_v4/internal/service/study/
git commit -m "refactor(study): replace ctxutil.UserIDFromCtx with s.userID() helper

Centralize user ID extraction into single helper method across
all 16 public service methods."
```

---

### Task 7: Collapse BatchCreateCards into Single Transaction

**Files:**
- Modify: `backend_v4/internal/service/study/card_crud.go:208-241`

**Step 1: Replace per-item transactions with single batch transaction**

Replace the for-loop-with-individual-txs (current lines 208-241) with:

```go
	// Create all cards in a single transaction
	err = s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		for _, entryID := range toCreate {
			createdCard, createErr := s.cards.Create(txCtx, userID, entryID)
			if createErr != nil {
				result.Errors = append(result.Errors, BatchCreateError{
					EntryID: entryID,
					Reason:  createErr.Error(),
				})
				continue
			}
			result.Created++

			auditErr := s.audit.Log(txCtx, domain.AuditRecord{
				UserID:     userID,
				EntityType: domain.EntityTypeCard,
				EntityID:   &createdCard.ID,
				Action:     domain.AuditActionCreate,
				Changes: map[string]any{
					"entry_id": map[string]any{"new": entryID},
				},
			})
			if auditErr != nil {
				return fmt.Errorf("audit log: %w", auditErr)
			}
		}
		return nil
	})
	if err != nil {
		return result, fmt.Errorf("batch create cards: %w", err)
	}
```

**Step 2: Run ALL tests**

Run: `cd backend_v4 && go test ./internal/service/study/... -race -count=1`
Expected: PASS. Some existing batch tests may need adjustment if they mock tx per-item. Check test output carefully.

**Step 3: Commit**

```bash
git add backend_v4/internal/service/study/card_crud.go
git commit -m "perf(study): collapse BatchCreateCards into single transaction

Replace N individual transactions with one batch transaction.
Reduces DB round-trips from O(N) to O(1) for up to 100 cards."
```

---

### Task 8: Final Verification and Cleanup

**Files:**
- All modified files in `backend_v4/internal/service/study/`

**Step 1: Run full test suite**

Run: `cd backend_v4 && go test ./internal/service/study/... -race -count=1 -v`
Expected: PASS (all tests)

**Step 2: Run linter**

Run: `cd backend_v4 && make lint`
Expected: No new lint errors in study package.

**Step 3: Run build**

Run: `cd backend_v4 && make build`
Expected: Successful build.

**Step 4: Verify no unused imports or dead code**

Run: `cd backend_v4 && go vet ./internal/service/study/...`
Expected: Clean.

**Step 5: Check that `ctxutil` import was removed from files that no longer need it**

Files that should still import `ctxutil`: only `convert.go`.
Files that should NOT import `ctxutil` anymore: `review_card.go`, `undo_review.go`, `study_queue.go`, `study_queue_entries.go`, `session.go`, `card_crud.go`, `dashboard.go`.

Verify: `grep -r "ctxutil" backend_v4/internal/service/study/*.go`
Expected: Only `convert.go` matches.

**Step 6: Run integration/E2E tests if available**

Run: `cd backend_v4 && make test-e2e` (requires Docker)
Expected: PASS — external API unchanged.
