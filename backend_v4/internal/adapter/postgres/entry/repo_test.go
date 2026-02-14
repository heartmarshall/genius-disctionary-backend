package entry_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/entry"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/testhelper"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// newRepo sets up a test DB and returns a ready Repo + pool.
func newRepo(t *testing.T) (*entry.Repo, *pgxpool.Pool) {
	t.Helper()
	pool := testhelper.SetupTestDB(t)
	return entry.New(pool), pool
}

// buildEntry creates a minimal domain.Entry suitable for Create.
func buildEntry(userID uuid.UUID, text string, refEntryID *uuid.UUID) domain.Entry {
	now := time.Now().UTC().Truncate(time.Microsecond)
	return domain.Entry{
		ID:             uuid.New(),
		UserID:         userID,
		RefEntryID:     refEntryID,
		Text:           text,
		TextNormalized: domain.NormalizeText(text),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// seedTopic creates a topic for the given user and returns its ID.
func seedTopic(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	id := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	_, err := pool.Exec(ctx,
		`INSERT INTO topics (id, user_id, name, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		id, userID, name, now, now,
	)
	if err != nil {
		t.Fatalf("seedTopic: %v", err)
	}
	return id
}

// linkEntryTopic links an entry to a topic.
func linkEntryTopic(t *testing.T, pool *pgxpool.Pool, entryID, topicID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx,
		`INSERT INTO entry_topics (entry_id, topic_id) VALUES ($1, $2)`,
		entryID, topicID,
	)
	if err != nil {
		t.Fatalf("linkEntryTopic: %v", err)
	}
}

// seedCard creates a card for the given entry.
func seedCard(t *testing.T, pool *pgxpool.Pool, userID, entryID uuid.UUID, status domain.LearningStatus) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	id := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	_, err := pool.Exec(ctx,
		`INSERT INTO cards (id, user_id, entry_id, status, learning_step, interval_days, ease_factor, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, 0, 0, 2.5, $5, $6)`,
		id, userID, entryID, string(status), now, now,
	)
	if err != nil {
		t.Fatalf("seedCard: %v", err)
	}
	return id
}

// seedSense creates a sense for the given entry.
func seedSense(t *testing.T, pool *pgxpool.Pool, entryID uuid.UUID, pos *domain.PartOfSpeech) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	id := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	var posStr *string
	if pos != nil {
		s := string(*pos)
		posStr = &s
	}
	_, err := pool.Exec(ctx,
		`INSERT INTO senses (id, entry_id, part_of_speech, source_slug, position, created_at)
		 VALUES ($1, $2, $3, 'test', 0, $4)`,
		id, entryID, posStr, now,
	)
	if err != nil {
		t.Fatalf("seedSense: %v", err)
	}
	return id
}

// ---------------------------------------------------------------------------
// Create tests
// ---------------------------------------------------------------------------

func TestRepo_Create_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "create-happy-"+uuid.New().String()[:8])

	e := buildEntry(user.ID, refEntry.Text, &refEntry.ID)
	notes := "some notes"
	e.Notes = &notes

	got, err := repo.Create(ctx, user.ID, e)
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	if got.ID != e.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, e.ID)
	}
	if got.UserID != user.ID {
		t.Errorf("UserID mismatch: got %s, want %s", got.UserID, user.ID)
	}
	if got.Text != e.Text {
		t.Errorf("Text mismatch: got %q, want %q", got.Text, e.Text)
	}
	if got.TextNormalized != e.TextNormalized {
		t.Errorf("TextNormalized mismatch: got %q, want %q", got.TextNormalized, e.TextNormalized)
	}
	if got.Notes == nil || *got.Notes != notes {
		t.Errorf("Notes mismatch: got %v, want %q", got.Notes, notes)
	}
	if got.RefEntryID == nil || *got.RefEntryID != refEntry.ID {
		t.Errorf("RefEntryID mismatch: got %v, want %s", got.RefEntryID, refEntry.ID)
	}
	if got.DeletedAt != nil {
		t.Errorf("expected DeletedAt to be nil, got %v", got.DeletedAt)
	}
}

func TestRepo_Create_WithoutRefEntry(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	text := "custom-word-" + uuid.New().String()[:8]
	e := buildEntry(user.ID, text, nil)

	got, err := repo.Create(ctx, user.ID, e)
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	if got.RefEntryID != nil {
		t.Errorf("expected RefEntryID to be nil, got %v", got.RefEntryID)
	}
}

func TestRepo_Create_DuplicateText(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	text := "duplicate-" + uuid.New().String()[:8]

	e1 := buildEntry(user.ID, text, nil)
	if _, err := repo.Create(ctx, user.ID, e1); err != nil {
		t.Fatalf("Create first: %v", err)
	}

	e2 := buildEntry(user.ID, text, nil)
	_, err := repo.Create(ctx, user.ID, e2)
	assertIsDomainError(t, err, domain.ErrAlreadyExists)
}

func TestRepo_Create_ConcurrentSameText(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	text := "concurrent-" + uuid.New().String()[:8]

	const goroutines = 5
	var wg sync.WaitGroup
	wg.Add(goroutines)

	errs := make([]error, goroutines)
	for i := range goroutines {
		go func() {
			defer wg.Done()
			e := buildEntry(user.ID, text, nil)
			_, errs[i] = repo.Create(ctx, user.ID, e)
		}()
	}
	wg.Wait()

	// Exactly 1 should succeed; the rest should get ErrAlreadyExists.
	successes := 0
	for _, err := range errs {
		if err == nil {
			successes++
		} else if !errors.Is(err, domain.ErrAlreadyExists) {
			t.Errorf("unexpected error: %v", err)
		}
	}
	if successes != 1 {
		t.Errorf("expected exactly 1 success, got %d", successes)
	}
}

// ---------------------------------------------------------------------------
// GetByID tests
// ---------------------------------------------------------------------------

func TestRepo_GetByID_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	e := buildEntry(user.ID, "getbyid-"+uuid.New().String()[:8], nil)
	created, err := repo.Create(ctx, user.ID, e)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, user.ID, created.ID)
	if err != nil {
		t.Fatalf("GetByID: unexpected error: %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, created.ID)
	}
	if got.TextNormalized != created.TextNormalized {
		t.Errorf("TextNormalized mismatch: got %q, want %q", got.TextNormalized, created.TextNormalized)
	}
}

func TestRepo_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	_, err := repo.GetByID(ctx, user.ID, uuid.New())
	assertIsDomainError(t, err, domain.ErrNotFound)
}

func TestRepo_GetByID_SoftDeletedNotVisible(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	e := buildEntry(user.ID, "softdel-hidden-"+uuid.New().String()[:8], nil)
	created, err := repo.Create(ctx, user.ID, e)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.SoftDelete(ctx, user.ID, created.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	_, err = repo.GetByID(ctx, user.ID, created.ID)
	assertIsDomainError(t, err, domain.ErrNotFound)
}

func TestRepo_GetByID_WrongUser(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user1 := testhelper.SeedUser(t, pool)
	user2 := testhelper.SeedUser(t, pool)

	e := buildEntry(user1.ID, "wronguser-"+uuid.New().String()[:8], nil)
	created, err := repo.Create(ctx, user1.ID, e)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err = repo.GetByID(ctx, user2.ID, created.ID)
	assertIsDomainError(t, err, domain.ErrNotFound)
}

// ---------------------------------------------------------------------------
// GetByText tests
// ---------------------------------------------------------------------------

func TestRepo_GetByText_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	text := "ByText-" + uuid.New().String()[:8]
	e := buildEntry(user.ID, text, nil)
	created, err := repo.Create(ctx, user.ID, e)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByText(ctx, user.ID, created.TextNormalized)
	if err != nil {
		t.Fatalf("GetByText: unexpected error: %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, created.ID)
	}
}

func TestRepo_GetByText_NotFound(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	_, err := repo.GetByText(ctx, user.ID, "nonexistent-"+uuid.New().String()[:8])
	assertIsDomainError(t, err, domain.ErrNotFound)
}

// ---------------------------------------------------------------------------
// GetByIDs tests
// ---------------------------------------------------------------------------

func TestRepo_GetByIDs_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	suffix := uuid.New().String()[:8]

	var ids []uuid.UUID
	for i := 0; i < 3; i++ {
		e := buildEntry(user.ID, "batch-"+suffix+"-"+uuid.New().String()[:4], nil)
		created, err := repo.Create(ctx, user.ID, e)
		if err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
		ids = append(ids, created.ID)
	}

	got, err := repo.GetByIDs(ctx, user.ID, ids)
	if err != nil {
		t.Fatalf("GetByIDs: unexpected error: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}
}

func TestRepo_GetByIDs_Empty(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	got, err := repo.GetByIDs(ctx, user.ID, []uuid.UUID{})
	if err != nil {
		t.Fatalf("GetByIDs empty: unexpected error: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("expected 0 entries, got %d", len(got))
	}
}

func TestRepo_GetByIDs_IgnoresSoftDeleted(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)

	e1 := buildEntry(user.ID, "batchdel-a-"+uuid.New().String()[:8], nil)
	c1, _ := repo.Create(ctx, user.ID, e1)
	e2 := buildEntry(user.ID, "batchdel-b-"+uuid.New().String()[:8], nil)
	c2, _ := repo.Create(ctx, user.ID, e2)

	_ = repo.SoftDelete(ctx, user.ID, c1.ID)

	got, err := repo.GetByIDs(ctx, user.ID, []uuid.UUID{c1.ID, c2.ID})
	if err != nil {
		t.Fatalf("GetByIDs: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 entry (non-deleted), got %d", len(got))
	}
	if got[0].ID != c2.ID {
		t.Errorf("expected entry %s, got %s", c2.ID, got[0].ID)
	}
}

// ---------------------------------------------------------------------------
// CountByUser tests
// ---------------------------------------------------------------------------

func TestRepo_CountByUser(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)

	count0, err := repo.CountByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("CountByUser: %v", err)
	}
	if count0 != 0 {
		t.Errorf("expected 0, got %d", count0)
	}

	for i := 0; i < 3; i++ {
		e := buildEntry(user.ID, "count-"+uuid.New().String()[:8], nil)
		if _, err := repo.Create(ctx, user.ID, e); err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
	}

	count3, err := repo.CountByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("CountByUser: %v", err)
	}
	if count3 != 3 {
		t.Errorf("expected 3, got %d", count3)
	}

	// Soft delete one: should not be counted.
	e := buildEntry(user.ID, "count-del-"+uuid.New().String()[:8], nil)
	created, _ := repo.Create(ctx, user.ID, e)
	_ = repo.SoftDelete(ctx, user.ID, created.ID)

	count3again, err := repo.CountByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("CountByUser: %v", err)
	}
	if count3again != 3 {
		t.Errorf("expected 3 after soft delete, got %d", count3again)
	}
}

// ---------------------------------------------------------------------------
// UpdateNotes tests
// ---------------------------------------------------------------------------

func TestRepo_UpdateNotes_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	e := buildEntry(user.ID, "notes-"+uuid.New().String()[:8], nil)
	created, _ := repo.Create(ctx, user.ID, e)

	newNotes := "updated notes"
	got, err := repo.UpdateNotes(ctx, user.ID, created.ID, &newNotes)
	if err != nil {
		t.Fatalf("UpdateNotes: unexpected error: %v", err)
	}

	if got.Notes == nil || *got.Notes != newNotes {
		t.Errorf("Notes mismatch: got %v, want %q", got.Notes, newNotes)
	}
	if !got.UpdatedAt.After(created.UpdatedAt) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestRepo_UpdateNotes_SetToNil(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	notes := "initial notes"
	e := buildEntry(user.ID, "notes-nil-"+uuid.New().String()[:8], nil)
	e.Notes = &notes
	created, _ := repo.Create(ctx, user.ID, e)

	got, err := repo.UpdateNotes(ctx, user.ID, created.ID, nil)
	if err != nil {
		t.Fatalf("UpdateNotes: unexpected error: %v", err)
	}

	if got.Notes != nil {
		t.Errorf("expected Notes to be nil, got %v", got.Notes)
	}
}

func TestRepo_UpdateNotes_NotFound(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	notes := "test"
	_, err := repo.UpdateNotes(ctx, user.ID, uuid.New(), &notes)
	assertIsDomainError(t, err, domain.ErrNotFound)
}

// ---------------------------------------------------------------------------
// SoftDelete tests
// ---------------------------------------------------------------------------

func TestRepo_SoftDelete_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	e := buildEntry(user.ID, "softdel-"+uuid.New().String()[:8], nil)
	created, _ := repo.Create(ctx, user.ID, e)

	if err := repo.SoftDelete(ctx, user.ID, created.ID); err != nil {
		t.Fatalf("SoftDelete: unexpected error: %v", err)
	}

	// Should not be visible via GetByID.
	_, err := repo.GetByID(ctx, user.ID, created.ID)
	assertIsDomainError(t, err, domain.ErrNotFound)
}

func TestRepo_SoftDelete_Idempotent(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	e := buildEntry(user.ID, "softdel-idem-"+uuid.New().String()[:8], nil)
	created, _ := repo.Create(ctx, user.ID, e)

	// First soft delete.
	if err := repo.SoftDelete(ctx, user.ID, created.ID); err != nil {
		t.Fatalf("SoftDelete first: %v", err)
	}

	// Second soft delete should not error (idempotent).
	if err := repo.SoftDelete(ctx, user.ID, created.ID); err != nil {
		t.Fatalf("SoftDelete second (idempotent): unexpected error: %v", err)
	}
}

func TestRepo_SoftDelete_NonexistentEntry(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)

	// Soft-deleting a nonexistent entry should not error (idempotent).
	if err := repo.SoftDelete(ctx, user.ID, uuid.New()); err != nil {
		t.Fatalf("SoftDelete nonexistent: unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Restore tests
// ---------------------------------------------------------------------------

func TestRepo_Restore_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	e := buildEntry(user.ID, "restore-"+uuid.New().String()[:8], nil)
	created, _ := repo.Create(ctx, user.ID, e)

	_ = repo.SoftDelete(ctx, user.ID, created.ID)

	got, err := repo.Restore(ctx, user.ID, created.ID)
	if err != nil {
		t.Fatalf("Restore: unexpected error: %v", err)
	}

	if got.DeletedAt != nil {
		t.Errorf("expected DeletedAt to be nil after restore, got %v", got.DeletedAt)
	}

	// Should be visible again via GetByID.
	fetched, err := repo.GetByID(ctx, user.ID, created.ID)
	if err != nil {
		t.Fatalf("GetByID after restore: %v", err)
	}
	if fetched.ID != created.ID {
		t.Errorf("ID mismatch after restore: got %s, want %s", fetched.ID, created.ID)
	}
}

func TestRepo_Restore_NotDeleted(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	e := buildEntry(user.ID, "restore-notdel-"+uuid.New().String()[:8], nil)
	created, _ := repo.Create(ctx, user.ID, e)

	// Restoring a non-deleted entry should return ErrNotFound.
	_, err := repo.Restore(ctx, user.ID, created.ID)
	assertIsDomainError(t, err, domain.ErrNotFound)
}

// ---------------------------------------------------------------------------
// Re-create after soft delete
// ---------------------------------------------------------------------------

func TestRepo_ReCreateAfterSoftDelete(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	text := "recreate-" + uuid.New().String()[:8]

	// Create and soft-delete.
	e1 := buildEntry(user.ID, text, nil)
	created1, err := repo.Create(ctx, user.ID, e1)
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}
	_ = repo.SoftDelete(ctx, user.ID, created1.ID)

	// Re-create with same text should succeed (partial unique index).
	e2 := buildEntry(user.ID, text, nil)
	created2, err := repo.Create(ctx, user.ID, e2)
	if err != nil {
		t.Fatalf("Create after soft delete: unexpected error: %v", err)
	}

	if created2.ID == created1.ID {
		t.Error("expected different IDs for re-created entry")
	}
	if created2.TextNormalized != created1.TextNormalized {
		t.Error("expected same normalized text")
	}
}

// ---------------------------------------------------------------------------
// HardDeleteOld tests
// ---------------------------------------------------------------------------

func TestRepo_HardDeleteOld_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)

	// Create entries: 2 soft-deleted old, 1 soft-deleted recent, 1 alive.
	threshold := time.Now().UTC().Add(-24 * time.Hour)

	// Old soft-deleted entries.
	for i := 0; i < 2; i++ {
		e := buildEntry(user.ID, "harddelold-"+uuid.New().String()[:8], nil)
		created, err := repo.Create(ctx, user.ID, e)
		if err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
		// Manually set deleted_at in the past.
		_, err = pool.Exec(ctx,
			`UPDATE entries SET deleted_at = $1 WHERE id = $2`,
			threshold.Add(-time.Hour), created.ID,
		)
		if err != nil {
			t.Fatalf("set old deleted_at: %v", err)
		}
	}

	// Recently soft-deleted entry.
	eRecent := buildEntry(user.ID, "harddelrecent-"+uuid.New().String()[:8], nil)
	createdRecent, _ := repo.Create(ctx, user.ID, eRecent)
	_ = repo.SoftDelete(ctx, user.ID, createdRecent.ID)

	// Alive entry.
	eAlive := buildEntry(user.ID, "harddelalive-"+uuid.New().String()[:8], nil)
	if _, err := repo.Create(ctx, user.ID, eAlive); err != nil {
		t.Fatalf("Create alive: %v", err)
	}

	// Hard delete old.
	deleted, err := repo.HardDeleteOld(ctx, threshold)
	if err != nil {
		t.Fatalf("HardDeleteOld: unexpected error: %v", err)
	}

	if deleted != 2 {
		t.Errorf("expected 2 hard-deleted, got %d", deleted)
	}

	// The recent soft-deleted entry should still exist.
	var recentExists bool
	_ = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM entries WHERE id = $1)`, createdRecent.ID).Scan(&recentExists)
	if !recentExists {
		t.Error("expected recent soft-deleted entry to still exist")
	}
}

func TestRepo_HardDeleteOld_NothingToDelete(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	// Use a very old threshold so nothing matches.
	veryOld := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	deleted, err := repo.HardDeleteOld(ctx, veryOld)
	if err != nil {
		t.Fatalf("HardDeleteOld: unexpected error: %v", err)
	}
	if deleted != 0 {
		t.Errorf("expected 0 deleted, got %d", deleted)
	}
}

// ---------------------------------------------------------------------------
// Find tests: offset-based pagination
// ---------------------------------------------------------------------------

func TestRepo_Find_AllEntries(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	suffix := uuid.New().String()[:8]

	for i := 0; i < 3; i++ {
		e := buildEntry(user.ID, "findall-"+suffix+"-"+uuid.New().String()[:4], nil)
		if _, err := repo.Create(ctx, user.ID, e); err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
	}

	// Also soft-delete one to verify it's excluded.
	eDel := buildEntry(user.ID, "findall-del-"+suffix, nil)
	cDel, _ := repo.Create(ctx, user.ID, eDel)
	_ = repo.SoftDelete(ctx, user.ID, cDel.ID)

	result, err := repo.Find(ctx, user.ID, entry.Filter{})
	if err != nil {
		t.Fatalf("Find: unexpected error: %v", err)
	}

	if result.TotalCount < 3 {
		t.Errorf("expected TotalCount >= 3, got %d", result.TotalCount)
	}
	if len(result.Entries) < 3 {
		t.Errorf("expected at least 3 entries, got %d", len(result.Entries))
	}

	// Verify soft-deleted entry is not in results.
	for _, e := range result.Entries {
		if e.ID == cDel.ID {
			t.Error("soft-deleted entry should not appear in Find results")
		}
	}
}

func TestRepo_Find_EmptyResult(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)

	result, err := repo.Find(ctx, user.ID, entry.Filter{})
	if err != nil {
		t.Fatalf("Find: unexpected error: %v", err)
	}

	if result.TotalCount != 0 {
		t.Errorf("expected TotalCount 0, got %d", result.TotalCount)
	}
	if len(result.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(result.Entries))
	}
}

