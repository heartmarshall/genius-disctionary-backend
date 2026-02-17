package topic_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/testhelper"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/topic"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// newRepo sets up a test DB and returns a ready Repo + pool.
func newRepo(t *testing.T) (*topic.Repo, *pgxpool.Pool) {
	t.Helper()
	pool := testhelper.SetupTestDB(t)
	return topic.New(pool), pool
}

// ---------------------------------------------------------------------------
// Create + GetByID tests
// ---------------------------------------------------------------------------

func TestRepo_Create_AndGetByID(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	desc := "words from travel"
	created, err := repo.Create(ctx, user.ID, &domain.Topic{Name: "Travel", Description: &desc})
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("expected non-nil topic ID")
	}
	if created.UserID != user.ID {
		t.Errorf("UserID mismatch: got %s, want %s", created.UserID, user.ID)
	}
	if created.Name != "Travel" {
		t.Errorf("Name mismatch: got %q, want %q", created.Name, "Travel")
	}
	if created.Description == nil || *created.Description != "words from travel" {
		t.Errorf("Description mismatch: got %v, want %q", created.Description, "words from travel")
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if created.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}

	// GetByID round-trip.
	got, err := repo.GetByID(ctx, user.ID, created.ID)
	if err != nil {
		t.Fatalf("GetByID: unexpected error: %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, created.ID)
	}
	if got.Name != created.Name {
		t.Errorf("Name mismatch: got %q, want %q", got.Name, created.Name)
	}
	if got.Description == nil || *got.Description != *created.Description {
		t.Errorf("Description mismatch: got %v, want %v", got.Description, created.Description)
	}
}

func TestRepo_Create_NilDescription(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	created, err := repo.Create(ctx, user.ID, &domain.Topic{Name: "NoDesc-" + uuid.New().String()[:8]})
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	if created.Description != nil {
		t.Errorf("expected nil Description, got %v", created.Description)
	}
}

func TestRepo_Create_DuplicateName(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	name := "Unique-" + uuid.New().String()[:8]
	_, err := repo.Create(ctx, user.ID, &domain.Topic{Name: name})
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}

	// Same name, same user -> ErrAlreadyExists.
	_, err = repo.Create(ctx, user.ID, &domain.Topic{Name: name})
	assertIsDomainError(t, err, domain.ErrAlreadyExists)
}

func TestRepo_Create_SameNameDifferentUsers(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user1 := testhelper.SeedUser(t, pool)
	user2 := testhelper.SeedUser(t, pool)

	name := "SharedName-" + uuid.New().String()[:8]
	_, err := repo.Create(ctx, user1.ID, &domain.Topic{Name: name})
	if err != nil {
		t.Fatalf("Create user1: %v", err)
	}

	// Same name but different user -> should succeed.
	_, err = repo.Create(ctx, user2.ID, &domain.Topic{Name: name})
	if err != nil {
		t.Fatalf("Create user2: expected success, got: %v", err)
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

func TestRepo_GetByID_WrongUser(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user1 := testhelper.SeedUser(t, pool)
	user2 := testhelper.SeedUser(t, pool)

	created, err := repo.Create(ctx, user1.ID, &domain.Topic{Name: "Private-" + uuid.New().String()[:8]})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// user2 should not be able to access user1's topic.
	_, err = repo.GetByID(ctx, user2.ID, created.ID)
	assertIsDomainError(t, err, domain.ErrNotFound)
}

// ---------------------------------------------------------------------------
// ListByUser tests
// ---------------------------------------------------------------------------

func TestRepo_ListByUser(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	// Create topics with names that sort alphabetically: A < B < C.
	suffix := uuid.New().String()[:8]
	names := []string{"C-" + suffix, "A-" + suffix, "B-" + suffix}
	for _, name := range names {
		if _, err := repo.Create(ctx, user.ID, &domain.Topic{Name: name}); err != nil {
			t.Fatalf("Create %q: %v", name, err)
		}
	}

	got, err := repo.List(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListByUser: unexpected error: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 topics, got %d", len(got))
	}

	// Verify alphabetical order.
	if got[0].Name != "A-"+suffix {
		t.Errorf("expected first topic %q, got %q", "A-"+suffix, got[0].Name)
	}
	if got[1].Name != "B-"+suffix {
		t.Errorf("expected second topic %q, got %q", "B-"+suffix, got[1].Name)
	}
	if got[2].Name != "C-"+suffix {
		t.Errorf("expected third topic %q, got %q", "C-"+suffix, got[2].Name)
	}
}

func TestRepo_ListByUser_Empty(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	got, err := repo.List(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListByUser: unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected 0 topics, got %d", len(got))
	}
}

// ---------------------------------------------------------------------------
// Update tests
// ---------------------------------------------------------------------------

func TestRepo_Update(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	desc := "original description"
	created, err := repo.Create(ctx, user.ID, &domain.Topic{Name: "Original-" + uuid.New().String()[:8], Description: &desc})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	newName := "Updated-" + uuid.New().String()[:8]
	newDesc := "updated description"
	_, err = repo.Update(ctx, user.ID, created.ID, domain.TopicUpdateParams{Name: &newName, Description: &newDesc})
	if err != nil {
		t.Fatalf("Update: unexpected error: %v", err)
	}

	// Verify update via GetByID.
	got, err := repo.GetByID(ctx, user.ID, created.ID)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}

	if got.Name != newName {
		t.Errorf("Name mismatch: got %q, want %q", got.Name, newName)
	}
	if got.Description == nil || *got.Description != newDesc {
		t.Errorf("Description mismatch: got %v, want %q", got.Description, newDesc)
	}
	if !got.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("expected UpdatedAt to advance after update: got %s, created %s", got.UpdatedAt, created.UpdatedAt)
	}
}

