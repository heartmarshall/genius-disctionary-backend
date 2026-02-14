package pronunciation_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/pronunciation"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/testhelper"
)

func TestRepo_Link_AndGet(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupTestDB(t)
	repo := pronunciation.New(pool)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "pronounce-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	// SeedEntry already links pronunciations via M2M.
	// Unlink all first, then test our Link.
	err := repo.UnlinkAll(ctx, entry.ID)
	require.NoError(t, err)

	// Link the first ref pronunciation.
	err = repo.Link(ctx, entry.ID, refEntry.Pronunciations[0].ID)
	require.NoError(t, err)

	// Get should return exactly 1.
	got, err := repo.GetByEntryID(ctx, entry.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, refEntry.Pronunciations[0].ID, got[0].ID)
	assert.Equal(t, refEntry.Pronunciations[0].RefEntryID, got[0].RefEntryID)
	assert.NotNil(t, got[0].Transcription)
	assert.NotNil(t, got[0].SourceSlug)
}

func TestRepo_Link_Idempotent(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupTestDB(t)
	repo := pronunciation.New(pool)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "idem-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	err := repo.UnlinkAll(ctx, entry.ID)
	require.NoError(t, err)

	// Link twice — should NOT error.
	err = repo.Link(ctx, entry.ID, refEntry.Pronunciations[0].ID)
	require.NoError(t, err)

	err = repo.Link(ctx, entry.ID, refEntry.Pronunciations[0].ID)
	require.NoError(t, err)

	// Only 1 link should exist.
	got, err := repo.GetByEntryID(ctx, entry.ID)
	require.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestRepo_Unlink(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupTestDB(t)
	repo := pronunciation.New(pool)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "unlink-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	err := repo.UnlinkAll(ctx, entry.ID)
	require.NoError(t, err)

	// Link then unlink.
	err = repo.Link(ctx, entry.ID, refEntry.Pronunciations[0].ID)
	require.NoError(t, err)

	err = repo.Unlink(ctx, entry.ID, refEntry.Pronunciations[0].ID)
	require.NoError(t, err)

	got, err := repo.GetByEntryID(ctx, entry.ID)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestRepo_Unlink_NonExisting(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupTestDB(t)
	repo := pronunciation.New(pool)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "unlinkne-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	// Unlink a link that doesn't exist — should NOT error.
	err := repo.Unlink(ctx, entry.ID, uuid.New())
	require.NoError(t, err)

	// Also unlink using a valid ref_pronunciation that is not linked.
	err = repo.UnlinkAll(ctx, entry.ID)
	require.NoError(t, err)

	err = repo.Unlink(ctx, entry.ID, refEntry.Pronunciations[0].ID)
	require.NoError(t, err)
}

func TestRepo_UnlinkAll(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupTestDB(t)
	repo := pronunciation.New(pool)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "unlinkall-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	// SeedEntry links 2 pronunciations already. Verify.
	got, err := repo.GetByEntryID(ctx, entry.ID)
	require.NoError(t, err)
	require.Len(t, got, 2)

	// UnlinkAll.
	err = repo.UnlinkAll(ctx, entry.ID)
	require.NoError(t, err)

	got, err = repo.GetByEntryID(ctx, entry.ID)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestRepo_GetByEntryID_Empty(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupTestDB(t)
	repo := pronunciation.New(pool)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "empty-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	err := repo.UnlinkAll(ctx, entry.ID)
	require.NoError(t, err)

	got, err := repo.GetByEntryID(ctx, entry.ID)
	require.NoError(t, err)
	assert.NotNil(t, got, "should return empty slice, not nil")
	assert.Empty(t, got)
}

func TestRepo_GetByEntryIDs_Batch(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupTestDB(t)
	repo := pronunciation.New(pool)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry1 := testhelper.SeedRefEntry(t, pool, "batch1-"+uuid.New().String()[:8])
	refEntry2 := testhelper.SeedRefEntry(t, pool, "batch2-"+uuid.New().String()[:8])
	entry1 := testhelper.SeedEntry(t, pool, user.ID, refEntry1.ID)
	entry2 := testhelper.SeedEntry(t, pool, user.ID, refEntry2.ID)

	// Each SeedEntry links 2 pronunciations.
	got, err := repo.GetByEntryIDs(ctx, []uuid.UUID{entry1.ID, entry2.ID})
	require.NoError(t, err)

	// Total should be 4 (2 per entry).
	assert.Len(t, got, 4)

	// Verify grouping: each result has correct EntryID.
	entry1Count := 0
	entry2Count := 0
	for _, p := range got {
		switch p.EntryID {
		case entry1.ID:
			entry1Count++
		case entry2.ID:
			entry2Count++
		default:
			t.Fatalf("unexpected entry_id: %s", p.EntryID)
		}
	}
	assert.Equal(t, 2, entry1Count, "entry1 should have 2 pronunciations")
	assert.Equal(t, 2, entry2Count, "entry2 should have 2 pronunciations")
}

func TestRepo_GetByEntryIDs_Empty(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupTestDB(t)
	repo := pronunciation.New(pool)
	ctx := context.Background()

	got, err := repo.GetByEntryIDs(ctx, []uuid.UUID{})
	require.NoError(t, err)
	assert.NotNil(t, got, "should return empty slice, not nil")
	assert.Empty(t, got)
}

func TestRepo_Link_MultiplePronunciations(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupTestDB(t)
	repo := pronunciation.New(pool)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "multi-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	err := repo.UnlinkAll(ctx, entry.ID)
	require.NoError(t, err)

	// Link both pronunciations.
	for _, p := range refEntry.Pronunciations {
		err := repo.Link(ctx, entry.ID, p.ID)
		require.NoError(t, err)
	}

	got, err := repo.GetByEntryID(ctx, entry.ID)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}
