package translation_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	postgres "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/testhelper"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/translation"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// newRepo sets up a test DB and returns a ready Repo + pool.
func newRepo(t *testing.T) (*translation.Repo, *pgxpool.Pool) {
	t.Helper()
	pool := testhelper.SetupTestDB(t)
	txm := postgres.NewTxManager(pool)
	return translation.New(pool, txm), pool
}

// ---------------------------------------------------------------------------
// CreateFromRef + GetBySenseID tests
// ---------------------------------------------------------------------------

func TestRepo_CreateFromRef_AndGetBySenseID(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "tr-fromref-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	sense := entry.Senses[0]
	refTranslation := refEntry.Senses[0].Translations[0]

	got, err := repo.CreateFromRef(ctx, sense.ID, refTranslation.ID, "test-source")
	if err != nil {
		t.Fatalf("CreateFromRef: unexpected error: %v", err)
	}

	// COALESCE should resolve ref text since user text is NULL.
	if got.Text == nil || *got.Text != refTranslation.Text {
		t.Errorf("Text mismatch: got %v, want %q", got.Text, refTranslation.Text)
	}
	if got.RefTranslationID == nil || *got.RefTranslationID != refTranslation.ID {
		t.Errorf("RefTranslationID mismatch: got %v, want %s", got.RefTranslationID, refTranslation.ID)
	}
	if got.SenseID != sense.ID {
		t.Errorf("SenseID mismatch: got %s, want %s", got.SenseID, sense.ID)
	}
	if got.SourceSlug != "test-source" {
		t.Errorf("SourceSlug mismatch: got %q, want %q", got.SourceSlug, "test-source")
	}

	// Verify it appears in GetBySenseID.
	all, err := repo.GetBySenseID(ctx, sense.ID)
	if err != nil {
		t.Fatalf("GetBySenseID: %v", err)
	}

	found := false
	for _, tr := range all {
		if tr.ID == got.ID {
			found = true
			if tr.Text == nil || *tr.Text != refTranslation.Text {
				t.Errorf("GetBySenseID: text mismatch for %s", tr.ID)
			}
			break
		}
	}
	if !found {
		t.Errorf("created translation %s not found in GetBySenseID result", got.ID)
	}
}

// ---------------------------------------------------------------------------
// CreateCustom + GetBySenseID tests
// ---------------------------------------------------------------------------

func TestRepo_CreateCustom_AndGetBySenseID(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	entry := testhelper.SeedEntryCustom(t, pool, user.ID)

	sense := entry.Senses[0]
	customText := "my custom translation"

	got, err := repo.CreateCustom(ctx, sense.ID, customText, "user")
	if err != nil {
		t.Fatalf("CreateCustom: unexpected error: %v", err)
	}

	if got.Text == nil || *got.Text != customText {
		t.Errorf("Text mismatch: got %v, want %q", got.Text, customText)
	}
	if got.RefTranslationID != nil {
		t.Errorf("expected RefTranslationID to be nil for custom translation, got %v", got.RefTranslationID)
	}
	if got.SourceSlug != "user" {
		t.Errorf("SourceSlug mismatch: got %q, want %q", got.SourceSlug, "user")
	}

	// Verify it appears in GetBySenseID.
	all, err := repo.GetBySenseID(ctx, sense.ID)
	if err != nil {
		t.Fatalf("GetBySenseID: %v", err)
	}

	found := false
	for _, tr := range all {
		if tr.ID == got.ID {
			found = true
			if tr.Text == nil || *tr.Text != customText {
				t.Errorf("GetBySenseID: text mismatch for %s", tr.ID)
			}
			break
		}
	}
	if !found {
		t.Errorf("created translation %s not found in GetBySenseID result", got.ID)
	}
}

// ---------------------------------------------------------------------------
// GetByID tests
// ---------------------------------------------------------------------------