func TestRepo_Update_NotFound(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	name := "name"
	_, err := repo.Update(ctx, user.ID, uuid.New(), domain.TopicUpdateParams{Name: &name})
	assertIsDomainError(t, err, domain.ErrNotFound)
}

func TestRepo_Update_WrongUser(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user1 := testhelper.SeedUser(t, pool)
	user2 := testhelper.SeedUser(t, pool)

	created, err := repo.Create(ctx, user1.ID, &domain.Topic{Name: "UpdWrong-" + uuid.New().String()[:8]})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// user2 should not be able to update user1's topic.
	hackedName := "hacked"
	_, err = repo.Update(ctx, user2.ID, created.ID, domain.TopicUpdateParams{Name: &hackedName})
	assertIsDomainError(t, err, domain.ErrNotFound)
}

// ---------------------------------------------------------------------------
// Delete tests
// ---------------------------------------------------------------------------

func TestRepo_Delete(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	created, err := repo.Create(ctx, user.ID, &domain.Topic{Name: "ToDelete-" + uuid.New().String()[:8]})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	err = repo.Delete(ctx, user.ID, created.ID)
	if err != nil {
		t.Fatalf("Delete: unexpected error: %v", err)
	}

	// Verify topic is gone.
	_, err = repo.GetByID(ctx, user.ID, created.ID)
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

func TestRepo_Delete_CascadeEntryTopics(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "cascade-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	created, err := repo.Create(ctx, user.ID, &domain.Topic{Name: "CascTopic-" + uuid.New().String()[:8]})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Link entry to topic.
	err = repo.LinkEntry(ctx, entry.ID, created.ID)
	if err != nil {
		t.Fatalf("LinkEntry: %v", err)
	}

	// Verify link exists.
	entryIDs, err := repo.GetEntryIDsByTopicID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetEntryIDsByTopicID: %v", err)
	}
	if len(entryIDs) != 1 {
		t.Fatalf("expected 1 entry linked, got %d", len(entryIDs))
	}

	// Delete topic -> CASCADE should remove entry_topics rows.
	err = repo.Delete(ctx, user.ID, created.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify entry still exists.
	var entryExists bool
	err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM entries WHERE id = $1)", entry.ID).Scan(&entryExists)
	if err != nil {
		t.Fatalf("check entry exists: %v", err)
	}
	if !entryExists {
		t.Error("entry should still exist after topic deletion")
	}

	// Verify entry_topics link is gone (query raw since topic is deleted).
	var linkCount int
	err = pool.QueryRow(ctx,
		"SELECT count(*) FROM entry_topics WHERE entry_id = $1 AND topic_id = $2",
		entry.ID, created.ID,
	).Scan(&linkCount)
	if err != nil {
		t.Fatalf("check entry_topics: %v", err)
	}
	if linkCount != 0 {
		t.Errorf("expected 0 entry_topics rows after topic delete, got %d", linkCount)
	}
}

