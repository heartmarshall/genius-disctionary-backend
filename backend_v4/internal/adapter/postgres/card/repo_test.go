package card_test

import (
	"context"
	"errors"
	"fmt"
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

	created, err := repo.Create(ctx, user.ID, entry.ID)
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
	if created.State != domain.CardStateNew {
		t.Errorf("State mismatch: got %s, want %s", created.State, domain.CardStateNew)
	}
	if created.Stability != 0 {
		t.Errorf("Stability mismatch: got %f, want 0", created.Stability)
	}
	if created.Step != 0 {
		t.Errorf("Step mismatch: got %d, want 0", created.Step)
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
	if got.State != domain.CardStateNew {
		t.Errorf("GetByID State mismatch: got %s, want %s", got.State, domain.CardStateNew)
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

	_, err := repo.Create(ctx, user.ID, entry.ID)
	if err != nil {
		t.Fatalf("Create[1]: unexpected error: %v", err)
	}

	_, err = repo.Create(ctx, user.ID, entry.ID)
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

	// Card 1: LEARNING, due 1h ago
	refEntry1 := testhelper.SeedRefEntry(t, pool, "due1-"+uuid.New().String()[:8])
	entry1 := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry1.ID)
	past1h := now.Add(-1 * time.Hour)
	_, err := pool.Exec(ctx, `UPDATE cards SET state = 'LEARNING', due = $1, step = 1 WHERE id = $2`, past1h, entry1.Card.ID)
	if err != nil {
		t.Fatalf("update card1: %v", err)
	}

	// Card 2: REVIEW, overdue by 24h
	refEntry2 := testhelper.SeedRefEntry(t, pool, "due2-"+uuid.New().String()[:8])
	entry2 := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry2.ID)
	past24h := now.Add(-24 * time.Hour)
	_, err = pool.Exec(ctx, `UPDATE cards SET state = 'REVIEW', due = $1, stability = 5.0, reps = 3 WHERE id = $2`, past24h, entry2.Card.ID)
	if err != nil {
		t.Fatalf("update card2: %v", err)
	}

	cards, err := repo.GetDueCards(ctx, user.ID, now, 10)
	if err != nil {
		t.Fatalf("GetDueCards: unexpected error: %v", err)
	}

	if len(cards) != 2 {
		t.Fatalf("expected 2 due cards, got %d", len(cards))
	}

	// Most overdue first (sorted by due ASC)
	if cards[0].ID != entry2.Card.ID {
		t.Errorf("expected first card to be entry2 (most overdue), got %s", cards[0].ID)
	}
	if cards[1].ID != entry1.Card.ID {
		t.Errorf("expected second card to be entry1 (less overdue), got %s", cards[1].ID)
	}
}

