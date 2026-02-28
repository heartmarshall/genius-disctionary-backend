# SRS Comprehensive Fixes — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix all 30 issues identified in the SRS audit across FSRS algorithm, database, service, and GraphQL layers.

**Architecture:** Five task groups in dependency order. Group 1 (FSRS pure logic) and Group 4 (GraphQL schema) have no cross-dependencies and can be parallelized. Groups 2→3 are sequential (migration before service). Group 5 is independent.

**Tech Stack:** Go 1.23, PostgreSQL, sqlc, gqlgen, goose migrations

---

## Task 1: FSRS Algorithm Fixes

**Files:**
- Modify: `backend_v4/internal/service/study/fsrs/scheduler.go`
- Modify: `backend_v4/internal/service/study/fsrs/algorithm.go`
- Modify: `backend_v4/internal/service/study/fsrs/fuzz.go`

### Step 1.1: Fix #3 — reviewReview uses pre-update difficulty for all stabilities

In `scheduler.go`, function `reviewReview`, save difficulty BEFORE updating card:

```go
// reviewReview handles REVIEW cards.
func reviewReview(params Parameters, card Card, rating Rating, now time.Time) Card {
	card.Reps++
	card.LastReview = &now

	elapsedDays := card.ElapsedDays
	if elapsedDays < 1 {
		elapsedDays = 1
	}

	r := Retrievability(elapsedDays, card.Stability)

	// Use PRE-UPDATE difficulty for all stability calculations (FSRS-5 spec).
	preD := card.Difficulty

	// Update difficulty with chosen rating (stored on card at end).
	d := NextDifficulty(params.W, card.Difficulty, rating)

	if rating == Again {
		card.Lapses++
		card.State = StateRelearning
		card.Step = 0
		card.Difficulty = d

		newS := StabilityAfterForgettingCapped(params.W, card.Stability, preD, r)
		card.Stability = newS

		steps := params.RelearningSteps
		if len(steps) == 0 {
			steps = []time.Duration{10 * time.Minute}
		}

		card.ElapsedDays = 0
		card.ScheduledDays = 0
		card.Due = now.Add(steps[0])
		return card
	}

	// Compute all recall stabilities using PRE-UPDATE difficulty
	hardS := StabilityAfterRecall(params.W, card.Stability, preD, r, Hard)
	goodS := StabilityAfterRecall(params.W, card.Stability, preD, r, Good)
	easyS := StabilityAfterRecall(params.W, card.Stability, preD, r, Easy)

	// ... rest unchanged, but set card.Difficulty = d after computing stabilities
```

### Step 1.2: Fix #6 — Remove extra Lapses++ in RELEARNING AGAIN

In `scheduler.go`, function `reviewLearning`, remove lines 153-155:

```go
case Again:
	card.Step = 0
	card.ElapsedDays = 0
	card.ScheduledDays = 0
	// REMOVED: if isRelearning { card.Lapses++ }
	// Lapses are only incremented on REVIEW → RELEARNING transition, not during relearning.
	card.Due = now.Add(steps[0])
```

### Step 1.3: Fix #21 — Return error for unknown CardState

Change `ReviewCard` signature to return error:

```go
func ReviewCard(params Parameters, card Card, rating Rating, now time.Time) (Card, error) {
	switch card.State {
	case StateNew:
		return reviewNew(params, card, rating, now), nil
	case StateLearning:
		return reviewLearning(params, card, rating, now, false), nil
	case StateRelearning:
		return reviewLearning(params, card, rating, now, true), nil
	case StateReview:
		return reviewReview(params, card, rating, now), nil
	default:
		return Card{}, fmt.Errorf("unknown card state: %q", card.State)
	}
}
```

Update caller in `review_card.go`:

```go
result, err := fsrs.ReviewCard(params, fsrsCard, rating, now)
if err != nil {
	return nil, fmt.Errorf("fsrs review: %w", err)
}
```

### Step 1.4: Fix #12 — Add weight validation

In `algorithm.go`, add:

```go
// ValidateWeights checks that all 19 FSRS weights are finite and non-NaN.
func ValidateWeights(w [19]float64) error {
	for i, v := range w {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("weight w[%d] is invalid: %v", i, v)
		}
	}
	if w[0] <= 0 || w[1] <= 0 || w[2] <= 0 || w[3] <= 0 {
		return fmt.Errorf("initial stability weights w[0]-w[3] must be positive")
	}
	return nil
}
```

