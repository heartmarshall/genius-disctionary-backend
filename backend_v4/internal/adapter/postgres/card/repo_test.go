package card_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/card"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/testhelper"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// newRepo sets up a test DB and returns a ready Repo + pool.
func newRepo(t *testing.T) (*card.Repo, *pgxpool.Pool) {
	t.Helper()
	pool := testhelper.SetupTestDB(t)
	return card.New(pool), pool
}

// ---------------------------------------------------------------------------
// Create + GetByID
// ---------------------------------------------------------------------------

func TestRepo_Create_AndGetByID(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "create-card-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	created, err := repo.Create(ctx, user.ID, entry.ID, domain.LearningStatusNew, 2.5)
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	if created == nil {
		t.Fatal("Create: expected non-nil result")
	}
	if created.UserID != user.ID {
		t.Errorf("UserID mismatch: got %s, want %s", created.UserID, user.ID)
	}
	if created.EntryID != entry.ID {
		t.Errorf("EntryID mismatch: got %s, want %s", created.EntryID, entry.ID)
	}
	if created.Status != domain.LearningStatusNew {
		t.Errorf("Status mismatch: got %s, want %s", created.Status, domain.LearningStatusNew)
	}
	if created.EaseFactor != 2.5 {
		t.Errorf("EaseFactor mismatch: got %f, want 2.5", created.EaseFactor)
	}
	if created.LearningStep != 0 {
		t.Errorf("LearningStep mismatch: got %d, want 0", created.LearningStep)
	}
	if created.IntervalDays != 0 {
		t.Errorf("IntervalDays mismatch: got %d, want 0", created.IntervalDays)
	}

	// GetByID
	got, err := repo.GetByID(ctx, user.ID, created.ID)
	if err != nil {
		t.Fatalf("GetByID: unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID: expected non-nil result")
	}
	if got.ID != created.ID {
		t.Errorf("GetByID ID mismatch: got %s, want %s", got.ID, created.ID)
	}
	if got.Status != domain.LearningStatusNew {
		t.Errorf("GetByID Status mismatch: got %s, want %s", got.Status, domain.LearningStatusNew)
	}
}

// ---------------------------------------------------------------------------
// Create duplicate entry
// ---------------------------------------------------------------------------

func TestRepo_Create_DuplicateEntry(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "dup-card-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	_, err := repo.Create(ctx, user.ID, entry.ID, domain.LearningStatusNew, 2.5)
	if err != nil {
		t.Fatalf("Create[1]: unexpected error: %v", err)
	}

	_, err = repo.Create(ctx, user.ID, entry.ID, domain.LearningStatusNew, 2.5)
	assertIsDomainError(t, err, domain.ErrAlreadyExists)
}

// ---------------------------------------------------------------------------
// GetByEntryID
// ---------------------------------------------------------------------------

func TestRepo_GetByEntryID(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "getbyentry-card-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry.ID)

	got, err := repo.GetByEntryID(ctx, user.ID, entry.ID)
	if err != nil {
		t.Fatalf("GetByEntryID: unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("GetByEntryID: expected non-nil result")
	}
	if got.EntryID != entry.ID {
		t.Errorf("EntryID mismatch: got %s, want %s", got.EntryID, entry.ID)
	}
	if got.ID != entry.Card.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, entry.Card.ID)
	}
}

func TestRepo_GetByEntryID_NotFound(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)

	_, err := repo.GetByEntryID(ctx, user.ID, uuid.New())
	assertIsDomainError(t, err, domain.ErrNotFound)
}

// ---------------------------------------------------------------------------
// GetDueCards
// ---------------------------------------------------------------------------