func TestRepo_Find_OffsetPagination(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	suffix := uuid.New().String()[:8]

	for i := 0; i < 5; i++ {
		e := buildEntry(user.ID, "page-"+suffix+"-"+uuid.New().String()[:4], nil)
		if _, err := repo.Create(ctx, user.ID, e); err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
		// Small sleep to ensure distinct created_at.
		time.Sleep(time.Millisecond)
	}

	// Page 1: limit 2, offset 0.
	result1, err := repo.Find(ctx, user.ID, entry.Filter{
		Limit:  2,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("Find page 1: %v", err)
	}

	if len(result1.Entries) != 2 {
		t.Fatalf("page 1: expected 2 entries, got %d", len(result1.Entries))
	}

	// TotalCount should reflect all matching entries (not just page).
	if result1.TotalCount < 5 {
		t.Errorf("totalCount should be >= 5, got %d", result1.TotalCount)
	}

	// Page 2: limit 2, offset 2.
	result2, err := repo.Find(ctx, user.ID, entry.Filter{
		Limit:  2,
		Offset: 2,
	})
	if err != nil {
		t.Fatalf("Find page 2: %v", err)
	}

	if len(result2.Entries) != 2 {
		t.Fatalf("page 2: expected 2 entries, got %d", len(result2.Entries))
	}

	// Pages should not overlap.
	for _, e1 := range result1.Entries {
		for _, e2 := range result2.Entries {
			if e1.ID == e2.ID {
				t.Error("pages should not contain duplicate entries")
			}
		}
	}
}

func TestRepo_Find_TotalCountIndependentOfLimit(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	suffix := uuid.New().String()[:8]

	for i := 0; i < 5; i++ {
		e := buildEntry(user.ID, "totalcount-"+suffix+"-"+uuid.New().String()[:4], nil)
		if _, err := repo.Create(ctx, user.ID, e); err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
	}

	// Fetch with limit 2.
	result, err := repo.Find(ctx, user.ID, entry.Filter{Limit: 2})
	if err != nil {
		t.Fatalf("Find: %v", err)
	}

	if len(result.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result.Entries))
	}
	if result.TotalCount < 5 {
		t.Errorf("totalCount should be >= 5, got %d", result.TotalCount)
	}
}