Call it in `ReviewCard` at the top (or better: call once at service startup in `NewService`).

### Step 1.5: Fix #29 — reviewNew Easy: use Good stability for goodInterval baseline

In `scheduler.go`, `reviewNew` Easy case:

```go
case Easy:
	card = graduateToReview(params, card, s, d, now)
	// Use Good stability (not Easy) as the baseline for the minimum interval
	goodS := InitialStability(params.W, Good)
	goodInterval := NextInterval(goodS, params.DesiredRetention)
	goodInterval = clampInterval(goodInterval, params.MaxIntervalDays)
	if card.ScheduledDays <= goodInterval {
		card.ScheduledDays = goodInterval + 1
		card.ScheduledDays = clampInterval(card.ScheduledDays, params.MaxIntervalDays)
		card.Due = now.Add(time.Duration(card.ScheduledDays) * 24 * time.Hour)
	}
```

### Step 1.6: Fix #30 — Improve FuzzSeed

In `fuzz.go`:

```go
import "hash/fnv"

func FuzzSeed(now time.Time, reps int, difficulty, stability float64) int64 {
	h := fnv.New64a()
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(now.Unix()))
	h.Write(b)
	binary.LittleEndian.PutUint64(b, uint64(reps))
	h.Write(b)
	binary.LittleEndian.PutUint64(b, math.Float64bits(difficulty))
	h.Write(b)
	binary.LittleEndian.PutUint64(b, math.Float64bits(stability))
	h.Write(b)
	return int64(h.Sum64())
}
```

### Step 1.7: Commit

```bash
git add backend_v4/internal/service/study/fsrs/
git commit -m "fix(fsrs): correct difficulty ordering, remove extra lapse, validate weights, improve fuzz seed"
```

---

## Task 2: Unified CardState (Issue #13)

**Files:**
- Modify: `backend_v4/internal/service/study/fsrs/scheduler.go`
- Modify: `backend_v4/internal/service/study/review_card.go`
- Modify: `backend_v4/internal/domain/enums.go`

### Step 2.1: Remove duplicate CardState from fsrs package

In `scheduler.go`, remove the `CardState` type and constants. Replace all references with `domain.CardState`:

```go
import "github.com/heartmarshall/myenglish-backend/internal/domain"

// Card holds the FSRS state of a flashcard.
type Card struct {
	State         domain.CardState
	// ... rest unchanged
}
```

Replace all `StateNew` → `domain.CardStateNew`, `StateLearning` → `domain.CardStateLearning`, etc.

### Step 2.2: Simplify review_card.go conversions

Remove the type casts `fsrs.CardState(card.State)` and `domain.CardState(result.State)` — they're now the same type.

### Step 2.3: Commit

```bash
git add backend_v4/internal/service/study/fsrs/ backend_v4/internal/service/study/review_card.go
git commit -m "refactor(fsrs): unify CardState type, use domain.CardState everywhere"
```

---

## Task 3: Database Migration + Repository Fixes (Issues #7, #14, #15, #16, #24)

**Files:**
- Create: `backend_v4/migrations/00019_srs_fixes.sql`
- Modify: `backend_v4/internal/adapter/postgres/card/query/cards.sql`
- Modify: `backend_v4/internal/adapter/postgres/card/repo.go`
- Modify: `backend_v4/internal/adapter/postgres/reviewlog/repo.go`

### Step 3.1: Create migration