// ---------------------------------------------------------------------------
// LinkEntry + GetByEntryID tests
// ---------------------------------------------------------------------------

func TestRepo_LinkEntry_AndGetByEntryID(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "link-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	topic1, err := repo.Create(ctx, user.ID, &domain.Topic{Name: "B-Topic-" + uuid.New().String()[:8]})
	if err != nil {
		t.Fatalf("Create topic1: %v", err)
	}
	topic2, err := repo.Create(ctx, user.ID, &domain.Topic{Name: "A-Topic-" + uuid.New().String()[:8]})
	if err != nil {
		t.Fatalf("Create topic2: %v", err)
	}

	// Link entry to both topics.
	if err := repo.LinkEntry(ctx, entry.ID, topic1.ID); err != nil {
		t.Fatalf("LinkEntry topic1: %v", err)
	}
	if err := repo.LinkEntry(ctx, entry.ID, topic2.ID); err != nil {
		t.Fatalf("LinkEntry topic2: %v", err)
	}

	// GetByEntryID should return both topics sorted by name.
	got, err := repo.GetTopicsByEntryID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("GetByEntryID: unexpected error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 topics, got %d", len(got))
	}

	// topic2 name starts with "A-", topic1 starts with "B-", so topic2 should come first.
	if got[0].ID != topic2.ID {
		t.Errorf("expected first topic to be %s (A-*), got %s", topic2.ID, got[0].ID)
	}
	if got[1].ID != topic1.ID {
		t.Errorf("expected second topic to be %s (B-*), got %s", topic1.ID, got[1].ID)
	}
}

func TestRepo_LinkEntry_Idempotent(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "idem-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	created, err := repo.Create(ctx, user.ID, &domain.Topic{Name: "Idem-" + uuid.New().String()[:8]})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Link twice — should NOT error.
	err = repo.LinkEntry(ctx, entry.ID, created.ID)
	if err != nil {
		t.Fatalf("LinkEntry first: %v", err)
	}

	err = repo.LinkEntry(ctx, entry.ID, created.ID)
	if err != nil {
		t.Fatalf("LinkEntry second (idempotent): %v", err)
	}

	// Only 1 link should exist.
	got, err := repo.GetTopicsByEntryID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("GetByEntryID: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 topic linked, got %d", len(got))
	}
}

// ---------------------------------------------------------------------------
// UnlinkEntry tests
// ---------------------------------------------------------------------------

func TestRepo_UnlinkEntry(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "unlink-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	created, err := repo.Create(ctx, user.ID, &domain.Topic{Name: "Unlink-" + uuid.New().String()[:8]})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Link then unlink.
	err = repo.LinkEntry(ctx, entry.ID, created.ID)
	if err != nil {
		t.Fatalf("LinkEntry: %v", err)
	}

	err = repo.UnlinkEntry(ctx, entry.ID, created.ID)
	if err != nil {
		t.Fatalf("UnlinkEntry: unexpected error: %v", err)
	}

	// Verify link is gone.
	got, err := repo.GetTopicsByEntryID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("GetByEntryID: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 topics after unlink, got %d", len(got))
	}
}