func TestRepo_GetDueCards_OverdueFirst(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	now := time.Now().UTC()

	// Card 1: NEW (no next_review_at)
	refEntry1 := testhelper.SeedRefEntry(t, pool, "due1-"+uuid.New().String()[:8])
	entry1 := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry1.ID)

	// Card 2: REVIEW, overdue by 24h
	refEntry2 := testhelper.SeedRefEntry(t, pool, "due2-"+uuid.New().String()[:8])
	entry2 := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry2.ID)
	past := now.Add(-24 * time.Hour)
	_, err := pool.Exec(ctx, `UPDATE cards SET status = 'REVIEW', next_review_at = $1, learning_step = 1 WHERE id = $2`, past, entry2.Card.ID)
	if err != nil {
		t.Fatalf("update card2 to overdue: %v", err)
	}

	// Card 3: LEARNING, overdue by 1h
	refEntry3 := testhelper.SeedRefEntry(t, pool, "due3-"+uuid.New().String()[:8])
	entry3 := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry3.ID)
	pastRecent := now.Add(-1 * time.Hour)
	_, err = pool.Exec(ctx, `UPDATE cards SET status = 'LEARNING', next_review_at = $1, learning_step = 1 WHERE id = $2`, pastRecent, entry3.Card.ID)
	if err != nil {
		t.Fatalf("update card3 to overdue: %v", err)
	}

	cards, err := repo.GetDueCards(ctx, user.ID, now, 10)
	if err != nil {
		t.Fatalf("GetDueCards: unexpected error: %v", err)
	}

	if len(cards) != 3 {
		t.Fatalf("expected 3 due cards, got %d", len(cards))
	}

	// Overdue cards should come first (sorted by next_review_at ASC), then NEW.
	// card2 (overdue 24h, earliest next_review_at) -> card3 (overdue 1h) -> card1 (NEW)
	if cards[0].ID != entry2.Card.ID {
		t.Errorf("expected first card to be entry2 (most overdue), got %s", cards[0].ID)
	}
	if cards[1].ID != entry3.Card.ID {
		t.Errorf("expected second card to be entry3 (less overdue), got %s", cards[1].ID)
	}
	if cards[2].ID != entry1.Card.ID {
		t.Errorf("expected third card to be entry1 (NEW), got %s", cards[2].ID)
	}
}