```sql
-- +goose Up

-- #24: CHECK constraints on stability and difficulty
ALTER TABLE cards ADD CONSTRAINT chk_stability_nonneg CHECK (stability >= 0);
ALTER TABLE cards ADD CONSTRAINT chk_difficulty_range CHECK (difficulty >= 0 AND difficulty <= 10);

-- #16: Index for NEW cards ordered by created_at
CREATE INDEX ix_cards_new_created ON cards(user_id, created_at) WHERE state = 'NEW';

-- #15: Add user_id to review_logs for efficient user-scoped queries
ALTER TABLE review_logs ADD COLUMN user_id UUID REFERENCES users(id);

-- Backfill user_id from cards
UPDATE review_logs rl SET user_id = c.user_id FROM cards c WHERE rl.card_id = c.id;

-- Make NOT NULL after backfill
ALTER TABLE review_logs ALTER COLUMN user_id SET NOT NULL;

-- #15: Indexes for user-scoped review_log queries
CREATE INDEX ix_review_logs_user_reviewed ON review_logs(user_id, reviewed_at);
CREATE INDEX ix_review_logs_user_card ON review_logs(user_id, card_id);

-- +goose Down
DROP INDEX IF EXISTS ix_review_logs_user_card;
DROP INDEX IF EXISTS ix_review_logs_user_reviewed;
ALTER TABLE review_logs DROP COLUMN IF EXISTS user_id;
DROP INDEX IF EXISTS ix_cards_new_created;
ALTER TABLE cards DROP CONSTRAINT IF EXISTS chk_difficulty_range;
ALTER TABLE cards DROP CONSTRAINT IF EXISTS chk_stability_nonneg;
```

### Step 3.2: Fix #14 — UpdateCardSRS with RETURNING

In `cards.sql`:

```sql
-- name: UpdateCardSRS :one
UPDATE cards
SET state = @state, step = @step, stability = @stability,
    difficulty = @difficulty, due = @due, last_review = @last_review,
    reps = @reps, lapses = @lapses, scheduled_days = @scheduled_days,
    elapsed_days = @elapsed_days, updated_at = now()
WHERE id = @id AND user_id = @user_id
RETURNING id, user_id, entry_id, state, step, stability, difficulty,
          due, last_review, reps, lapses, scheduled_days, elapsed_days,
          created_at, updated_at;
```

Update `repo.go` `UpdateSRS` to use the returned row directly (no extra `GetByID`):

```go
func (r *Repo) UpdateSRS(ctx context.Context, userID, cardID uuid.UUID, params domain.SRSUpdateParams) (*domain.Card, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.UpdateCardSRS(ctx, sqlc.UpdateCardSRSParams{
		ID:            cardID,
		UserID:        userID,
		State:         sqlc.CardState(params.State),
		Step:          int32(params.Step),
		Stability:     params.Stability,
		Difficulty:    params.Difficulty,
		Due:           params.Due,
		LastReview:    params.LastReview,
		Reps:          int32(params.Reps),
		Lapses:        int32(params.Lapses),
		ScheduledDays: int32(params.ScheduledDays),
		ElapsedDays:   int32(params.ElapsedDays),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("card %s: %w", cardID, domain.ErrNotFound)
		}
		return nil, mapError(err, "card", cardID)
	}

	c := toDomainCard(fromUpdateSRSRow(row))
	return &c, nil
}
```

### Step 3.3: Fix #7 — Add user_id to getByEntryIDsSQL

```go
var getByEntryIDsSQL = `
SELECT ` + cardColumns + `
FROM cards c
WHERE c.entry_id = ANY($1::uuid[]) AND c.user_id = $2`
```

Update `GetByEntryIDs` signature to accept `userID`:

```go
func (r *Repo) GetByEntryIDs(ctx context.Context, userID uuid.UUID, entryIDs []uuid.UUID) ([]domain.Card, error) {
	// ...
	rows, err := querier.Query(ctx, getByEntryIDsSQL, entryIDs, userID)
```

Update the DataLoader and callers to pass userID.

### Step 3.4: Fix #15 — Update reviewlog SQL to use user_id directly

Replace JOIN-based user_id filtering with direct column:

```go
const countTodaySQL = `
SELECT count(*) FROM review_logs
WHERE user_id = $1 AND reviewed_at >= $2`

const countNewTodaySQL = `
SELECT count(*) FROM review_logs
WHERE user_id = $1 AND reviewed_at >= $2
AND prev_state IS NOT NULL
AND prev_state->>'state' = 'NEW'`

const getByPeriodSQL = `
SELECT id, card_id, grade, prev_state, duration_ms, reviewed_at
FROM review_logs
WHERE user_id = $1 AND reviewed_at >= $2 AND reviewed_at <= $3
ORDER BY reviewed_at DESC`
```