// ---------------------------------------------------------------------------
// Find tests: search filter
// ---------------------------------------------------------------------------

func TestRepo_Find_SearchFilter(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	suffix := uuid.New().String()[:8]

	// Create entries with known text patterns.
	e1 := buildEntry(user.ID, "elephant-"+suffix, nil)
	if _, err := repo.Create(ctx, user.ID, e1); err != nil {
		t.Fatalf("Create elephant: %v", err)
	}
	e2 := buildEntry(user.ID, "elephantine-"+suffix, nil)
	if _, err := repo.Create(ctx, user.ID, e2); err != nil {
		t.Fatalf("Create elephantine: %v", err)
	}
	e3 := buildEntry(user.ID, "zebra-"+suffix, nil)
	if _, err := repo.Create(ctx, user.ID, e3); err != nil {
		t.Fatalf("Create zebra: %v", err)
	}

	search := "elephant-" + suffix
	result, err := repo.Find(ctx, user.ID, entry.Filter{Search: &search})
	if err != nil {
		t.Fatalf("Find with search: %v", err)
	}

	// Should find exactly 1 entry matching the exact ILIKE pattern.
	if result.TotalCount != 1 {
		t.Errorf("expected totalCount 1, got %d", result.TotalCount)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.Entries))
	}
	if result.Entries[0].TextNormalized != domain.NormalizeText("elephant-"+suffix) {
		t.Errorf("expected elephant entry, got %q", result.Entries[0].TextNormalized)
	}
}