func TestRepo_GetDueCards_ExcludesSoftDeleted(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	now := time.Now().UTC()

	// Create entry with card, then soft-delete the entry.
	refEntry := testhelper.SeedRefEntry(t, pool, "softdel-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry.ID)

	_, err := pool.Exec(ctx, `UPDATE entries SET deleted_at = now() WHERE id = $1`, entry.ID)
	if err != nil {
		t.Fatalf("soft-delete entry: %v", err)
	}

	cards, err := repo.GetDueCards(ctx, user.ID, now, 10)
	if err != nil {
		t.Fatalf("GetDueCards: unexpected error: %v", err)
	}

	// Card for soft-deleted entry should not appear.
	for _, c := range cards {
		if c.EntryID == entry.ID {
			t.Errorf("expected card for soft-deleted entry %s to be excluded", entry.ID)
		}
	}
}

func TestRepo_GetDueCards_ExcludesMastered(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	now := time.Now().UTC()

	// Create entry with card, then set it to MASTERED.
	refEntry := testhelper.SeedRefEntry(t, pool, "mastered-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry.ID)

	_, err := pool.Exec(ctx, `UPDATE cards SET status = 'MASTERED' WHERE id = $1`, entry.Card.ID)
	if err != nil {
		t.Fatalf("update card to mastered: %v", err)
	}

	cards, err := repo.GetDueCards(ctx, user.ID, now, 10)
	if err != nil {
		t.Fatalf("GetDueCards: unexpected error: %v", err)
	}

	for _, c := range cards {
		if c.ID == entry.Card.ID {
			t.Errorf("expected MASTERED card %s to be excluded", entry.Card.ID)
		}
	}
}

func TestRepo_GetDueCards_RespectsLimit(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	now := time.Now().UTC()

	// Create 3 NEW cards.
	for range 3 {
		refEntry := testhelper.SeedRefEntry(t, pool, "limit-"+uuid.New().String()[:8])
		testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry.ID)
	}

	cards, err := repo.GetDueCards(ctx, user.ID, now, 2)
	if err != nil {
		t.Fatalf("GetDueCards: unexpected error: %v", err)
	}

	if len(cards) != 2 {
		t.Errorf("expected 2 cards (limit), got %d", len(cards))
	}
}

// ---------------------------------------------------------------------------
// CountDue
// ---------------------------------------------------------------------------

func TestRepo_CountDue(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	now := time.Now().UTC()

	// Card 1: NEW
	refEntry1 := testhelper.SeedRefEntry(t, pool, "countdue1-"+uuid.New().String()[:8])
	testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry1.ID)

	// Card 2: REVIEW, overdue
	refEntry2 := testhelper.SeedRefEntry(t, pool, "countdue2-"+uuid.New().String()[:8])
	entry2 := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry2.ID)
	past := now.Add(-24 * time.Hour)
	_, err := pool.Exec(ctx, `UPDATE cards SET status = 'REVIEW', next_review_at = $1, learning_step = 1 WHERE id = $2`, past, entry2.Card.ID)
	if err != nil {
		t.Fatalf("update card2: %v", err)
	}

	// Card 3: MASTERED (should not count)
	refEntry3 := testhelper.SeedRefEntry(t, pool, "countdue3-"+uuid.New().String()[:8])
	entry3 := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry3.ID)
	_, err = pool.Exec(ctx, `UPDATE cards SET status = 'MASTERED' WHERE id = $1`, entry3.Card.ID)
	if err != nil {
		t.Fatalf("update card3: %v", err)
	}

	count, err := repo.CountDue(ctx, user.ID, now)
	if err != nil {
		t.Fatalf("CountDue: unexpected error: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 due cards, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// CountNew
// ---------------------------------------------------------------------------

func TestRepo_CountNew(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)

	// Card 1: NEW
	refEntry1 := testhelper.SeedRefEntry(t, pool, "countnew1-"+uuid.New().String()[:8])
	testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry1.ID)

	// Card 2: NEW
	refEntry2 := testhelper.SeedRefEntry(t, pool, "countnew2-"+uuid.New().String()[:8])
	testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry2.ID)

	// Card 3: REVIEW (should not count)
	refEntry3 := testhelper.SeedRefEntry(t, pool, "countnew3-"+uuid.New().String()[:8])
	entry3 := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry3.ID)
	past := time.Now().UTC().Add(-24 * time.Hour)
	_, err := pool.Exec(ctx, `UPDATE cards SET status = 'REVIEW', next_review_at = $1, learning_step = 1 WHERE id = $2`, past, entry3.Card.ID)
	if err != nil {
		t.Fatalf("update card3: %v", err)
	}

	count, err := repo.CountNew(ctx, user.ID)
	if err != nil {
		t.Fatalf("CountNew: unexpected error: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 new cards, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// CountByStatus
// ---------------------------------------------------------------------------

func TestRepo_CountByStatus(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)

	// Card 1: NEW
	refEntry1 := testhelper.SeedRefEntry(t, pool, "countstat1-"+uuid.New().String()[:8])
	testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry1.ID)

	// Card 2: REVIEW
	refEntry2 := testhelper.SeedRefEntry(t, pool, "countstat2-"+uuid.New().String()[:8])
	entry2 := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry2.ID)
	past := time.Now().UTC().Add(-24 * time.Hour)
	_, err := pool.Exec(ctx, `UPDATE cards SET status = 'REVIEW', next_review_at = $1, learning_step = 1 WHERE id = $2`, past, entry2.Card.ID)
	if err != nil {
		t.Fatalf("update card2: %v", err)
	}

	// Card 3: REVIEW
	refEntry3 := testhelper.SeedRefEntry(t, pool, "countstat3-"+uuid.New().String()[:8])
	entry3 := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry3.ID)
	_, err = pool.Exec(ctx, `UPDATE cards SET status = 'REVIEW', next_review_at = $1, learning_step = 1 WHERE id = $2`, past, entry3.Card.ID)
	if err != nil {
		t.Fatalf("update card3: %v", err)
	}

	counts, err := repo.CountByStatus(ctx, user.ID)
	if err != nil {
		t.Fatalf("CountByStatus: unexpected error: %v", err)
	}

	// Should have NEW=1, REVIEW=2, Total=3
	if counts.New != 1 {
		t.Errorf("expected 1 NEW card, got %d", counts.New)
	}
	if counts.Review != 2 {
		t.Errorf("expected 2 REVIEW cards, got %d", counts.Review)
	}
	if counts.Learning != 0 {
		t.Errorf("expected 0 LEARNING cards, got %d", counts.Learning)
	}
	if counts.Mastered != 0 {
		t.Errorf("expected 0 MASTERED cards, got %d", counts.Mastered)
	}
	if counts.Total != 3 {
		t.Errorf("expected 3 Total cards, got %d", counts.Total)
	}
}