Also update `CreateReviewLog` in `review_logs.sql` to include `user_id`:

```sql
-- name: CreateReviewLog :one
INSERT INTO review_logs (id, card_id, user_id, grade, prev_state, duration_ms, reviewed_at)
VALUES (@id, @card_id, @user_id, @grade, @prev_state, @duration_ms, @reviewed_at)
RETURNING id, card_id, user_id, grade, prev_state, duration_ms, reviewed_at;
```

Propagate `user_id` through `domain.ReviewLog`, `Create` method, and all callers.

### Step 3.5: Run sqlc generate

```bash
cd backend_v4 && make generate
```

### Step 3.6: Commit

```bash
git add backend_v4/migrations/ backend_v4/internal/adapter/postgres/
git commit -m "fix(db): add review_logs.user_id, RETURNING on UpdateSRS, indexes, CHECK constraints"
```

---

## Task 4: Service Layer Race Condition Fixes (Issues #1, #2, #8)

**Files:**
- Modify: `backend_v4/internal/service/study/review_card.go`
- Modify: `backend_v4/internal/service/study/undo_review.go`
- Modify: `backend_v4/internal/service/study/session.go`
- Modify: `backend_v4/internal/adapter/postgres/card/repo.go`

### Step 4.1: Add GetByIDForUpdate to card repo

Add a new raw SQL query and method:

```go
var getByIDForUpdateSQL = `
SELECT ` + cardColumns + `
FROM cards c
WHERE c.id = $1 AND c.user_id = $2
FOR UPDATE`

func (r *Repo) GetByIDForUpdate(ctx context.Context, userID, cardID uuid.UUID) (*domain.Card, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)
	row := querier.QueryRow(ctx, getByIDForUpdateSQL, cardID, userID)
	// scan same as scanCardFromRows but for single row
	var c domain.Card
	// ... scan all fields
	return &c, nil
}
```

Add to the `cardRepo` interface in `service.go`:

```go
GetByIDForUpdate(ctx context.Context, userID, cardID uuid.UUID) (*domain.Card, error)
```

### Step 4.2: Fix #2 — ReviewCard: move card read inside transaction

```go
func (s *Service) ReviewCard(ctx context.Context, input ReviewCardInput) (*domain.Card, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}
	if err := input.Validate(); err != nil {
		return nil, err
	}

	now := time.Now()

	// Load settings (outside tx — read-only, no lock needed)
	settings, err := s.settings.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}

	params := fsrs.Parameters{ /* ... same as before ... */ }

	rating := mapGradeToRating(input.Grade)

	var updatedCard *domain.Card

	err = s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Lock card row inside transaction
		card, cardErr := s.cards.GetByIDForUpdate(txCtx, userID, input.CardID)
		if cardErr != nil {
			return fmt.Errorf("get card: %w", cardErr)
		}

		// Snapshot, convert, compute elapsed, run FSRS — all inside tx
		snapshot := &domain.CardSnapshot{ /* ... from card ... */ }
		fsrsCard := fsrs.Card{ /* ... from card ... */ }

		if card.LastReview != nil {
			elapsed := now.Sub(*card.LastReview)
			fsrsCard.ElapsedDays = max(0, int(elapsed.Hours()/24))
		}

		result, fsrsErr := fsrs.ReviewCard(params, fsrsCard, rating, now)
		if fsrsErr != nil {
			return fmt.Errorf("fsrs review: %w", fsrsErr)
		}

		// Update card, create log, audit — same as before
		var updateErr error
		updatedCard, updateErr = s.cards.UpdateSRS(txCtx, userID, card.ID, /* ... */)
		if updateErr != nil {
			return fmt.Errorf("update card: %w", updateErr)
		}

		_, logErr := s.reviews.Create(txCtx, &domain.ReviewLog{ /* ... */ })
		if logErr != nil {
			return fmt.Errorf("create review log: %w", logErr)
		}

		auditErr := s.audit.Log(txCtx, /* ... */)
		if auditErr != nil {
			return fmt.Errorf("audit log: %w", auditErr)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return updatedCard, nil
}
```

### Step 4.3: Fix #1 — UndoReview: move reads inside transaction

Same pattern — move `GetByIDForUpdate` and `GetLastByCardID` inside `RunInTx`:

```go
func (s *Service) UndoReview(ctx context.Context, input UndoReviewInput) (*domain.Card, error) {
	// ... validation ...

	now := time.Now()
	var restoredCard *domain.Card

	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Lock card row
		card, cardErr := s.cards.GetByIDForUpdate(txCtx, userID, input.CardID)
		if cardErr != nil {
			return fmt.Errorf("get card: %w", cardErr)
		}

		lastLog, logErr := s.reviews.GetLastByCardID(txCtx, input.CardID)
		if logErr != nil {
			if errors.Is(logErr, domain.ErrNotFound) {
				return domain.NewValidationError("card_id", "card has no reviews to undo")
			}
			return fmt.Errorf("get last review: %w", logErr)
		}

		if lastLog.PrevState == nil {
			return domain.NewValidationError("review", "review cannot be undone")
		}

		undoWindow := time.Duration(s.srsConfig.UndoWindowMinutes) * time.Minute
		if now.Sub(lastLog.ReviewedAt) > undoWindow {
			return domain.NewValidationError("review", "undo window expired")
		}

		// Restore, delete log, audit — inside tx
		// ...
		return nil
	})

	return restoredCard, err
}
```

### Step 4.4: Fix #8 — FinishSession: wrap in transaction

```go
func (s *Service) FinishSession(ctx context.Context, input FinishSessionInput) (*domain.StudySession, error) {
	// ... validation ...

	session, err := s.sessions.GetByID(ctx, userID, input.SessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	if session.Status != domain.SessionStatusActive {
		return nil, domain.NewValidationError("session", "session already finished")
	}

	now := time.Now()
	var finishedSession *domain.StudySession

	err = s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		logs, logErr := s.reviews.GetByPeriod(txCtx, userID, session.StartedAt, now)
		if logErr != nil {
			return fmt.Errorf("get review logs: %w", logErr)
		}

		// Aggregate stats... same logic
		result := domain.SessionResult{ /* ... */ }

		var finErr error
		finishedSession, finErr = s.sessions.Finish(txCtx, userID, session.ID, result)
		return finErr
	})

	return finishedSession, err
}
```

### Step 4.5: Commit

```bash
git add backend_v4/internal/service/study/ backend_v4/internal/adapter/postgres/card/
git commit -m "fix(study): eliminate race conditions in ReviewCard, UndoReview, FinishSession with SELECT FOR UPDATE"
```

---

## Task 5: Timezone Fix (Issue #5)

**Files:**
- Modify: `backend_v4/internal/adapter/postgres/reviewlog/repo.go`
- Modify: `backend_v4/internal/service/study/service.go` (interface)
- Modify: `backend_v4/internal/service/study/dashboard.go`

### Step 5.1: Pass timezone string to GetStreakDays

Update SQL:

```go
const getStreakDaysSQL = `
SELECT
    date_trunc('day', rl.reviewed_at AT TIME ZONE $4)::date AS review_date,
    count(*) AS review_count
FROM review_logs rl
WHERE rl.user_id = $1 AND rl.reviewed_at >= $2
GROUP BY review_date
ORDER BY review_date DESC
LIMIT $3`
```

Update `GetStreakDays` to accept `timezone string`:

```go
func (r *Repo) GetStreakDays(ctx context.Context, userID uuid.UUID, dayStart time.Time, lastNDays int, timezone string) ([]domain.DayReviewCount, error) {
	from := dayStart.AddDate(0, 0, -lastNDays)
	rows, err := querier.Query(ctx, getStreakDaysSQL, userID, from, lastNDays, timezone)
	// ...
}
```

Update interface and dashboard caller to pass `settings.Timezone`.

### Step 5.2: Commit

```bash
git add backend_v4/internal/adapter/postgres/reviewlog/ backend_v4/internal/service/study/
git commit -m "fix(study): use user timezone for streak day grouping in GetStreakDays"
```

---

## Task 6: Dashboard + Domain Fixes (Issues #17, #20)

**Files:**
- Modify: `backend_v4/internal/domain/study.go`
- Modify: `backend_v4/internal/service/study/dashboard.go`
- Modify: `backend_v4/internal/transport/graphql/resolver/study.resolvers.go`