func TestRepo_UnlinkEntry_NonExisting(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "unlinkne-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	created, err := repo.Create(ctx, user.ID, &domain.Topic{Name: "UnlinkNE-" + uuid.New().String()[:8]})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Unlink without ever linking — should NOT error.
	err = repo.UnlinkEntry(ctx, entry.ID, created.ID)
	if err != nil {
		t.Fatalf("UnlinkEntry (non-existing): unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetByEntryIDs batch tests
// ---------------------------------------------------------------------------

func TestRepo_GetByEntryIDs_Batch(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry1 := testhelper.SeedRefEntry(t, pool, "batch1-"+uuid.New().String()[:8])
	refEntry2 := testhelper.SeedRefEntry(t, pool, "batch2-"+uuid.New().String()[:8])
	entry1 := testhelper.SeedEntry(t, pool, user.ID, refEntry1.ID)
	entry2 := testhelper.SeedEntry(t, pool, user.ID, refEntry2.ID)

	topicA, err := repo.Create(ctx, user.ID, &domain.Topic{Name: "BatchA-" + uuid.New().String()[:8]})
	if err != nil {
		t.Fatalf("Create topicA: %v", err)
	}
	topicB, err := repo.Create(ctx, user.ID, &domain.Topic{Name: "BatchB-" + uuid.New().String()[:8]})
	if err != nil {
		t.Fatalf("Create topicB: %v", err)
	}

	// entry1 -> topicA, topicB; entry2 -> topicB.
	if err := repo.LinkEntry(ctx, entry1.ID, topicA.ID); err != nil {
		t.Fatalf("LinkEntry entry1-topicA: %v", err)
	}
	if err := repo.LinkEntry(ctx, entry1.ID, topicB.ID); err != nil {
		t.Fatalf("LinkEntry entry1-topicB: %v", err)
	}
	if err := repo.LinkEntry(ctx, entry2.ID, topicB.ID); err != nil {
		t.Fatalf("LinkEntry entry2-topicB: %v", err)
	}

	got, err := repo.GetByEntryIDs(ctx, []uuid.UUID{entry1.ID, entry2.ID})
	if err != nil {
		t.Fatalf("GetByEntryIDs: unexpected error: %v", err)
	}

	// entry1 has 2 topics, entry2 has 1 topic => 3 total.
	if len(got) != 3 {
		t.Fatalf("expected 3 results, got %d", len(got))
	}

	// Group by entry_id.
	entry1Count := 0
	entry2Count := 0
	for _, tw := range got {
		switch tw.EntryID {
		case entry1.ID:
			entry1Count++
		case entry2.ID:
			entry2Count++
		default:
			t.Fatalf("unexpected entry_id: %s", tw.EntryID)
		}
	}

	if entry1Count != 2 {
		t.Errorf("expected 2 topics for entry1, got %d", entry1Count)
	}
	if entry2Count != 1 {
		t.Errorf("expected 1 topic for entry2, got %d", entry2Count)
	}
}

func TestRepo_GetByEntryIDs_Empty(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	got, err := repo.GetByEntryIDs(ctx, []uuid.UUID{})
	if err != nil {
		t.Fatalf("GetByEntryIDs empty: unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected 0 results, got %d", len(got))
	}
}

// ---------------------------------------------------------------------------
// GetEntryIDsByTopicID tests
// ---------------------------------------------------------------------------

func TestRepo_GetEntryIDsByTopicID(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry1 := testhelper.SeedRefEntry(t, pool, "eid1-"+uuid.New().String()[:8])
	refEntry2 := testhelper.SeedRefEntry(t, pool, "eid2-"+uuid.New().String()[:8])
	entry1 := testhelper.SeedEntry(t, pool, user.ID, refEntry1.ID)
	entry2 := testhelper.SeedEntry(t, pool, user.ID, refEntry2.ID)

	created, err := repo.Create(ctx, user.ID, &domain.Topic{Name: "EntryIDs-" + uuid.New().String()[:8]})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Link both entries to the topic.
	if err := repo.LinkEntry(ctx, entry1.ID, created.ID); err != nil {
		t.Fatalf("LinkEntry entry1: %v", err)
	}
	if err := repo.LinkEntry(ctx, entry2.ID, created.ID); err != nil {
		t.Fatalf("LinkEntry entry2: %v", err)
	}

	got, err := repo.GetEntryIDsByTopicID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetEntryIDsByTopicID: unexpected error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 entry IDs, got %d", len(got))
	}

	// Verify both entry IDs are present.
	idSet := make(map[uuid.UUID]bool)
	for _, id := range got {
		idSet[id] = true
	}

	if !idSet[entry1.ID] {
		t.Errorf("expected entry1 ID %s in results", entry1.ID)
	}
	if !idSet[entry2.ID] {
		t.Errorf("expected entry2 ID %s in results", entry2.ID)
	}
}

func TestRepo_GetEntryIDsByTopicID_Empty(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	created, err := repo.Create(ctx, user.ID, &domain.Topic{Name: "NoEntries-" + uuid.New().String()[:8]})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetEntryIDsByTopicID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetEntryIDsByTopicID: unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected 0 entry IDs, got %d", len(got))
	}
}

// ---------------------------------------------------------------------------
// GetByEntryID empty tests
// ---------------------------------------------------------------------------

func TestRepo_GetByEntryID_Empty(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "empty-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	got, err := repo.GetTopicsByEntryID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("GetByEntryID: unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected 0 topics, got %d", len(got))
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