// ---------------------------------------------------------------------------
// UpdateSRS
// ---------------------------------------------------------------------------

func TestRepo_UpdateSRS(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "updatesrs-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry.ID)

	nextReview := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Microsecond)
	params := domain.SRSUpdateParams{
		Status:       domain.LearningStatusReview,
		NextReviewAt: &nextReview,
		IntervalDays: 3,
		EaseFactor:   2.1,
		LearningStep: 2,
	}

	got, err := repo.UpdateSRS(ctx, user.ID, entry.Card.ID, params)
	if err != nil {
		t.Fatalf("UpdateSRS: unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("UpdateSRS: expected non-nil result")
	}
	if got.Status != domain.LearningStatusReview {
		t.Errorf("Status mismatch: got %s, want %s", got.Status, domain.LearningStatusReview)
	}
	if got.NextReviewAt == nil {
		t.Fatal("expected NextReviewAt to be set, got nil")
	}
	if !got.NextReviewAt.Equal(nextReview) {
		t.Errorf("NextReviewAt mismatch: got %v, want %v", got.NextReviewAt, nextReview)
	}
	if got.IntervalDays != 3 {
		t.Errorf("IntervalDays mismatch: got %d, want 3", got.IntervalDays)
	}
	if got.EaseFactor != 2.1 {
		t.Errorf("EaseFactor mismatch: got %f, want 2.1", got.EaseFactor)
	}
	if got.LearningStep != 2 {
		t.Errorf("LearningStep mismatch: got %d, want 2", got.LearningStep)
	}
	if !got.UpdatedAt.After(entry.Card.UpdatedAt) {
		t.Errorf("expected UpdatedAt to be updated after SRS change")
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestRepo_Delete(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "delete-card-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry.ID)

	if err := repo.Delete(ctx, user.ID, entry.Card.ID); err != nil {
		t.Fatalf("Delete: unexpected error: %v", err)
	}

	// Should not be found anymore.
	_, err := repo.GetByID(ctx, user.ID, entry.Card.ID)
	assertIsDomainError(t, err, domain.ErrNotFound)
}

func TestRepo_Delete_NotFound(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)

	err := repo.Delete(ctx, user.ID, uuid.New())
	assertIsDomainError(t, err, domain.ErrNotFound)
}

// ---------------------------------------------------------------------------
// GetByEntryIDs batch
// ---------------------------------------------------------------------------

func TestRepo_GetByEntryIDs_Batch(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)

	refEntry1 := testhelper.SeedRefEntry(t, pool, "batch1-"+uuid.New().String()[:8])
	entry1 := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry1.ID)

	refEntry2 := testhelper.SeedRefEntry(t, pool, "batch2-"+uuid.New().String()[:8])
	entry2 := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry2.ID)

	// Also include a nonexistent entry ID to test it doesn't cause errors.
	cards, err := repo.GetByEntryIDs(ctx, []uuid.UUID{entry1.ID, entry2.ID, uuid.New()})
	if err != nil {
		t.Fatalf("GetByEntryIDs: unexpected error: %v", err)
	}

	if len(cards) != 2 {
		t.Fatalf("expected 2 cards, got %d", len(cards))
	}

	// Verify grouping by entry_id.
	byEntry := make(map[uuid.UUID][]domain.Card)
	for _, c := range cards {
		byEntry[c.EntryID] = append(byEntry[c.EntryID], c)
	}

	if len(byEntry[entry1.ID]) != 1 {
		t.Errorf("expected 1 card for entry1, got %d", len(byEntry[entry1.ID]))
	}
	if len(byEntry[entry2.ID]) != 1 {
		t.Errorf("expected 1 card for entry2, got %d", len(byEntry[entry2.ID]))
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