func TestRepo_GetDueCards_ExcludesSoftDeleted(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	now := time.Now().UTC()

	// Create entry with card (state=LEARNING, due in past), then soft-delete the entry.
	refEntry := testhelper.SeedRefEntry(t, pool, "softdel-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry.ID)

	past := now.Add(-1 * time.Hour)
	_, err := pool.Exec(ctx, `UPDATE cards SET state = 'LEARNING', due = $1 WHERE id = $2`, past, entry.Card.ID)
	if err != nil {
		t.Fatalf("update card: %v", err)
	}

	_, err = pool.Exec(ctx, `UPDATE entries SET deleted_at = now() WHERE id = $1`, entry.ID)
	if err != nil {
		t.Fatalf("soft-delete entry: %v", err)
	}

	cards, err := repo.GetDueCards(ctx, user.ID, now, 10)
	if err != nil {
		t.Fatalf("GetDueCards: unexpected error: %v", err)
	}

	for _, c := range cards {
		if c.EntryID == entry.ID {
			t.Errorf("expected card for soft-deleted entry %s to be excluded", entry.ID)
		}
	}
}

func TestRepo_GetDueCards_ExcludesNew(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	now := time.Now().UTC()

	// Create entry with card in state NEW — should not appear in due cards.
	refEntry := testhelper.SeedRefEntry(t, pool, "new-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry.ID)

	cards, err := repo.GetDueCards(ctx, user.ID, now, 10)
	if err != nil {
		t.Fatalf("GetDueCards: unexpected error: %v", err)
	}

	for _, c := range cards {
		if c.ID == entry.Card.ID {
			t.Errorf("expected NEW card %s to be excluded from due cards", entry.Card.ID)
		}
	}
}

func TestRepo_GetDueCards_RespectsLimit(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	now := time.Now().UTC()

	// Create 3 LEARNING cards with due in the past.
	for range 3 {
		refEntry := testhelper.SeedRefEntry(t, pool, "limit-"+uuid.New().String()[:8])
		e := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry.ID)
		past := now.Add(-1 * time.Hour)
		_, err := pool.Exec(ctx, `UPDATE cards SET state = 'LEARNING', due = $1 WHERE id = $2`, past, e.Card.ID)
		if err != nil {
			t.Fatalf("update card: %v", err)
		}
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
// CountOverdue
// ---------------------------------------------------------------------------

func TestRepo_CountOverdue(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	now := time.Now().UTC()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Card 1: REVIEW, due yesterday (overdue)
	ref1 := testhelper.SeedRefEntry(t, pool, "overdue-"+uuid.New().String()[:8])
	e1 := testhelper.SeedEntryWithCard(t, pool, user.ID, ref1.ID)
	_, err := pool.Exec(ctx, `UPDATE cards SET state = 'REVIEW', due = $1, stability = 5.0, reps = 3 WHERE id = $2`,
		dayStart.AddDate(0, 0, -1), e1.Card.ID)
	if err != nil {
		t.Fatalf("update card1: %v", err)
	}

	// Card 2: REVIEW, due in 1 hour (due but not overdue)
	ref2 := testhelper.SeedRefEntry(t, pool, "duesoon-"+uuid.New().String()[:8])
	e2 := testhelper.SeedEntryWithCard(t, pool, user.ID, ref2.ID)
	_, err = pool.Exec(ctx, `UPDATE cards SET state = 'REVIEW', due = $1, stability = 5.0, reps = 3 WHERE id = $2`,
		now.Add(time.Hour), e2.Card.ID)
	if err != nil {
		t.Fatalf("update card2: %v", err)
	}

	// Card 3: NEW (not overdue regardless of due)
	ref3 := testhelper.SeedRefEntry(t, pool, "newcard-"+uuid.New().String()[:8])
	testhelper.SeedEntryWithCard(t, pool, user.ID, ref3.ID)

	count, err := repo.CountOverdue(ctx, user.ID, dayStart)
	if err != nil {
		t.Fatalf("CountOverdue: unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 overdue card, got %d", count)
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

	// Card 1: REVIEW, overdue
	refEntry1 := testhelper.SeedRefEntry(t, pool, "countdue1-"+uuid.New().String()[:8])
	entry1 := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry1.ID)
	past := now.Add(-24 * time.Hour)
	_, err := pool.Exec(ctx, `UPDATE cards SET state = 'REVIEW', due = $1, stability = 5.0, reps = 3 WHERE id = $2`, past, entry1.Card.ID)
	if err != nil {
		t.Fatalf("update card1: %v", err)
	}

	// Card 2: LEARNING, overdue
	refEntry2 := testhelper.SeedRefEntry(t, pool, "countdue2-"+uuid.New().String()[:8])
	entry2 := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry2.ID)
	_, err = pool.Exec(ctx, `UPDATE cards SET state = 'LEARNING', due = $1 WHERE id = $2`, past, entry2.Card.ID)
	if err != nil {
		t.Fatalf("update card2: %v", err)
	}

	// Card 3: NEW (should not count as due)
	refEntry3 := testhelper.SeedRefEntry(t, pool, "countdue3-"+uuid.New().String()[:8])
	testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry3.ID)

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
	_, err := pool.Exec(ctx, `UPDATE cards SET state = 'REVIEW', due = $1, stability = 5.0, reps = 3 WHERE id = $2`, past, entry3.Card.ID)
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
	_, err := pool.Exec(ctx, `UPDATE cards SET state = 'REVIEW', due = $1, stability = 5.0, reps = 3 WHERE id = $2`, past, entry2.Card.ID)
	if err != nil {
		t.Fatalf("update card2: %v", err)
	}

	// Card 3: REVIEW
	refEntry3 := testhelper.SeedRefEntry(t, pool, "countstat3-"+uuid.New().String()[:8])
	entry3 := testhelper.SeedEntryWithCard(t, pool, user.ID, refEntry3.ID)
	_, err = pool.Exec(ctx, `UPDATE cards SET state = 'REVIEW', due = $1, stability = 5.0, reps = 3 WHERE id = $2`, past, entry3.Card.ID)
	if err != nil {
		t.Fatalf("update card3: %v", err)
	}

	counts, err := repo.CountByStatus(ctx, user.ID)
	if err != nil {
		t.Fatalf("CountByStatus: unexpected error: %v", err)
	}

	if counts.New != 1 {
		t.Errorf("expected 1 NEW card, got %d", counts.New)
	}
	if counts.Review != 2 {
		t.Errorf("expected 2 REVIEW cards, got %d", counts.Review)
	}
	if counts.Learning != 0 {
		t.Errorf("expected 0 LEARNING cards, got %d", counts.Learning)
	}
	if counts.Relearning != 0 {
		t.Errorf("expected 0 RELEARNING cards, got %d", counts.Relearning)
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

	now := time.Now().UTC().Truncate(time.Microsecond)
	due := now.Add(3 * 24 * time.Hour)
	params := domain.SRSUpdateParams{
		State:         domain.CardStateReview,
		Step:          0,
		Stability:     5.5,
		Difficulty:    4.2,
		Due:           due,
		LastReview:    &now,
		Reps:          1,
		Lapses:        0,
		ScheduledDays: 3,
		ElapsedDays:   0,
	}

	got, err := repo.UpdateSRS(ctx, user.ID, entry.Card.ID, params)
	if err != nil {
		t.Fatalf("UpdateSRS: unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("UpdateSRS: expected non-nil result")
	}
	if got.State != domain.CardStateReview {
		t.Errorf("State mismatch: got %s, want %s", got.State, domain.CardStateReview)
	}
	if got.Stability != 5.5 {
		t.Errorf("Stability mismatch: got %f, want 5.5", got.Stability)
	}
	if got.Difficulty != 4.2 {
		t.Errorf("Difficulty mismatch: got %f, want 4.2", got.Difficulty)
	}
	if got.ScheduledDays != 3 {
		t.Errorf("ScheduledDays mismatch: got %d, want 3", got.ScheduledDays)
	}
	if got.Reps != 1 {
		t.Errorf("Reps mismatch: got %d, want 1", got.Reps)
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
	cards, err := repo.GetByEntryIDs(ctx, user.ID, []uuid.UUID{entry1.ID, entry2.ID, uuid.New()})
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
// Create — different users can have cards for entries sharing the same ref
// ---------------------------------------------------------------------------

func TestRepo_Create_DifferentUsersCanShareEntry(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user1 := testhelper.SeedUser(t, pool)
	user2 := testhelper.SeedUser(t, pool)
	ref := testhelper.SeedRefEntry(t, pool, "shared-"+uuid.New().String()[:8])
	entry1 := testhelper.SeedEntry(t, pool, user1.ID, ref.ID)
	entry2 := testhelper.SeedEntry(t, pool, user2.ID, ref.ID)

	_, err := repo.Create(ctx, user1.ID, entry1.ID)
	if err != nil {
		t.Fatalf("Create user1: unexpected error: %v", err)
	}

	// This should succeed — different users, different entries.
	_, err = repo.Create(ctx, user2.ID, entry2.ID)
	if err != nil {
		t.Fatalf("Create user2: unexpected error (ux_cards_entry may be global): %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetNewCards and ExistsByEntryIDs tests (Task 10b)
// ---------------------------------------------------------------------------

func TestRepo_GetNewCards_OrderedByCreatedAt(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)

	// Create 3 entries with cards in NEW state, with known creation order
	var entryIDs []uuid.UUID
	for i := 0; i < 3; i++ {
		ref := testhelper.SeedRefEntry(t, pool, fmt.Sprintf("new-order-%d-%s", i, uuid.New().String()[:8]))
		entry := testhelper.SeedEntryWithCard(t, pool, user.ID, ref.ID)
		entryIDs = append(entryIDs, entry.ID)
		time.Sleep(2 * time.Millisecond) // ensure different created_at
	}

	cards, err := repo.GetNewCards(ctx, user.ID, 10)
	if err != nil {
		t.Fatalf("GetNewCards: %v", err)
	}

	if len(cards) < 3 {
		t.Fatalf("GetNewCards: got %d cards, want >= 3", len(cards))
	}

	// Verify FIFO ordering: first created should appear first
	for i := 1; i < len(cards); i++ {
		if cards[i].CreatedAt.Before(cards[i-1].CreatedAt) {
			t.Errorf("card[%d].CreatedAt (%v) is before card[%d].CreatedAt (%v) — not FIFO",
				i, cards[i].CreatedAt, i-1, cards[i-1].CreatedAt)
		}
	}
}

func TestRepo_ExistsByEntryIDs_ReturnsCorrectMap(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	ref1 := testhelper.SeedRefEntry(t, pool, "exists1-"+uuid.New().String()[:8])
	ref2 := testhelper.SeedRefEntry(t, pool, "exists2-"+uuid.New().String()[:8])
	ref3 := testhelper.SeedRefEntry(t, pool, "exists3-"+uuid.New().String()[:8])

	// Entry 1 and 2 get cards, entry 3 does not
	entry1 := testhelper.SeedEntryWithCard(t, pool, user.ID, ref1.ID)
	entry2 := testhelper.SeedEntryWithCard(t, pool, user.ID, ref2.ID)
	entry3 := testhelper.SeedEntry(t, pool, user.ID, ref3.ID)

	result, err := repo.ExistsByEntryIDs(ctx, user.ID, []uuid.UUID{entry1.ID, entry2.ID, entry3.ID})
	if err != nil {
		t.Fatalf("ExistsByEntryIDs: %v", err)
	}

	if !result[entry1.ID] {
		t.Errorf("entry1 should have card")
	}
	if !result[entry2.ID] {
		t.Errorf("entry2 should have card")
	}
	if result[entry3.ID] {
		t.Errorf("entry3 should NOT have card")
	}
}

func TestRepo_GetDueCards_UserIsolation(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	userA := testhelper.SeedUser(t, pool)
	userB := testhelper.SeedUser(t, pool)

	// Each user has a card set to due in the past (REVIEW state)
	refA := testhelper.SeedRefEntry(t, pool, "isoA-"+uuid.New().String()[:8])
	refB := testhelper.SeedRefEntry(t, pool, "isoB-"+uuid.New().String()[:8])
	entryA := testhelper.SeedEntryWithCard(t, pool, userA.ID, refA.ID)
	entryB := testhelper.SeedEntryWithCard(t, pool, userB.ID, refB.ID)

	now := time.Now()
	past := now.Add(-1 * time.Hour)

	// Update both cards to REVIEW state with due in the past
	for _, c := range []struct {
		uid uuid.UUID
		cid uuid.UUID
	}{{userA.ID, entryA.Card.ID}, {userB.ID, entryB.Card.ID}} {
		_, err := repo.UpdateSRS(ctx, c.uid, c.cid, domain.SRSUpdateParams{
			State:      domain.CardStateReview,
			Stability:  5.0,
			Difficulty: 5.0,
			Due:        past,
			LastReview: &past,
			Reps:       1,
		})
		if err != nil {
			t.Fatalf("UpdateSRS: %v", err)
		}
	}

	// User A should only see their card
	cardsA, err := repo.GetDueCards(ctx, userA.ID, now, 10)
	if err != nil {
		t.Fatalf("GetDueCards userA: %v", err)
	}
	for _, c := range cardsA {
		if c.EntryID != entryA.ID {
			t.Errorf("userA got card with entryID %v, expected only %v", c.EntryID, entryA.ID)
		}
	}

	// User B should only see their card
	cardsB, err := repo.GetDueCards(ctx, userB.ID, now, 10)
	if err != nil {
		t.Fatalf("GetDueCards userB: %v", err)
	}
	for _, c := range cardsB {
		if c.EntryID != entryB.ID {
			t.Errorf("userB got card with entryID %v, expected only %v", c.EntryID, entryB.ID)
		}
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
