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
func seedCard(t *testing.T, pool *pgxpool.Pool, userID, entryID uuid.UUID, state domain.CardState) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	id := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	_, err := pool.Exec(ctx,
		`INSERT INTO cards (id, user_id, entry_id, state, step, stability, difficulty, due, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, 0, 0, 0, $5, $5, $6)`,
		id, userID, entryID, string(state), now, now,
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

// intPtr returns a pointer to the given int.
func intPtr(v int) *int {
	return &v
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

	got, err := repo.Create(ctx, &e)
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	if got.ID != e.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, e.ID)
	}
	if got.UserID != e.UserID {
		t.Errorf("UserID mismatch: got %s, want %s", got.UserID, e.UserID)
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

	got, err := repo.Create(ctx, &e)
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
	if _, err := repo.Create(ctx, &e1); err != nil {
		t.Fatalf("Create first: %v", err)
	}

	e2 := buildEntry(user.ID, text, nil)
	_, err := repo.Create(ctx, &e2)
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
			_, errs[i] = repo.Create(ctx, &e)
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
	created, err := repo.Create(ctx, &e)
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
	created, err := repo.Create(ctx, &e)
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
	created, err := repo.Create(ctx, &e)
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
	created, err := repo.Create(ctx, &e)
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
		created, err := repo.Create(ctx, &e)
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
	c1, _ := repo.Create(ctx, &e1)
	e2 := buildEntry(user.ID, "batchdel-b-"+uuid.New().String()[:8], nil)
	c2, _ := repo.Create(ctx, &e2)

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
		if _, err := repo.Create(ctx, &e); err != nil {
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
	created, _ := repo.Create(ctx, &e)
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
	created, _ := repo.Create(ctx, &e)

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
	created, _ := repo.Create(ctx, &e)

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
	created, _ := repo.Create(ctx, &e)

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
	created, _ := repo.Create(ctx, &e)

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
	created, _ := repo.Create(ctx, &e)

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
	created, _ := repo.Create(ctx, &e)

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
	created1, err := repo.Create(ctx, &e1)
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}
	_ = repo.SoftDelete(ctx, user.ID, created1.ID)

	// Re-create with same text should succeed (partial unique index).
	e2 := buildEntry(user.ID, text, nil)
	created2, err := repo.Create(ctx, &e2)
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
		created, err := repo.Create(ctx, &e)
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
	createdRecent, _ := repo.Create(ctx, &eRecent)
	_ = repo.SoftDelete(ctx, user.ID, createdRecent.ID)

	// Alive entry.
	eAlive := buildEntry(user.ID, "harddelalive-"+uuid.New().String()[:8], nil)
	if _, err := repo.Create(ctx, &eAlive); err != nil {
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
		if _, err := repo.Create(ctx, &e); err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
	}

	// Also soft-delete one to verify it's excluded.
	eDel := buildEntry(user.ID, "findall-del-"+suffix, nil)
	cDel, _ := repo.Create(ctx, &eDel)
	_ = repo.SoftDelete(ctx, user.ID, cDel.ID)

	entries, totalCount, err := repo.Find(ctx, user.ID, domain.EntryFilter{})
	if err != nil {
		t.Fatalf("Find: unexpected error: %v", err)
	}

	if totalCount < 3 {
		t.Errorf("expected TotalCount >= 3, got %d", totalCount)
	}
	if len(entries) < 3 {
		t.Errorf("expected at least 3 entries, got %d", len(entries))
	}

	// Verify soft-deleted entry is not in results.
	for _, e := range entries {
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

	entries, totalCount, err := repo.Find(ctx, user.ID, domain.EntryFilter{})
	if err != nil {
		t.Fatalf("Find: unexpected error: %v", err)
	}

	if totalCount != 0 {
		t.Errorf("expected TotalCount 0, got %d", totalCount)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
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
		if _, err := repo.Create(ctx, &e); err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
		// Small sleep to ensure distinct created_at.
		time.Sleep(time.Millisecond)
	}

	// Page 1: limit 2, offset 0.
	entries1, totalCount1, err := repo.Find(ctx, user.ID, domain.EntryFilter{
		Limit:  2,
		Offset: intPtr(0),
	})
	if err != nil {
		t.Fatalf("Find page 1: %v", err)
	}

	if len(entries1) != 2 {
		t.Fatalf("page 1: expected 2 entries, got %d", len(entries1))
	}

	// TotalCount should reflect all matching entries (not just page).
	if totalCount1 < 5 {
		t.Errorf("totalCount should be >= 5, got %d", totalCount1)
	}

	// Page 2: limit 2, offset 2.
	entries2, _, err := repo.Find(ctx, user.ID, domain.EntryFilter{
		Limit:  2,
		Offset: intPtr(2),
	})
	if err != nil {
		t.Fatalf("Find page 2: %v", err)
	}

	if len(entries2) != 2 {
		t.Fatalf("page 2: expected 2 entries, got %d", len(entries2))
	}

	// Pages should not overlap.
	for _, e1 := range entries1 {
		for _, e2 := range entries2 {
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
		if _, err := repo.Create(ctx, &e); err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
	}

	// Fetch with limit 2.
	entries, totalCount, err := repo.Find(ctx, user.ID, domain.EntryFilter{Limit: 2})
	if err != nil {
		t.Fatalf("Find: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
	if totalCount < 5 {
		t.Errorf("totalCount should be >= 5, got %d", totalCount)
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
	if _, err := repo.Create(ctx, &e1); err != nil {
		t.Fatalf("Create elephant: %v", err)
	}
	e2 := buildEntry(user.ID, "elephantine-"+suffix, nil)
	if _, err := repo.Create(ctx, &e2); err != nil {
		t.Fatalf("Create elephantine: %v", err)
	}
	e3 := buildEntry(user.ID, "zebra-"+suffix, nil)
	if _, err := repo.Create(ctx, &e3); err != nil {
		t.Fatalf("Create zebra: %v", err)
	}

	search := "elephant-" + suffix
	entries, totalCount, err := repo.Find(ctx, user.ID, domain.EntryFilter{Search: &search})
	if err != nil {
		t.Fatalf("Find with search: %v", err)
	}

	// Should find exactly 1 entry matching the exact ILIKE pattern.
	if totalCount != 1 {
		t.Errorf("expected totalCount 1, got %d", totalCount)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].TextNormalized != domain.NormalizeText("elephant-"+suffix) {
		t.Errorf("expected elephant entry, got %q", entries[0].TextNormalized)
	}
}

func TestRepo_Find_EmptySearchIgnored(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	suffix := uuid.New().String()[:8]

	e := buildEntry(user.ID, "emptysearch-"+suffix, nil)
	if _, err := repo.Create(ctx, &e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	empty := ""
	_, totalCount, err := repo.Find(ctx, user.ID, domain.EntryFilter{Search: &empty})
	if err != nil {
		t.Fatalf("Find with empty search: %v", err)
	}

	if totalCount < 1 {
		t.Errorf("expected at least 1 entry, got %d", totalCount)
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
	cWithCard, _ := repo.Create(ctx, &eWithCard)
	seedCard(t, pool, user.ID, cWithCard.ID, domain.CardStateNew)

	// Entry without card.
	eNoCard := buildEntry(user.ID, "hascard-no-"+suffix, nil)
	repo.Create(ctx, &eNoCard)

	// Filter: HasCard = true.
	hasCard := true
	search := suffix
	_, totalCount, err := repo.Find(ctx, user.ID, domain.EntryFilter{HasCard: &hasCard, Search: &search})
	if err != nil {
		t.Fatalf("Find HasCard=true: %v", err)
	}
	if totalCount != 1 {
		t.Errorf("HasCard=true: expected 1, got %d", totalCount)
	}

	// Filter: HasCard = false.
	noCard := false
	_, totalCount2, err := repo.Find(ctx, user.ID, domain.EntryFilter{HasCard: &noCard, Search: &search})
	if err != nil {
		t.Fatalf("Find HasCard=false: %v", err)
	}
	if totalCount2 != 1 {
		t.Errorf("HasCard=false: expected 1, got %d", totalCount2)
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
	cNoun, _ := repo.Create(ctx, &eNoun)
	posNoun := domain.PartOfSpeechNoun
	seedSense(t, pool, cNoun.ID, &posNoun)

	// Entry with VERB sense.
	eVerb := buildEntry(user.ID, "pos-verb-"+suffix, nil)
	cVerb, _ := repo.Create(ctx, &eVerb)
	posVerb := domain.PartOfSpeechVerb
	seedSense(t, pool, cVerb.ID, &posVerb)

	// Filter: NOUN.
	posFilter := domain.PartOfSpeechNoun
	search := suffix
	entries, totalCount, err := repo.Find(ctx, user.ID, domain.EntryFilter{PartOfSpeech: &posFilter, Search: &search})
	if err != nil {
		t.Fatalf("Find POS=NOUN: %v", err)
	}
	if totalCount != 1 {
		t.Errorf("POS=NOUN: expected 1, got %d", totalCount)
	}
	if len(entries) == 1 && entries[0].ID != cNoun.ID {
		t.Errorf("expected noun entry, got %s", entries[0].ID)
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
	cInTopic, _ := repo.Create(ctx, &eInTopic)
	linkEntryTopic(t, pool, cInTopic.ID, topicID)

	// Entry not in topic.
	eNotInTopic := buildEntry(user.ID, "topic-out-"+suffix, nil)
	repo.Create(ctx, &eNotInTopic)

	search := suffix
	_, totalCount, err := repo.Find(ctx, user.ID, domain.EntryFilter{TopicID: &topicID, Search: &search})
	if err != nil {
		t.Fatalf("Find TopicID: %v", err)
	}
	if totalCount != 1 {
		t.Errorf("TopicID filter: expected 1, got %d", totalCount)
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
	cNew, _ := repo.Create(ctx, &eNew)
	seedCard(t, pool, user.ID, cNew.ID, domain.CardStateNew)

	// Entry with LEARNING card.
	eLearning := buildEntry(user.ID, "status-learning-"+suffix, nil)
	cLearning, _ := repo.Create(ctx, &eLearning)
	seedCard(t, pool, user.ID, cLearning.ID, domain.CardStateLearning)

	// Filter: NEW.
	statusNew := domain.CardStateNew
	search := suffix
	_, totalCount, err := repo.Find(ctx, user.ID, domain.EntryFilter{Status: &statusNew, Search: &search})
	if err != nil {
		t.Fatalf("Find Status=NEW: %v", err)
	}
	if totalCount != 1 {
		t.Errorf("Status=NEW: expected 1, got %d", totalCount)
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
	cMatch, _ := repo.Create(ctx, &eMatch)
	seedCard(t, pool, user.ID, cMatch.ID, domain.CardStateNew)
	posNoun := domain.PartOfSpeechNoun
	seedSense(t, pool, cMatch.ID, &posNoun)
	linkEntryTopic(t, pool, cMatch.ID, topicID)

	// Entry that matches only some filters.
	ePartial := buildEntry(user.ID, "combined-partial-"+suffix, nil)
	cPartial, _ := repo.Create(ctx, &ePartial)
	posVerb := domain.PartOfSpeechVerb
	seedSense(t, pool, cPartial.ID, &posVerb) // wrong POS

	hasCard := true
	pos := domain.PartOfSpeechNoun
	status := domain.CardStateNew
	search := suffix

	entries, totalCount, err := repo.Find(ctx, user.ID, domain.EntryFilter{
		Search:       &search,
		HasCard:      &hasCard,
		PartOfSpeech: &pos,
		TopicID:      &topicID,
		Status:       &status,
	})
	if err != nil {
		t.Fatalf("Find combined: %v", err)
	}
	if totalCount != 1 {
		t.Errorf("combined filters: expected 1, got %d", totalCount)
	}
	if len(entries) == 1 && entries[0].ID != cMatch.ID {
		t.Errorf("expected matching entry %s, got %s", cMatch.ID, entries[0].ID)
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
		if _, err := repo.Create(ctx, &e); err != nil {
			t.Fatalf("Create %s: %v", text, err)
		}
	}

	search := suffix
	entries, _, err := repo.Find(ctx, user.ID, domain.EntryFilter{
		Search:    &search,
		SortBy:    "text",
		SortOrder: "ASC",
	})
	if err != nil {
		t.Fatalf("Find sort by text: %v", err)
	}

	if len(entries) < 3 {
		t.Fatalf("expected at least 3 entries, got %d", len(entries))
	}

	// Verify ascending text order.
	for i := 1; i < len(entries); i++ {
		if entries[i].TextNormalized < entries[i-1].TextNormalized {
			t.Errorf("entries not sorted by text ASC: %q < %q at index %d",
				entries[i].TextNormalized, entries[i-1].TextNormalized, i)
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
		if _, err := repo.Create(ctx, &e); err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
		time.Sleep(time.Millisecond)
	}

	search := "sortcreated-" + suffix
	entries, _, err := repo.Find(ctx, user.ID, domain.EntryFilter{
		Search:    &search,
		SortBy:    "created_at",
		SortOrder: "DESC",
	})
	if err != nil {
		t.Fatalf("Find sort DESC: %v", err)
	}

	// Verify descending created_at order.
	for i := 1; i < len(entries); i++ {
		if entries[i].CreatedAt.After(entries[i-1].CreatedAt) {
			t.Errorf("entries not sorted by created_at DESC at index %d", i)
		}
	}
}

// ---------------------------------------------------------------------------
// FindCursor tests: cursor-based pagination
// ---------------------------------------------------------------------------

func TestRepo_FindCursor_Pagination(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	suffix := uuid.New().String()[:8]

	// Create 5 entries with distinct timestamps.
	for i := 0; i < 5; i++ {
		e := buildEntry(user.ID, "cursor-"+suffix+"-"+uuid.New().String()[:4], nil)
		if _, err := repo.Create(ctx, &e); err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
		time.Sleep(2 * time.Millisecond)
	}

	search := "cursor-" + suffix

	// First page: limit 2, no cursor.
	entries1, hasNext1, err := repo.FindCursor(ctx, user.ID, domain.EntryFilter{
		Search:    &search,
		SortBy:    "created_at",
		SortOrder: "DESC",
		Limit:     2,
	})
	if err != nil {
		t.Fatalf("FindCursor page 1: %v", err)
	}
	if len(entries1) != 2 {
		t.Fatalf("page 1: expected 2, got %d", len(entries1))
	}
	// With 5 entries total, first page of 2 should have more.
	if !hasNext1 {
		t.Error("page 1: expected hasNextPage=true")
	}

	// Create cursor from last entry of page 1.
	lastEntry := entries1[len(entries1)-1]
	cursor := entry.CursorFromEntry(lastEntry, "created_at")

	// Second page: limit 2, with cursor.
	entries2, hasNext2, err := repo.FindCursor(ctx, user.ID, domain.EntryFilter{
		Search:    &search,
		SortBy:    "created_at",
		SortOrder: "DESC",
		Limit:     2,
		Cursor:    &cursor,
	})
	if err != nil {
		t.Fatalf("FindCursor page 2: %v", err)
	}
	if len(entries2) != 2 {
		t.Fatalf("page 2: expected 2, got %d", len(entries2))
	}
	if !hasNext2 {
		t.Error("page 2: expected hasNextPage=true")
	}

	// Verify no overlap between pages.
	page1IDs := make(map[uuid.UUID]bool)
	for _, e := range entries1 {
		page1IDs[e.ID] = true
	}
	for _, e := range entries2 {
		if page1IDs[e.ID] {
			t.Error("cursor pagination: pages should not overlap")
		}
	}

	// Third page: should have 1 entry and hasNextPage=false.
	lastEntry2 := entries2[len(entries2)-1]
	cursor2 := entry.CursorFromEntry(lastEntry2, "created_at")

	entries3, hasNext3, err := repo.FindCursor(ctx, user.ID, domain.EntryFilter{
		Search:    &search,
		SortBy:    "created_at",
		SortOrder: "DESC",
		Limit:     2,
		Cursor:    &cursor2,
	})
	if err != nil {
		t.Fatalf("FindCursor page 3: %v", err)
	}
	if len(entries3) != 1 {
		t.Errorf("page 3: expected 1, got %d", len(entries3))
	}
	if hasNext3 {
		t.Error("page 3: expected hasNextPage=false")
	}
}

func TestRepo_FindCursor_TextSort(t *testing.T) {
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
		if _, err := repo.Create(ctx, &e); err != nil {
			t.Fatalf("Create %s: %v", text, err)
		}
	}

	search := "cursor-text-" + suffix

	// First page: limit 1, sort by text ASC.
	entries1, _, err := repo.FindCursor(ctx, user.ID, domain.EntryFilter{
		Search:    &search,
		SortBy:    "text",
		SortOrder: "ASC",
		Limit:     1,
	})
	if err != nil {
		t.Fatalf("FindCursor text page 1: %v", err)
	}
	if len(entries1) != 1 {
		t.Fatalf("expected 1, got %d", len(entries1))
	}
	if entries1[0].TextNormalized != domain.NormalizeText("aaa-cursor-text-"+suffix) {
		t.Errorf("expected aaa entry, got %q", entries1[0].TextNormalized)
	}

	// Second page with cursor.
	cursor := entry.CursorFromEntry(entries1[0], "text")
	entries2, _, err := repo.FindCursor(ctx, user.ID, domain.EntryFilter{
		Search:    &search,
		SortBy:    "text",
		SortOrder: "ASC",
		Limit:     1,
		Cursor:    &cursor,
	})
	if err != nil {
		t.Fatalf("FindCursor text page 2: %v", err)
	}
	if len(entries2) != 1 {
		t.Fatalf("expected 1, got %d", len(entries2))
	}
	if entries2[0].TextNormalized != domain.NormalizeText("bbb-cursor-text-"+suffix) {
		t.Errorf("expected bbb entry, got %q", entries2[0].TextNormalized)
	}
}

func TestRepo_FindCursor_InvalidCursor(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)

	// Test with garbage cursor.
	badCursor := "not-valid-base64!!!"
	_, _, err := repo.FindCursor(ctx, user.ID, domain.EntryFilter{Cursor: &badCursor})
	assertIsDomainError(t, err, domain.ErrValidation)

	// Test with valid base64 but bad format (no pipe).
	badCursor2 := "bm9waXBl" // base64("nopipe")
	_, _, err = repo.FindCursor(ctx, user.ID, domain.EntryFilter{Cursor: &badCursor2})
	assertIsDomainError(t, err, domain.ErrValidation)

	// Test with valid base64, pipe, but bad UUID.
	badCursor3 := entry.EncodeCursor("2024-01-01T00:00:00Z", "not-a-uuid")
	_, _, err = repo.FindCursor(ctx, user.ID, domain.EntryFilter{Cursor: &badCursor3})
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
	_, _, err := repo.Find(ctx, user.ID, domain.EntryFilter{Limit: 0})
	if err != nil {
		t.Fatalf("Find: %v", err)
	}

	// With negative limit, should use default.
	_, _, err = repo.Find(ctx, user.ID, domain.EntryFilter{Limit: -5})
	if err != nil {
		t.Fatalf("Find negative limit: %v", err)
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
