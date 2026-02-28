package reviewlog_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/reviewlog"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/testhelper"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// newRepo sets up a test DB and returns a ready Repo + pool.
func newRepo(t *testing.T) (*reviewlog.Repo, *pgxpool.Pool) {
	t.Helper()
	pool := testhelper.SetupTestDB(t)
	return reviewlog.New(pool), pool
}

// seedCard creates a user, ref_entry, entry, and card for testing.
// Returns the card and the pool.
func seedCard(t *testing.T, pool *pgxpool.Pool) (domain.User, domain.Card) {
	t.Helper()
	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "rl-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry.ID)
	return user, *entry.Card
}

// buildReviewLog creates a domain.ReviewLog for testing.
func buildReviewLog(cardID uuid.UUID, grade domain.ReviewGrade, prevState *domain.CardSnapshot, durationMs *int) domain.ReviewLog {
	return domain.ReviewLog{
		ID:         uuid.New(),
		CardID:     cardID,
		Grade:      grade,
		PrevState:  prevState,
		DurationMs: durationMs,
		ReviewedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
}

// ---------------------------------------------------------------------------
// Create + prev_state serialization
// ---------------------------------------------------------------------------

func TestRepo_Create_WithPrevState(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	_, card := seedCard(t, pool)

	dueTime := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Microsecond)
	lastReview := time.Now().UTC().Truncate(time.Microsecond)
	prevState := &domain.CardSnapshot{
		State:         domain.CardStateLearning,
		Step:          2,
		Stability:     4.5,
		Difficulty:    5.3,
		Due:           dueTime,
		LastReview:    &lastReview,
		Reps:          3,
		Lapses:        1,
		ScheduledDays: 1,
		ElapsedDays:   0,
	}
	durationMs := 3500
	input := buildReviewLog(card.ID, domain.ReviewGradeGood, prevState, &durationMs)

	got, err := repo.Create(ctx, &input)
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	if got.ID != input.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, input.ID)
	}
	if got.CardID != card.ID {
		t.Errorf("CardID mismatch: got %s, want %s", got.CardID, card.ID)
	}
	if got.Grade != domain.ReviewGradeGood {
		t.Errorf("Grade mismatch: got %s, want %s", got.Grade, domain.ReviewGradeGood)
	}
	if got.DurationMs == nil || *got.DurationMs != 3500 {
		t.Errorf("DurationMs mismatch: got %v, want 3500", got.DurationMs)
	}
	if !got.ReviewedAt.Equal(input.ReviewedAt) {
		t.Errorf("ReviewedAt mismatch: got %v, want %v", got.ReviewedAt, input.ReviewedAt)
	}

	// Verify prev_state round-trip.
	if got.PrevState == nil {
		t.Fatal("PrevState should not be nil")
	}
	if got.PrevState.State != domain.CardStateLearning {
		t.Errorf("PrevState.State mismatch: got %s, want %s", got.PrevState.State, domain.CardStateLearning)
	}
	if got.PrevState.Step != 2 {
		t.Errorf("PrevState.Step mismatch: got %d, want 2", got.PrevState.Step)
	}
	if got.PrevState.Stability != 4.5 {
		t.Errorf("PrevState.Stability mismatch: got %f, want 4.5", got.PrevState.Stability)
	}
	if got.PrevState.Difficulty != 5.3 {
		t.Errorf("PrevState.Difficulty mismatch: got %f, want 5.3", got.PrevState.Difficulty)
	}
	if !got.PrevState.Due.Equal(dueTime) {
		t.Errorf("PrevState.Due mismatch: got %v, want %v", got.PrevState.Due, dueTime)
	}
	if got.PrevState.LastReview == nil {
		t.Fatal("PrevState.LastReview should not be nil")
	}
	if !got.PrevState.LastReview.Equal(lastReview) {
		t.Errorf("PrevState.LastReview mismatch: got %v, want %v", got.PrevState.LastReview, lastReview)
	}
}