func TestRepo_GetByID(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "tr-getbyid-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	// SeedEntry creates translations linked to ref_translations.
	translationID := entry.Senses[0].Translations[0].ID

	got, err := repo.GetByID(ctx, translationID)
	if err != nil {
		t.Fatalf("GetByID: unexpected error: %v", err)
	}

	if got.ID != translationID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, translationID)
	}
	// COALESCE should resolve ref text.
	if got.Text == nil {
		t.Error("expected Text to be resolved from ref, got nil")
	}
}

func TestRepo_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	assertIsDomainError(t, err, domain.ErrNotFound)
}

// ---------------------------------------------------------------------------
// Update tests
// ---------------------------------------------------------------------------

func TestRepo_Update(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "tr-update-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	translationID := entry.Senses[0].Translations[0].ID
	newText := "updated translation text"

	got, err := repo.Update(ctx, translationID, &newText)
	if err != nil {
		t.Fatalf("Update: unexpected error: %v", err)
	}

	// COALESCE should use user value since it's non-NULL.
	if got.Text == nil || *got.Text != newText {
		t.Errorf("Text mismatch: got %v, want %q", got.Text, newText)
	}
}

func TestRepo_Update_PreservesRefLink(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "tr-updref-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	translationID := entry.Senses[0].Translations[0].ID
	origRefTranslationID := entry.Senses[0].Translations[0].RefTranslationID

	// Update text only.
	newText := "user override translation"
	got, err := repo.Update(ctx, translationID, &newText)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	// ref_translation_id should NOT be touched.
	if got.RefTranslationID == nil || *got.RefTranslationID != *origRefTranslationID {
		t.Errorf("RefTranslationID changed: got %v, want %v", got.RefTranslationID, origRefTranslationID)
	}

	// Text should be the user override.
	if got.Text == nil || *got.Text != newText {
		t.Errorf("Text mismatch: got %v, want %q", got.Text, newText)
	}
}

func TestRepo_Update_SetToNil(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "tr-updnil-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	translationID := entry.Senses[0].Translations[0].ID
	refTranslation := refEntry.Senses[0].Translations[0]

	// Set text to nil -- should fallback to ref via COALESCE.
	got, err := repo.Update(ctx, translationID, nil)
	if err != nil {
		t.Fatalf("Update to nil: unexpected error: %v", err)
	}

	// COALESCE should resolve to ref text.
	if got.Text == nil || *got.Text != refTranslation.Text {
		t.Errorf("Text after nil update: got %v, want %q (from ref)", got.Text, refTranslation.Text)
	}
}

func TestRepo_Update_NotFound(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	text := "test"
	_, err := repo.Update(ctx, uuid.New(), &text)
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
	entry := testhelper.SeedEntryCustom(t, pool, user.ID)

	translationID := entry.Senses[0].Translations[0].ID

	if err := repo.Delete(ctx, translationID); err != nil {
		t.Fatalf("Delete: unexpected error: %v", err)
	}

	// Should not be found anymore.
	_, err := repo.GetByID(ctx, translationID)
	assertIsDomainError(t, err, domain.ErrNotFound)
}

func TestRepo_Delete_NotFound(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	err := repo.Delete(ctx, uuid.New())
	assertIsDomainError(t, err, domain.ErrNotFound)
}

// ---------------------------------------------------------------------------
// Reorder tests
// ---------------------------------------------------------------------------

func TestRepo_Reorder(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "tr-reorder-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	sense := entry.Senses[0]
	// SeedEntry creates 2 translations per sense at positions 0, 1.
	tr0 := sense.Translations[0]
	tr1 := sense.Translations[1]

	// Swap positions.
	items := []translation.ReorderItem{
		{ID: tr0.ID, Position: 1},
		{ID: tr1.ID, Position: 0},
	}

	if err := repo.Reorder(ctx, items); err != nil {
		t.Fatalf("Reorder: unexpected error: %v", err)
	}

	// Verify positions swapped.
	translations, err := repo.GetBySenseID(ctx, sense.ID)
	if err != nil {
		t.Fatalf("GetBySenseID after reorder: %v", err)
	}

	if len(translations) < 2 {
		t.Fatalf("expected at least 2 translations, got %d", len(translations))
	}

	// Translations should be ordered by position (0, 1).
	if translations[0].ID != tr1.ID {
		t.Errorf("expected translation at position 0 to be %s, got %s", tr1.ID, translations[0].ID)
	}
	if translations[1].ID != tr0.ID {
		t.Errorf("expected translation at position 1 to be %s, got %s", tr0.ID, translations[1].ID)
	}
}