### Step 6.1: Fix #17 — Dashboard carries full StudySession

In `domain/study.go`:

```go
type Dashboard struct {
	DueCount      int
	NewCount      int
	ReviewedToday int
	NewToday      int
	Streak        int
	StatusCounts  CardStatusCounts
	OverdueCount  int
	ActiveSession *StudySession // Changed from *uuid.UUID
}
```

Where `StudySession` is imported from `domain/card.go` (it's `domain.StudySession`).

In `dashboard.go`, update to store the full session:

```go
dashboard := domain.Dashboard{
	// ...
	ActiveSession: activeSession, // already *domain.StudySession
}
```

In `study.resolvers.go`, simplify the field resolver:

```go
func (r *dashboardResolver) ActiveSession(ctx context.Context, obj *domain.Dashboard) (*domain.StudySession, error) {
	return obj.ActiveSession, nil
}
```

### Step 6.2: Commit

```bash
git add backend_v4/internal/domain/ backend_v4/internal/service/study/ backend_v4/internal/transport/graphql/
git commit -m "fix(study): carry full StudySession in Dashboard, remove double-fetch"
```

---

## Task 7: GraphQL Schema + Resolver Fixes (Issues #4, #9, #10, #11, #18, #19, #25, #26, #27, #28)

**Files:**
- Modify: `backend_v4/internal/transport/graphql/schema/study.graphql`
- Modify: `backend_v4/internal/transport/graphql/resolver/study.resolvers.go`
- Modify: `backend_v4/internal/transport/graphql/errpresenter.go`
- Modify: `backend_v4/internal/service/study/input.go`

### Step 7.1: Update study.graphql

```graphql
type Card {
  id: UUID!
  entryId: UUID!
  state: CardState!
  step: Int!
  stability: Float!
  difficulty: Float!
  due: DateTime!
  lastReview: DateTime
  scheduledDays: Int!
  reps: Int!
  lapses: Int!
  createdAt: DateTime!
  updatedAt: DateTime!
}

type ReviewLog {
  id: UUID!
  cardId: UUID!
  grade: ReviewGrade!
  prevState: CardSnapshotOutput
  durationMs: Int
  reviewedAt: DateTime!
}

type CardSnapshotOutput {
  state: CardState!
  step: Int!
  stability: Float!
  difficulty: Float!
  scheduledDays: Int!
}

type SessionResult {
  totalReviews: Int!
  newReviewed: Int!
  dueReviewed: Int!
  gradeCounts: GradeCounts!
  totalDurationMs: Int!
  accuracyRate: Float!
}

type Dashboard {
  dueCount: Int!
  newCount: Int!
  reviewedToday: Int!
  newToday: Int!
  streak: Int!
  statusCounts: CardStatusCounts!
  overdueCount: Int!
  activeSession: StudySession
}

type CardStatusCounts {
  new: Int!
  learning: Int!
  review: Int!
  relearning: Int!
  total: Int!
}

type CardStats {
  totalReviews: Int!
  averageDurationMs: Int!
  accuracy: Float!
  currentState: CardState!
  stability: Float!
  difficulty: Float!
  scheduledDays: Int!
  gradeDistribution: GradeCounts
}

type BatchCreateCardsPayload {
  createdCount: Int!
  skippedExisting: Int!
  skippedNoSenses: Int!
  errors: [BatchCreateCardError!]!
}

type CardHistoryPayload {
  logs: [ReviewLog!]!
  totalCount: Int!
}

extend type Query {
  studyQueue(limit: Int): [DictionaryEntry!]!
  dashboard: Dashboard!
  cardHistory(input: GetCardHistoryInput!): CardHistoryPayload!
  cardStats(cardId: UUID!): CardStats!
}

extend type Mutation {
  reviewCard(input: ReviewCardInput!): ReviewCardPayload!
  undoReview(cardId: UUID!): UndoReviewPayload!
  createCard(entryId: UUID!): CreateCardPayload!
  deleteCard(id: UUID!): DeleteCardPayload!
  batchCreateCards(entryIds: [UUID!]!): BatchCreateCardsPayload!
  startStudySession: StartSessionPayload!
  finishStudySession: FinishSessionPayload!
  abandonStudySession: AbandonSessionPayload!
}
```

Key changes:
- **#25**: Added `step`, `lastReview` to Card
- **#9**: Added `newToday` to Dashboard
- **#11**: Added `currentState`, `stability`, `difficulty`, `scheduledDays` to CardStats; `gradeDistribution` nullable
- **#10**: SessionResult: renamed `averageDurationMs` → `totalDurationMs`, added `newReviewed`, `dueReviewed`, `accuracyRate`
- **#19**: `cardHistory` returns `CardHistoryPayload` with `totalCount`
- **#18**: `finishStudySession` no longer requires input
- **#26**: `BatchCreateCardsPayload` splits skip reasons
- **#4**: `gradeDistribution: GradeCounts` (nullable, no `!`)
- **CardStatusCounts**: added `total`

### Step 7.2: Fix #18 — finishStudySession without required session ID

In `session.go`, add `FinishActiveSession`:

```go
func (s *Service) FinishActiveSession(ctx context.Context) (*domain.StudySession, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	session, err := s.sessions.GetActive(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get active session: %w", err)
	}

	return s.finishSession(ctx, userID, session)
}
```

Extract the common logic into a `finishSession` private method. Remove `FinishSessionInput`.

### Step 7.3: Fix #27 — Align studyQueue default limit

In `study.resolvers.go`:

```go
func (r *queryResolver) StudyQueue(ctx context.Context, limit *int) ([]*domain.Entry, error) {
	// ... auth check ...
	l := 50 // align with service default
	if limit != nil {
		l = *limit
	}
	// ...
}
```

### Step 7.4: Fix #28 — Error presenter

In `errpresenter.go`, use `err` consistently for both `errors.Is` and `errors.As`:

```go
switch {
case errors.Is(err, domain.ErrNotFound):
	gqlErr.Extensions = map[string]interface{}{"code": "NOT_FOUND"}

case errors.Is(err, domain.ErrAlreadyExists):
	gqlErr.Extensions = map[string]interface{}{"code": "ALREADY_EXISTS"}

case errors.Is(err, domain.ErrValidation):
	gqlErr.Extensions = map[string]interface{}{"code": "VALIDATION"}
	var ve *domain.ValidationError
	if errors.As(err, &ve) {
		gqlErr.Extensions["fields"] = ve.Errors
	}

case errors.Is(err, domain.ErrUnauthorized):
	gqlErr.Extensions = map[string]interface{}{"code": "UNAUTHENTICATED"}

case errors.Is(err, domain.ErrForbidden):
	gqlErr.Extensions = map[string]interface{}{"code": "FORBIDDEN"}

case errors.Is(err, domain.ErrConflict):
	gqlErr.Extensions = map[string]interface{}{"code": "CONFLICT"}

default:
	// ...
}
```

Remove the `origErr` variable entirely — `errors.Is` and `errors.As` already traverse the full error chain.

### Step 7.5: Run gqlgen + fix resolvers

```bash
cd backend_v4 && make generate
```

Then update all resolvers in `study.resolvers.go` to match the new schema.

### Step 7.6: Commit

```bash
git add backend_v4/internal/transport/graphql/ backend_v4/internal/service/study/ backend_v4/internal/domain/
git commit -m "fix(graphql): add missing fields, fix nullable types, align defaults, fix error presenter"
```

---

## Task 8: Low Priority Cleanup (Issue #23)

**Files:**
- Modify: `backend_v4/migrations/00018_fsrs_migration.sql`

### Step 8.1: Fix migration down partial index filter

This is documentation-only since we won't rollback, but fix for correctness:

```sql
-- In Down section, replace:
CREATE INDEX ix_cards_user_due ON cards(user_id, status, next_review_at);
-- With:
CREATE INDEX ix_cards_user_due ON cards(user_id, status, next_review_at) WHERE status != 'MASTERED';
```

### Step 8.2: Commit

```bash
git add backend_v4/migrations/00018_fsrs_migration.sql
git commit -m "fix(migration): restore partial index filter in 00018 down path"
```

---

## Verification

After all tasks:

```bash
cd backend_v4
make generate        # regenerate sqlc + gqlgen
make build           # verify compilation
make test            # unit tests
make test-e2e        # E2E tests (requires Docker)
make lint            # golangci-lint
```