func TestRepo_Create_NilPrevState(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	_, card := seedCard(t, pool)

	input := buildReviewLog(card.ID, domain.ReviewGradeAgain, nil, nil)

	got, err := repo.Create(ctx, &input)
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	if got.PrevState != nil {
		t.Errorf("PrevState should be nil for first review, got %+v", got.PrevState)
	}
	if got.DurationMs != nil {
		t.Errorf("DurationMs should be nil, got %v", got.DurationMs)
	}
}

// ---------------------------------------------------------------------------
// GetByCardID
// ---------------------------------------------------------------------------

func TestRepo_GetByCardID(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	_, card := seedCard(t, pool)

	// Create 3 review logs with staggered times.
	var createdIDs []uuid.UUID
	for i := range 3 {
		rl := buildReviewLog(card.ID, domain.ReviewGradeGood, nil, nil)
		rl.ReviewedAt = time.Now().UTC().Truncate(time.Microsecond).Add(time.Duration(i) * time.Second)

		created, err := repo.Create(ctx, &rl)
		if err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
		createdIDs = append(createdIDs, created.ID)
	}

	got, _, err := repo.GetByCardID(ctx, card.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetByCardID: unexpected error: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 logs, got %d", len(got))
	}

	// Verify descending order by reviewed_at.
	for i := 1; i < len(got); i++ {
		if got[i].ReviewedAt.After(got[i-1].ReviewedAt) {
			t.Errorf("logs not in DESC order: [%d].ReviewedAt=%s > [%d].ReviewedAt=%s",
				i, got[i].ReviewedAt, i-1, got[i-1].ReviewedAt)
		}
	}

	// Most recent should be first (index 2 was created last).
	if got[0].ID != createdIDs[2] {
		t.Errorf("expected first log to be most recent (ID=%s), got %s", createdIDs[2], got[0].ID)
	}
}

// ---------------------------------------------------------------------------
// GetLastByCardID
// ---------------------------------------------------------------------------

func TestRepo_GetLastByCardID(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	_, card := seedCard(t, pool)

	// Create 2 review logs.
	rl1 := buildReviewLog(card.ID, domain.ReviewGradeHard, nil, nil)
	rl1.ReviewedAt = time.Now().UTC().Truncate(time.Microsecond).Add(-1 * time.Hour)
	if _, err := repo.Create(ctx, &rl1); err != nil {
		t.Fatalf("Create[1]: %v", err)
	}

	rl2 := buildReviewLog(card.ID, domain.ReviewGradeEasy, nil, nil)
	rl2.ReviewedAt = time.Now().UTC().Truncate(time.Microsecond)
	created2, err := repo.Create(ctx, &rl2)
	if err != nil {
		t.Fatalf("Create[2]: %v", err)
	}

	got, err := repo.GetLastByCardID(ctx, card.ID)
	if err != nil {
		t.Fatalf("GetLastByCardID: unexpected error: %v", err)
	}

	if got.ID != created2.ID {
		t.Errorf("expected most recent log (ID=%s), got %s", created2.ID, got.ID)
	}
	if got.Grade != domain.ReviewGradeEasy {
		t.Errorf("Grade mismatch: got %s, want %s", got.Grade, domain.ReviewGradeEasy)
	}
}