func TestRepo_Reorder_EmptyItems(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	if err := repo.Reorder(ctx, nil); err != nil {
		t.Fatalf("Reorder empty: unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// PositionAutoIncrement tests
// ---------------------------------------------------------------------------

func TestRepo_PositionAutoIncrement(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	entry := testhelper.SeedEntryCustom(t, pool, user.ID)

	sense := entry.Senses[0]
	// SeedEntryCustom creates 1 custom translation at position 0.

	tr1, err := repo.CreateCustom(ctx, sense.ID, "second translation", "user")
	if err != nil {
		t.Fatalf("CreateCustom[1]: %v", err)
	}

	tr2, err := repo.CreateCustom(ctx, sense.ID, "third translation", "user")
	if err != nil {
		t.Fatalf("CreateCustom[2]: %v", err)
	}

	// Position 0 is the seed translation, so new ones should be 1, 2.
	if tr1.Position != 1 {
		t.Errorf("expected tr1 position 1, got %d", tr1.Position)
	}
	if tr2.Position != 2 {
		t.Errorf("expected tr2 position 2, got %d", tr2.Position)
	}
}

// ---------------------------------------------------------------------------
// GetBySenseIDs batch tests
// ---------------------------------------------------------------------------

func TestRepo_GetBySenseIDs_Batch(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "tr-batch-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	sense0 := entry.Senses[0]
	sense1 := entry.Senses[1]

	got, err := repo.GetBySenseIDs(ctx, []uuid.UUID{sense0.ID, sense1.ID})
	if err != nil {
		t.Fatalf("GetBySenseIDs: unexpected error: %v", err)
	}

	// Each sense has 2 translations from SeedEntry.
	if len(got) != 4 {
		t.Fatalf("expected 4 translations total, got %d", len(got))
	}

	// Verify results can be grouped by sense_id.
	bySense := make(map[uuid.UUID][]domain.Translation)
	for _, tr := range got {
		bySense[tr.SenseID] = append(bySense[tr.SenseID], tr.Translation)
	}

	if len(bySense[sense0.ID]) != 2 {
		t.Errorf("expected 2 translations for sense0, got %d", len(bySense[sense0.ID]))
	}
	if len(bySense[sense1.ID]) != 2 {
		t.Errorf("expected 2 translations for sense1, got %d", len(bySense[sense1.ID]))
	}

	// Verify COALESCE resolved ref text.
	for _, tr := range got {
		if tr.Text == nil {
			t.Errorf("expected translation %s text to be resolved from ref, got nil", tr.ID)
		}
	}
}

func TestRepo_GetBySenseIDs_Empty(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	got, err := repo.GetBySenseIDs(ctx, []uuid.UUID{})
	if err != nil {
		t.Fatalf("GetBySenseIDs empty: unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 translations, got %d", len(got))
	}
}

// ---------------------------------------------------------------------------
// CountBySense tests
// ---------------------------------------------------------------------------

func TestRepo_CountBySense(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "tr-count-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	sense := entry.Senses[0]

	count, err := repo.CountBySense(ctx, sense.ID)
	if err != nil {
		t.Fatalf("CountBySense: %v", err)
	}

	// SeedEntry creates 2 translations per sense.
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestRepo_CountBySense_Zero(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	count, err := repo.CountBySense(ctx, uuid.New())
	if err != nil {
		t.Fatalf("CountBySense: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
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