func TestRepo_Find_EmptySearchIgnored(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	suffix := uuid.New().String()[:8]

	e := buildEntry(user.ID, "emptysearch-"+suffix, nil)
	if _, err := repo.Create(ctx, user.ID, e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	empty := ""
	result, err := repo.Find(ctx, user.ID, entry.Filter{Search: &empty})
	if err != nil {
		t.Fatalf("Find with empty search: %v", err)
	}

	if result.TotalCount < 1 {
		t.Errorf("expected at least 1 entry, got %d", result.TotalCount)
	}
}

// ---------------------------------------------------------------------------
// Find tests: HasCard filter
// ---------------------------------------------------------------------------

func TestRepo_Find_HasCardFilter(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	suffix := uuid.New().String()[:8]

	// Entry with card.
	eWithCard := buildEntry(user.ID, "hascard-yes-"+suffix, nil)
	cWithCard, _ := repo.Create(ctx, user.ID, eWithCard)
	seedCard(t, pool, user.ID, cWithCard.ID, domain.LearningStatusNew)

	// Entry without card.
	eNoCard := buildEntry(user.ID, "hascard-no-"+suffix, nil)
	repo.Create(ctx, user.ID, eNoCard)

	// Filter: HasCard = true.
	hasCard := true
	search := suffix
	result, err := repo.Find(ctx, user.ID, entry.Filter{HasCard: &hasCard, Search: &search})
	if err != nil {
		t.Fatalf("Find HasCard=true: %v", err)
	}
	if result.TotalCount != 1 {
		t.Errorf("HasCard=true: expected 1, got %d", result.TotalCount)
	}

	// Filter: HasCard = false.
	noCard := false
	result2, err := repo.Find(ctx, user.ID, entry.Filter{HasCard: &noCard, Search: &search})
	if err != nil {
		t.Fatalf("Find HasCard=false: %v", err)
	}
	if result2.TotalCount != 1 {
		t.Errorf("HasCard=false: expected 1, got %d", result2.TotalCount)
	}
}

// ---------------------------------------------------------------------------
// Find tests: PartOfSpeech filter
// ---------------------------------------------------------------------------

func TestRepo_Find_PartOfSpeechFilter(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	suffix := uuid.New().String()[:8]

	// Entry with NOUN sense.
	eNoun := buildEntry(user.ID, "pos-noun-"+suffix, nil)
	cNoun, _ := repo.Create(ctx, user.ID, eNoun)
	posNoun := domain.PartOfSpeechNoun
	seedSense(t, pool, cNoun.ID, &posNoun)

	// Entry with VERB sense.
	eVerb := buildEntry(user.ID, "pos-verb-"+suffix, nil)
	cVerb, _ := repo.Create(ctx, user.ID, eVerb)
	posVerb := domain.PartOfSpeechVerb
	seedSense(t, pool, cVerb.ID, &posVerb)

	// Filter: NOUN.
	posFilter := domain.PartOfSpeechNoun
	search := suffix
	result, err := repo.Find(ctx, user.ID, entry.Filter{PartOfSpeech: &posFilter, Search: &search})
	if err != nil {
		t.Fatalf("Find POS=NOUN: %v", err)
	}
	if result.TotalCount != 1 {
		t.Errorf("POS=NOUN: expected 1, got %d", result.TotalCount)
	}
	if len(result.Entries) == 1 && result.Entries[0].ID != cNoun.ID {
		t.Errorf("expected noun entry, got %s", result.Entries[0].ID)
	}
}

// ---------------------------------------------------------------------------
// Find tests: TopicID filter
// ---------------------------------------------------------------------------

func TestRepo_Find_TopicFilter(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	suffix := uuid.New().String()[:8]

	topicID := seedTopic(t, pool, user.ID, "topic-"+suffix)

	// Entry in topic.
	eInTopic := buildEntry(user.ID, "topic-in-"+suffix, nil)
	cInTopic, _ := repo.Create(ctx, user.ID, eInTopic)
	linkEntryTopic(t, pool, cInTopic.ID, topicID)

	// Entry not in topic.
	eNotInTopic := buildEntry(user.ID, "topic-out-"+suffix, nil)
	repo.Create(ctx, user.ID, eNotInTopic)

	search := suffix
	result, err := repo.Find(ctx, user.ID, entry.Filter{TopicID: &topicID, Search: &search})
	if err != nil {
		t.Fatalf("Find TopicID: %v", err)
	}
	if result.TotalCount != 1 {
		t.Errorf("TopicID filter: expected 1, got %d", result.TotalCount)
	}
}

// ---------------------------------------------------------------------------
// Find tests: Status filter
// ---------------------------------------------------------------------------

func TestRepo_Find_StatusFilter(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	suffix := uuid.New().String()[:8]

	// Entry with NEW card.
	eNew := buildEntry(user.ID, "status-new-"+suffix, nil)
	cNew, _ := repo.Create(ctx, user.ID, eNew)
	seedCard(t, pool, user.ID, cNew.ID, domain.LearningStatusNew)

	// Entry with LEARNING card.
	eLearning := buildEntry(user.ID, "status-learning-"+suffix, nil)
	cLearning, _ := repo.Create(ctx, user.ID, eLearning)
	seedCard(t, pool, user.ID, cLearning.ID, domain.LearningStatusLearning)

	// Filter: NEW.
	statusNew := domain.LearningStatusNew
	search := suffix
	result, err := repo.Find(ctx, user.ID, entry.Filter{Status: &statusNew, Search: &search})
	if err != nil {
		t.Fatalf("Find Status=NEW: %v", err)
	}
	if result.TotalCount != 1 {
		t.Errorf("Status=NEW: expected 1, got %d", result.TotalCount)
	}
}

// ---------------------------------------------------------------------------
// Find tests: combined filters
// ---------------------------------------------------------------------------

func TestRepo_Find_CombinedFilters(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	suffix := uuid.New().String()[:8]
	topicID := seedTopic(t, pool, user.ID, "combined-topic-"+suffix)

	// Entry that matches ALL filters: has card (NEW), is NOUN, in topic.
	eMatch := buildEntry(user.ID, "combined-match-"+suffix, nil)
	cMatch, _ := repo.Create(ctx, user.ID, eMatch)
	seedCard(t, pool, user.ID, cMatch.ID, domain.LearningStatusNew)
	posNoun := domain.PartOfSpeechNoun
	seedSense(t, pool, cMatch.ID, &posNoun)
	linkEntryTopic(t, pool, cMatch.ID, topicID)

	// Entry that matches only some filters.
	ePartial := buildEntry(user.ID, "combined-partial-"+suffix, nil)
	cPartial, _ := repo.Create(ctx, user.ID, ePartial)
	posVerb := domain.PartOfSpeechVerb
	seedSense(t, pool, cPartial.ID, &posVerb) // wrong POS

	hasCard := true
	pos := domain.PartOfSpeechNoun
	status := domain.LearningStatusNew
	search := suffix

	result, err := repo.Find(ctx, user.ID, entry.Filter{
		Search:       &search,
		HasCard:      &hasCard,
		PartOfSpeech: &pos,
		TopicID:      &topicID,
		Status:       &status,
	})
	if err != nil {
		t.Fatalf("Find combined: %v", err)
	}
	if result.TotalCount != 1 {
		t.Errorf("combined filters: expected 1, got %d", result.TotalCount)
	}
	if len(result.Entries) == 1 && result.Entries[0].ID != cMatch.ID {
		t.Errorf("expected matching entry %s, got %s", cMatch.ID, result.Entries[0].ID)
	}
}

// ---------------------------------------------------------------------------
// Find tests: sorting
// ---------------------------------------------------------------------------

func TestRepo_Find_SortByText(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	suffix := uuid.New().String()[:8]

	texts := []string{"apple-" + suffix, "banana-" + suffix, "cherry-" + suffix}
	for _, text := range texts {
		e := buildEntry(user.ID, text, nil)
		if _, err := repo.Create(ctx, user.ID, e); err != nil {
			t.Fatalf("Create %s: %v", text, err)
		}
	}

	search := suffix
	result, err := repo.Find(ctx, user.ID, entry.Filter{
		Search:    &search,
		SortBy:    "text",
		SortOrder: "ASC",
	})
	if err != nil {
		t.Fatalf("Find sort by text: %v", err)
	}

	if len(result.Entries) < 3 {
		t.Fatalf("expected at least 3 entries, got %d", len(result.Entries))
	}

	// Verify ascending text order.
	for i := 1; i < len(result.Entries); i++ {
		if result.Entries[i].TextNormalized < result.Entries[i-1].TextNormalized {
			t.Errorf("entries not sorted by text ASC: %q < %q at index %d",
				result.Entries[i].TextNormalized, result.Entries[i-1].TextNormalized, i)
		}
	}
}

func TestRepo_Find_SortByCreatedAtDESC(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	suffix := uuid.New().String()[:8]

	for i := 0; i < 3; i++ {
		e := buildEntry(user.ID, "sortcreated-"+suffix+"-"+uuid.New().String()[:4], nil)
		if _, err := repo.Create(ctx, user.ID, e); err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
		time.Sleep(time.Millisecond)
	}

	search := "sortcreated-" + suffix
	result, err := repo.Find(ctx, user.ID, entry.Filter{
		Search:    &search,
		SortBy:    "created_at",
		SortOrder: "DESC",
	})
	if err != nil {
		t.Fatalf("Find sort DESC: %v", err)
	}

	// Verify descending created_at order.
	for i := 1; i < len(result.Entries); i++ {
		if result.Entries[i].CreatedAt.After(result.Entries[i-1].CreatedAt) {
			t.Errorf("entries not sorted by created_at DESC at index %d", i)
		}
	}
}

// ---------------------------------------------------------------------------
// Find tests: cursor-based pagination
// ---------------------------------------------------------------------------

func TestRepo_Find_CursorPagination(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	suffix := uuid.New().String()[:8]

	// Create 5 entries with distinct timestamps.
	for i := 0; i < 5; i++ {
		e := buildEntry(user.ID, "cursor-"+suffix+"-"+uuid.New().String()[:4], nil)
		if _, err := repo.Create(ctx, user.ID, e); err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
		time.Sleep(2 * time.Millisecond)
	}

	search := "cursor-" + suffix

	// First page: limit 2, no cursor.
	result1, err := repo.Find(ctx, user.ID, entry.Filter{
		Search:    &search,
		SortBy:    "created_at",
		SortOrder: "DESC",
		Limit:     2,
	})
	if err != nil {
		t.Fatalf("Find page 1: %v", err)
	}
	if len(result1.Entries) != 2 {
		t.Fatalf("page 1: expected 2, got %d", len(result1.Entries))
	}

	// Create cursor from last entry of page 1.
	lastEntry := result1.Entries[len(result1.Entries)-1]
	cursor := entry.CursorFromEntry(lastEntry, "created_at")

	// Second page: limit 2, with cursor.
	result2, err := repo.Find(ctx, user.ID, entry.Filter{
		Search:    &search,
		SortBy:    "created_at",
		SortOrder: "DESC",
		Limit:     2,
		Cursor:    &cursor,
	})
	if err != nil {
		t.Fatalf("Find page 2: %v", err)
	}
	if len(result2.Entries) != 2 {
		t.Fatalf("page 2: expected 2, got %d", len(result2.Entries))
	}
	if result2.HasNextPage != true {
		t.Error("page 2: expected HasNextPage=true")
	}

	// Verify no overlap between pages.
	page1IDs := make(map[uuid.UUID]bool)
	for _, e := range result1.Entries {
		page1IDs[e.ID] = true
	}
	for _, e := range result2.Entries {
		if page1IDs[e.ID] {
			t.Error("cursor pagination: pages should not overlap")
		}
	}

	// Third page: should have 1 entry and hasNextPage=false.
	lastEntry2 := result2.Entries[len(result2.Entries)-1]
	cursor2 := entry.CursorFromEntry(lastEntry2, "created_at")

	result3, err := repo.Find(ctx, user.ID, entry.Filter{
		Search:    &search,
		SortBy:    "created_at",
		SortOrder: "DESC",
		Limit:     2,
		Cursor:    &cursor2,
	})
	if err != nil {
		t.Fatalf("Find page 3: %v", err)
	}
	if len(result3.Entries) != 1 {
		t.Errorf("page 3: expected 1, got %d", len(result3.Entries))
	}
	if result3.HasNextPage {
		t.Error("page 3: expected HasNextPage=false")
	}
}

func TestRepo_Find_CursorPaginationTextSort(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	suffix := uuid.New().String()[:8]

	texts := []string{
		"aaa-cursor-text-" + suffix,
		"bbb-cursor-text-" + suffix,
		"ccc-cursor-text-" + suffix,
	}
	for _, text := range texts {
		e := buildEntry(user.ID, text, nil)
		if _, err := repo.Create(ctx, user.ID, e); err != nil {
			t.Fatalf("Create %s: %v", text, err)
		}
	}

	search := "cursor-text-" + suffix

	// First page: limit 1, sort by text ASC.
	result1, err := repo.Find(ctx, user.ID, entry.Filter{
		Search:    &search,
		SortBy:    "text",
		SortOrder: "ASC",
		Limit:     1,
	})
	if err != nil {
		t.Fatalf("Find text cursor page 1: %v", err)
	}
	if len(result1.Entries) != 1 {
		t.Fatalf("expected 1, got %d", len(result1.Entries))
	}
	if result1.Entries[0].TextNormalized != domain.NormalizeText("aaa-cursor-text-"+suffix) {
		t.Errorf("expected aaa entry, got %q", result1.Entries[0].TextNormalized)
	}

	// Second page with cursor.
	cursor := entry.CursorFromEntry(result1.Entries[0], "text")
	result2, err := repo.Find(ctx, user.ID, entry.Filter{
		Search:    &search,
		SortBy:    "text",
		SortOrder: "ASC",
		Limit:     1,
		Cursor:    &cursor,
	})
	if err != nil {
		t.Fatalf("Find text cursor page 2: %v", err)
	}
	if len(result2.Entries) != 1 {
		t.Fatalf("expected 1, got %d", len(result2.Entries))
	}
	if result2.Entries[0].TextNormalized != domain.NormalizeText("bbb-cursor-text-"+suffix) {
		t.Errorf("expected bbb entry, got %q", result2.Entries[0].TextNormalized)
	}
}

func TestRepo_Find_InvalidCursor(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)

	// Test with garbage cursor.
	badCursor := "not-valid-base64!!!"
	_, err := repo.Find(ctx, user.ID, entry.Filter{Cursor: &badCursor})
	assertIsDomainError(t, err, domain.ErrValidation)

	// Test with valid base64 but bad format (no pipe).
	badCursor2 := "bm9waXBl" // base64("nopipe")
	_, err = repo.Find(ctx, user.ID, entry.Filter{Cursor: &badCursor2})
	assertIsDomainError(t, err, domain.ErrValidation)

	// Test with valid base64, pipe, but bad UUID.
	badCursor3 := entry.EncodeCursor("2024-01-01T00:00:00Z", "not-a-uuid")
	_, err = repo.Find(ctx, user.ID, entry.Filter{Cursor: &badCursor3})
	assertIsDomainError(t, err, domain.ErrValidation)
}

// ---------------------------------------------------------------------------
// Find tests: limit clamping
// ---------------------------------------------------------------------------

func TestRepo_Find_LimitDefaults(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)

	// With limit 0, should use default (50).
	result, err := repo.Find(ctx, user.ID, entry.Filter{Limit: 0})
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	// Just verify it doesn't fail; actual entries may be 0 or more.
	_ = result

	// With negative limit, should use default.
	result2, err := repo.Find(ctx, user.ID, entry.Filter{Limit: -5})
	if err != nil {
		t.Fatalf("Find negative limit: %v", err)
	}
	_ = result2
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