func TestRepo_GetLastByCardID_NotFound(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	_, card := seedCard(t, pool)

	_, err := repo.GetLastByCardID(ctx, card.ID)
	assertIsDomainError(t, err, domain.ErrNotFound)
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestRepo_Delete(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	_, card := seedCard(t, pool)

	input := buildReviewLog(card.ID, domain.ReviewGradeGood, nil, nil)
	created, err := repo.Create(ctx, &input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: unexpected error: %v", err)
	}

	// Should not be findable anymore.
	got, _, err := repo.GetByCardID(ctx, card.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetByCardID after delete: %v", err)
	}
	for _, rl := range got {
		if rl.ID == created.ID {
			t.Errorf("expected deleted review log %s to be absent", created.ID)
		}
	}
}

func TestRepo_Delete_NotFound(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	err := repo.Delete(ctx, uuid.New())
	assertIsDomainError(t, err, domain.ErrNotFound)
}

// ---------------------------------------------------------------------------
// CountToday
// ---------------------------------------------------------------------------

func TestRepo_CountToday(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user, card := seedCard(t, pool)

	now := time.Now().UTC()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Create 2 logs today.
	for range 2 {
		rl := buildReviewLog(card.ID, domain.ReviewGradeGood, nil, nil)
		rl.ReviewedAt = now.Truncate(time.Microsecond)
		if _, err := repo.Create(ctx, &rl); err != nil {
			t.Fatalf("Create today: %v", err)
		}
	}

	// Create 1 log yesterday via raw SQL.
	yesterday := dayStart.Add(-1 * time.Hour)
	_, err := pool.Exec(ctx,
		`INSERT INTO review_logs (id, card_id, grade, reviewed_at) VALUES ($1, $2, $3, $4)`,
		uuid.New(), card.ID, "GOOD", yesterday,
	)
	if err != nil {
		t.Fatalf("insert yesterday log: %v", err)
	}

	count, err := repo.CountToday(ctx, user.ID, dayStart)
	if err != nil {
		t.Fatalf("CountToday: unexpected error: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 reviews today, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// GetStreakDays
// ---------------------------------------------------------------------------

func TestRepo_GetStreakDays(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user, card := seedCard(t, pool)

	now := time.Now().UTC()

	// Create logs on 3 different days.
	days := []time.Time{
		now.Truncate(time.Microsecond),                               // today
		now.Add(-24 * time.Hour).Truncate(time.Microsecond),          // yesterday
		now.Add(-48 * time.Hour).Truncate(time.Microsecond),          // 2 days ago
	}

	for i, reviewedAt := range days {
		// Create 2 logs per day to test counting.
		for j := range 2 {
			_, err := pool.Exec(ctx,
				`INSERT INTO review_logs (id, card_id, grade, reviewed_at) VALUES ($1, $2, $3, $4)`,
				uuid.New(), card.ID, "GOOD", reviewedAt.Add(time.Duration(j)*time.Minute),
			)
			if err != nil {
				t.Fatalf("insert log day=%d, j=%d: %v", i, j, err)
			}
		}
	}

	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	counts, err := repo.GetStreakDays(ctx, user.ID, dayStart, 10, "UTC")
	if err != nil {
		t.Fatalf("GetStreakDays: unexpected error: %v", err)
	}

	if len(counts) != 3 {
		t.Fatalf("expected 3 days, got %d", len(counts))
	}

	// Verify descending order by date.
	for i := 1; i < len(counts); i++ {
		if counts[i].Date.After(counts[i-1].Date) {
			t.Errorf("days not in DESC order: [%d].Date=%s > [%d].Date=%s",
				i, counts[i].Date, i-1, counts[i-1].Date)
		}
	}

	// Each day should have 2 reviews.
	for i, dc := range counts {
		if dc.Count != 2 {
			t.Errorf("day[%d]: expected count 2, got %d", i, dc.Count)
		}
	}
}

// ---------------------------------------------------------------------------
// GetByCardIDs batch
// ---------------------------------------------------------------------------

func TestRepo_GetByCardIDs_Batch(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	_, card1 := seedCard(t, pool)
	_, card2 := seedCard(t, pool)

	// Create 2 logs for card1 and 1 for card2.
	for range 2 {
		rl := buildReviewLog(card1.ID, domain.ReviewGradeGood, nil, nil)
		if _, err := repo.Create(ctx, &rl); err != nil {
			t.Fatalf("Create card1: %v", err)
		}
	}
	rl := buildReviewLog(card2.ID, domain.ReviewGradeHard, nil, nil)
	if _, err := repo.Create(ctx, &rl); err != nil {
		t.Fatalf("Create card2: %v", err)
	}

	got, err := repo.GetByCardIDs(ctx, []uuid.UUID{card1.ID, card2.ID})
	if err != nil {
		t.Fatalf("GetByCardIDs: unexpected error: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 logs total, got %d", len(got))
	}

	// Group by card_id and verify counts.
	byCard := make(map[uuid.UUID][]reviewlog.ReviewLogWithCardID)
	for _, rl := range got {
		byCard[rl.CardID] = append(byCard[rl.CardID], rl)
	}

	if len(byCard[card1.ID]) != 2 {
		t.Errorf("expected 2 logs for card1, got %d", len(byCard[card1.ID]))
	}
	if len(byCard[card2.ID]) != 1 {
		t.Errorf("expected 1 log for card2, got %d", len(byCard[card2.ID]))
	}
}

func TestRepo_GetByCardIDs_Empty(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	got, err := repo.GetByCardIDs(ctx, []uuid.UUID{})
	if err != nil {
		t.Fatalf("GetByCardIDs empty: unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("result should not be nil (empty input should return empty slice)")
	}
	if len(got) != 0 {
		t.Errorf("expected 0 logs, got %d", len(got))
	}
}

// ---------------------------------------------------------------------------
// PrevState serialization edge cases
// ---------------------------------------------------------------------------

func TestRepo_PrevState_WithDue(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	_, card := seedCard(t, pool)

	dueTime := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	lastReview := time.Date(2025, 6, 8, 10, 30, 0, 0, time.UTC)
	prevState := &domain.CardSnapshot{
		State:         domain.CardStateReview,
		Step:          0,
		Stability:     15.2,
		Difficulty:    5.8,
		Due:           dueTime,
		LastReview:    &lastReview,
		Reps:          5,
		Lapses:        0,
		ScheduledDays: 7,
		ElapsedDays:   7,
	}
	input := buildReviewLog(card.ID, domain.ReviewGradeEasy, prevState, nil)

	created, err := repo.Create(ctx, &input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Read back via GetLastByCardID to verify round-trip.
	got, err := repo.GetLastByCardID(ctx, card.ID)
	if err != nil {
		t.Fatalf("GetLastByCardID: %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, created.ID)
	}
	if got.PrevState == nil {
		t.Fatal("PrevState should not be nil")
	}
	if !got.PrevState.Due.Equal(dueTime) {
		t.Errorf("PrevState.Due mismatch: got %v, want %v", got.PrevState.Due, dueTime)
	}
	if got.PrevState.State != domain.CardStateReview {
		t.Errorf("PrevState.State mismatch: got %s, want %s", got.PrevState.State, domain.CardStateReview)
	}
	if got.PrevState.ScheduledDays != 7 {
		t.Errorf("PrevState.ScheduledDays mismatch: got %d, want 7", got.PrevState.ScheduledDays)
	}
	if got.PrevState.Stability != 15.2 {
		t.Errorf("PrevState.Stability mismatch: got %f, want 15.2", got.PrevState.Stability)
	}
}

func TestRepo_PrevState_NewCard(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	_, card := seedCard(t, pool)

	prevState := &domain.CardSnapshot{
		State:         domain.CardStateNew,
		Step:          0,
		Stability:     0,
		Difficulty:    0,
		Due:           time.Time{},
		LastReview:    nil,
		Reps:          0,
		Lapses:        0,
		ScheduledDays: 0,
		ElapsedDays:   0,
	}
	input := buildReviewLog(card.ID, domain.ReviewGradeAgain, prevState, nil)

	created, err := repo.Create(ctx, &input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Read back to verify round-trip.
	got, err := repo.GetLastByCardID(ctx, card.ID)
	if err != nil {
		t.Fatalf("GetLastByCardID: %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, created.ID)
	}
	if got.PrevState == nil {
		t.Fatal("PrevState should not be nil")
	}
	if got.PrevState.LastReview != nil {
		t.Errorf("PrevState.LastReview should be nil, got %v", got.PrevState.LastReview)
	}
	if got.PrevState.State != domain.CardStateNew {
		t.Errorf("PrevState.State mismatch: got %s, want %s", got.PrevState.State, domain.CardStateNew)
	}
	if got.PrevState.Stability != 0 {
		t.Errorf("PrevState.Stability mismatch: got %f, want 0", got.PrevState.Stability)
	}
}

// ---------------------------------------------------------------------------
// CountNewToday and GetByPeriod tests (Task 10c)
// ---------------------------------------------------------------------------

func TestRepo_CountNewToday_OnlyCountsNewState(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user, card := seedCard(t, pool)

	// Review with PrevState.State = NEW (should be counted)
	rl1 := buildReviewLog(card.ID, domain.ReviewGradeGood, &domain.CardSnapshot{
		State: domain.CardStateNew,
		Due:   time.Now().UTC(),
	}, nil)
	_, err := repo.Create(ctx, &rl1)
	if err != nil {
		t.Fatalf("Create rl1: %v", err)
	}

	// Review with PrevState.State = REVIEW (should NOT be counted)
	rl2 := buildReviewLog(card.ID, domain.ReviewGradeGood, &domain.CardSnapshot{
		State: domain.CardStateReview,
		Due:   time.Now().UTC(),
	}, nil)
	_, err = repo.Create(ctx, &rl2)
	if err != nil {
		t.Fatalf("Create rl2: %v", err)
	}

	dayStart := time.Now().Add(-24 * time.Hour)
	count, err := repo.CountNewToday(ctx, user.ID, dayStart)
	if err != nil {
		t.Fatalf("CountNewToday: %v", err)
	}
	if count != 1 {
		t.Errorf("CountNewToday: got %d, want 1 (only NEW prev_state reviews)", count)
	}
}

func TestRepo_GetByPeriod_FiltersByTimeRange(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user, card := seedCard(t, pool)
	now := time.Now().UTC().Truncate(time.Microsecond)

	// Review 3 days ago (outside range)
	rl1 := domain.ReviewLog{
		ID:         uuid.New(),
		CardID:     card.ID,
		Grade:      domain.ReviewGradeGood,
		ReviewedAt: now.Add(-72 * time.Hour),
	}
	_, err := repo.Create(ctx, &rl1)
	if err != nil {
		t.Fatalf("Create rl1: %v", err)
	}

	// Review 1 hour ago (inside range)
	rl2 := domain.ReviewLog{
		ID:         uuid.New(),
		CardID:     card.ID,
		Grade:      domain.ReviewGradeEasy,
		ReviewedAt: now.Add(-1 * time.Hour),
	}
	_, err = repo.Create(ctx, &rl2)
	if err != nil {
		t.Fatalf("Create rl2: %v", err)
	}

	// Query last 24 hours
	from := now.Add(-24 * time.Hour)
	to := now.Add(1 * time.Hour)
	logs, err := repo.GetByPeriod(ctx, user.ID, from, to)
	if err != nil {
		t.Fatalf("GetByPeriod: %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("GetByPeriod: got %d logs, want 1 (only recent review)", len(logs))
	}
	if len(logs) == 1 && logs[0].ID != rl2.ID {
		t.Errorf("GetByPeriod returned wrong log: got %v, want %v", logs[0].ID, rl2.ID)
	}
}

// ---------------------------------------------------------------------------
// JSONB key dependency guard (Task 9)
// ---------------------------------------------------------------------------

// TestRepo_CountNewToday_MatchesJSONBKey verifies that CountNewToday correctly
// matches review logs where prev_state.state = 'NEW'. This guards the JSONB key
// dependency between countNewTodaySQL and cardSnapshotJSON.State's json tag.
func TestRepo_CountNewToday_MatchesJSONBKey(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user, card := seedCard(t, pool)

	// Create a review log with PrevState.State = "NEW"
	rl := buildReviewLog(card.ID, domain.ReviewGradeGood, &domain.CardSnapshot{
		State: domain.CardStateNew,
		Due:   time.Now().UTC(),
	}, nil)
	_, err := repo.Create(ctx, &rl)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	dayStart := time.Now().Add(-24 * time.Hour)
	count, err := repo.CountNewToday(ctx, user.ID, dayStart)
	if err != nil {
		t.Fatalf("CountNewToday: %v", err)
	}
	if count != 1 {
		t.Errorf("CountNewToday: got %d, want 1 (should find review where prev_state.state = 'NEW')", count)
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func assertIsDomainError(t *testing.T, err error, target error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error wrapping %v, got nil", target)
	}
	if !errors.Is(err, target) {
		t.Fatalf("expected error wrapping %v, got: %v", target, err)
	}
}
