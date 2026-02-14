package image_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/image"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/testhelper"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// seedRefImage creates a ref_image for a given ref_entry directly in the DB.
func seedRefImage(t *testing.T, pool *pgxpool.Pool, refEntryID uuid.UUID, url, caption, sourceSlug string) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	id := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO ref_images (id, ref_entry_id, url, caption, source_slug) VALUES ($1, $2, $3, $4, $5)`,
		id, refEntryID, url, caption, sourceSlug,
	)
	require.NoError(t, err)

	return id
}

// ---------------------------------------------------------------------------
// Catalog image tests
// ---------------------------------------------------------------------------

func TestRepo_LinkCatalog_AndGet(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupTestDB(t)
	repo := image.New(pool)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "linkcat-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	refImageID := seedRefImage(t, pool, refEntry.ID, "https://example.com/img1.jpg", "Test caption", "test-source")

	// Link catalog image.
	err := repo.LinkCatalog(ctx, entry.ID, refImageID)
	require.NoError(t, err)

	// Get should return exactly 1 image.
	got, err := repo.GetCatalogByEntryID(ctx, entry.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, refImageID, got[0].ID)
	assert.Equal(t, refEntry.ID, got[0].RefEntryID)
	assert.Equal(t, "https://example.com/img1.jpg", got[0].URL)
	assert.NotNil(t, got[0].Caption)
	assert.Equal(t, "Test caption", *got[0].Caption)
	assert.Equal(t, "test-source", got[0].SourceSlug)
}

func TestRepo_LinkCatalog_Idempotent(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupTestDB(t)
	repo := image.New(pool)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "idemcat-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	refImageID := seedRefImage(t, pool, refEntry.ID, "https://example.com/idem.jpg", "Idem caption", "test-source")

	// Link twice -- should NOT error.
	err := repo.LinkCatalog(ctx, entry.ID, refImageID)
	require.NoError(t, err)

	err = repo.LinkCatalog(ctx, entry.ID, refImageID)
	require.NoError(t, err)

	// Only 1 link should exist.
	got, err := repo.GetCatalogByEntryID(ctx, entry.ID)
	require.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestRepo_UnlinkCatalog(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupTestDB(t)
	repo := image.New(pool)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "unlinkcat-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	refImageID := seedRefImage(t, pool, refEntry.ID, "https://example.com/unlink.jpg", "Unlink caption", "test-source")

	// Link then unlink.
	err := repo.LinkCatalog(ctx, entry.ID, refImageID)
	require.NoError(t, err)

	err = repo.UnlinkCatalog(ctx, entry.ID, refImageID)
	require.NoError(t, err)

	got, err := repo.GetCatalogByEntryID(ctx, entry.ID)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestRepo_GetCatalogByEntryIDs_Batch(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupTestDB(t)
	repo := image.New(pool)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)

	// Create 2 entries with different ref_entries.
	refEntry1 := testhelper.SeedRefEntry(t, pool, "batchcat1-"+uuid.New().String()[:8])
	refEntry2 := testhelper.SeedRefEntry(t, pool, "batchcat2-"+uuid.New().String()[:8])
	entry1 := testhelper.SeedEntry(t, pool, user.ID, refEntry1.ID)
	entry2 := testhelper.SeedEntry(t, pool, user.ID, refEntry2.ID)

	// Create ref_images and link them.
	refImg1 := seedRefImage(t, pool, refEntry1.ID, "https://example.com/batch1.jpg", "Batch 1", "src1")
	refImg2 := seedRefImage(t, pool, refEntry2.ID, "https://example.com/batch2.jpg", "Batch 2", "src2")

	err := repo.LinkCatalog(ctx, entry1.ID, refImg1)
	require.NoError(t, err)
	err = repo.LinkCatalog(ctx, entry2.ID, refImg2)
	require.NoError(t, err)

	got, err := repo.GetCatalogByEntryIDs(ctx, []uuid.UUID{entry1.ID, entry2.ID})
	require.NoError(t, err)
	assert.Len(t, got, 2)

	// Verify grouping: each result has correct EntryID.
	entry1Count := 0
	entry2Count := 0
	for _, img := range got {
		switch img.EntryID {
		case entry1.ID:
			entry1Count++
			assert.Equal(t, refImg1, img.ID)
		case entry2.ID:
			entry2Count++
			assert.Equal(t, refImg2, img.ID)
		default:
			t.Fatalf("unexpected entry_id: %s", img.EntryID)
		}
	}
	assert.Equal(t, 1, entry1Count, "entry1 should have 1 catalog image")
	assert.Equal(t, 1, entry2Count, "entry2 should have 1 catalog image")
}

func TestRepo_GetCatalogByEntryID_Empty(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupTestDB(t)
	repo := image.New(pool)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	entry := testhelper.SeedEntryCustom(t, pool, user.ID)

	got, err := repo.GetCatalogByEntryID(ctx, entry.ID)
	require.NoError(t, err)
	assert.NotNil(t, got, "should return empty slice, not nil")
	assert.Empty(t, got)
}

// ---------------------------------------------------------------------------
// User image tests
// ---------------------------------------------------------------------------

func TestRepo_CreateUser_AndGet(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupTestDB(t)
	repo := image.New(pool)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	entry := testhelper.SeedEntryCustom(t, pool, user.ID)

	caption := "My photo"
	created, err := repo.CreateUser(ctx, entry.ID, "https://example.com/user-img.jpg", &caption)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, created.ID)
	assert.Equal(t, entry.ID, created.EntryID)
	assert.Equal(t, "https://example.com/user-img.jpg", created.URL)
	assert.NotNil(t, created.Caption)
	assert.Equal(t, "My photo", *created.Caption)
	assert.False(t, created.CreatedAt.IsZero())

	// Get should return the created image.
	got, err := repo.GetUserByEntryID(ctx, entry.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, created.ID, got[0].ID)
	assert.Equal(t, created.URL, got[0].URL)
	assert.Equal(t, *created.Caption, *got[0].Caption)
}

func TestRepo_DeleteUser(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupTestDB(t)
	repo := image.New(pool)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	entry := testhelper.SeedEntryCustom(t, pool, user.ID)

	created, err := repo.CreateUser(ctx, entry.ID, "https://example.com/del.jpg", nil)
	require.NoError(t, err)

	err = repo.DeleteUser(ctx, created.ID)
	require.NoError(t, err)

	// Should be gone.
	got, err := repo.GetUserByEntryID(ctx, entry.ID)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestRepo_DeleteUser_NotFound(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupTestDB(t)
	repo := image.New(pool)
	ctx := context.Background()

	err := repo.DeleteUser(ctx, uuid.New())
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrNotFound), "expected ErrNotFound, got: %v", err)
}

func TestRepo_GetUserByEntryIDs_Batch(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupTestDB(t)
	repo := image.New(pool)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	entry1 := testhelper.SeedEntryCustom(t, pool, user.ID)
	entry2 := testhelper.SeedEntryCustom(t, pool, user.ID)

	_, err := repo.CreateUser(ctx, entry1.ID, "https://example.com/u1.jpg", nil)
	require.NoError(t, err)
	_, err = repo.CreateUser(ctx, entry2.ID, "https://example.com/u2.jpg", nil)
	require.NoError(t, err)

	got, err := repo.GetUserByEntryIDs(ctx, []uuid.UUID{entry1.ID, entry2.ID})
	require.NoError(t, err)
	assert.Len(t, got, 2)

	// Verify grouping.
	entry1Count := 0
	entry2Count := 0
	for _, img := range got {
		switch img.EntryID {
		case entry1.ID:
			entry1Count++
		case entry2.ID:
			entry2Count++
		default:
			t.Fatalf("unexpected entry_id: %s", img.EntryID)
		}
	}
	assert.Equal(t, 1, entry1Count, "entry1 should have 1 user image")
	assert.Equal(t, 1, entry2Count, "entry2 should have 1 user image")
}

func TestRepo_GetUserByEntryID_Empty(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupTestDB(t)
	repo := image.New(pool)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	entry := testhelper.SeedEntryCustom(t, pool, user.ID)

	got, err := repo.GetUserByEntryID(ctx, entry.ID)
	require.NoError(t, err)
	assert.NotNil(t, got, "should return empty slice, not nil")
	assert.Empty(t, got)
}
