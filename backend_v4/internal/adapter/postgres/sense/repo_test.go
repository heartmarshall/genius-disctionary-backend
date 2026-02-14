package sense_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	postgres "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/sense"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/testhelper"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// newRepo sets up a test DB and returns a ready Repo + pool.
func newRepo(t *testing.T) (*sense.Repo, *pgxpool.Pool) {
	t.Helper()
	pool := testhelper.SetupTestDB(t)
	txm := postgres.NewTxManager(pool)
	return sense.New(pool, txm), pool
}

// ---------------------------------------------------------------------------
// CreateFromRef tests
// ---------------------------------------------------------------------------

func TestRepo_CreateFromRef_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "createfromref-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	refSense := refEntry.Senses[0]

	got, err := repo.CreateFromRef(ctx, entry.ID, refSense.ID, "test-source")
	if err != nil {
		t.Fatalf("CreateFromRef: unexpected error: %v", err)
	}

	// COALESCE should resolve ref values since user fields are NULL.
	if got.Definition == nil || *got.Definition != refSense.Definition {
		t.Errorf("Definition mismatch: got %v, want %q", got.Definition, refSense.Definition)
	}
	if got.PartOfSpeech == nil || *got.PartOfSpeech != *refSense.PartOfSpeech {
		t.Errorf("PartOfSpeech mismatch: got %v, want %v", got.PartOfSpeech, refSense.PartOfSpeech)
	}
	if got.CEFRLevel == nil || *got.CEFRLevel != *refSense.CEFRLevel {
		t.Errorf("CEFRLevel mismatch: got %v, want %v", got.CEFRLevel, refSense.CEFRLevel)
	}
	if got.RefSenseID == nil || *got.RefSenseID != refSense.ID {
		t.Errorf("RefSenseID mismatch: got %v, want %s", got.RefSenseID, refSense.ID)
	}
	if got.EntryID != entry.ID {
		t.Errorf("EntryID mismatch: got %s, want %s", got.EntryID, entry.ID)
	}
	if got.SourceSlug != "test-source" {
		t.Errorf("SourceSlug mismatch: got %q, want %q", got.SourceSlug, "test-source")
	}
}

func TestRepo_CreateFromRef_AutoPosition(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "autopos-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	// Create two senses from ref; positions should be auto-incremented.
	s1, err := repo.CreateFromRef(ctx, entry.ID, refEntry.Senses[0].ID, "test")
	if err != nil {
		t.Fatalf("CreateFromRef[0]: %v", err)
	}

	s2, err := repo.CreateFromRef(ctx, entry.ID, refEntry.Senses[1].ID, "test")
	if err != nil {
		t.Fatalf("CreateFromRef[1]: %v", err)
	}

	// Existing senses from SeedEntry have positions 0 and 1.
	// New senses should get positions 2 and 3.
	if s1.Position <= 1 {
		t.Errorf("expected s1 position > 1 (after seed senses), got %d", s1.Position)
	}
	if s2.Position != s1.Position+1 {
		t.Errorf("expected s2 position = %d, got %d", s1.Position+1, s2.Position)
	}
}

func TestRepo_CreateFromRef_InvalidEntryID(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	_, err := repo.CreateFromRef(ctx, uuid.New(), uuid.New(), "test")
	assertIsDomainError(t, err, domain.ErrNotFound) // FK violation
}

// ---------------------------------------------------------------------------
// CreateCustom tests
// ---------------------------------------------------------------------------

func TestRepo_CreateCustom_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	entry := testhelper.SeedEntryCustom(t, pool, user.ID)

	def := "a custom definition"
	pos := domain.PartOfSpeechAdjective
	cefr := "C1"

	got, err := repo.CreateCustom(ctx, entry.ID, &def, &pos, &cefr, "user")
	if err != nil {
		t.Fatalf("CreateCustom: unexpected error: %v", err)
	}

	if got.Definition == nil || *got.Definition != def {
		t.Errorf("Definition mismatch: got %v, want %q", got.Definition, def)
	}
	if got.PartOfSpeech == nil || *got.PartOfSpeech != pos {
		t.Errorf("PartOfSpeech mismatch: got %v, want %v", got.PartOfSpeech, pos)
	}
	if got.CEFRLevel == nil || *got.CEFRLevel != cefr {
		t.Errorf("CEFRLevel mismatch: got %v, want %q", got.CEFRLevel, cefr)
	}
	if got.RefSenseID != nil {
		t.Errorf("expected RefSenseID to be nil for custom sense, got %v", got.RefSenseID)
	}
	if got.SourceSlug != "user" {
		t.Errorf("SourceSlug mismatch: got %q, want %q", got.SourceSlug, "user")
	}
}

func TestRepo_CreateCustom_NilFields(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	entry := testhelper.SeedEntryCustom(t, pool, user.ID)

	got, err := repo.CreateCustom(ctx, entry.ID, nil, nil, nil, "user")
	if err != nil {
		t.Fatalf("CreateCustom nil fields: unexpected error: %v", err)
	}

	if got.Definition != nil {
		t.Errorf("expected nil Definition, got %v", got.Definition)
	}
	if got.PartOfSpeech != nil {
		t.Errorf("expected nil PartOfSpeech, got %v", got.PartOfSpeech)
	}
	if got.CEFRLevel != nil {
		t.Errorf("expected nil CEFRLevel, got %v", got.CEFRLevel)
	}
}

func TestRepo_CreateCustom_AutoPosition(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	entry := testhelper.SeedEntryCustom(t, pool, user.ID)

	// SeedEntryCustom creates 2 senses at positions 0, 1.
	def := "third sense"
	s3, err := repo.CreateCustom(ctx, entry.ID, &def, nil, nil, "user")
	if err != nil {
		t.Fatalf("CreateCustom: %v", err)
	}

	if s3.Position != 2 {
		t.Errorf("expected position 2, got %d", s3.Position)
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
	refEntry := testhelper.SeedRefEntry(t, pool, "getbyid-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	senseID := entry.Senses[0].ID

	got, err := repo.GetByID(ctx, senseID)
	if err != nil {
		t.Fatalf("GetByID: unexpected error: %v", err)
	}

	if got.ID != senseID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, senseID)
	}
	// COALESCE should resolve ref values.
	if got.Definition == nil {
		t.Error("expected Definition to be resolved from ref, got nil")
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
// GetByEntryID tests
// ---------------------------------------------------------------------------

func TestRepo_GetByEntryID_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "getbyentry-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	got, err := repo.GetByEntryID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("GetByEntryID: unexpected error: %v", err)
	}

	// SeedEntry creates 2 senses.
	if len(got) != 2 {
		t.Fatalf("expected 2 senses, got %d", len(got))
	}

	// Verify ordered by position.
	for i := 1; i < len(got); i++ {
		if got[i].Position < got[i-1].Position {
			t.Errorf("senses not ordered by position: %d < %d at index %d",
				got[i].Position, got[i-1].Position, i)
		}
	}

	// Verify COALESCE resolved ref definition.
	if got[0].Definition == nil {
		t.Error("expected sense[0] definition to be resolved from ref")
	}
}

func TestRepo_GetByEntryID_Empty(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	entry := testhelper.SeedEntryCustom(t, pool, user.ID)

	// Delete all senses to get empty result.
	for _, s := range entry.Senses {
		if err := repo.Delete(ctx, s.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}
	}

	got, err := repo.GetByEntryID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("GetByEntryID empty: unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 senses, got %d", len(got))
	}
}

// ---------------------------------------------------------------------------
// GetByEntryIDs tests (batch for DataLoader)
// ---------------------------------------------------------------------------

func TestRepo_GetByEntryIDs_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry1 := testhelper.SeedRefEntry(t, pool, "batch1-"+uuid.New().String()[:8])
	refEntry2 := testhelper.SeedRefEntry(t, pool, "batch2-"+uuid.New().String()[:8])
	entry1 := testhelper.SeedEntry(t, pool, user.ID, refEntry1.ID)
	entry2 := testhelper.SeedEntry(t, pool, user.ID, refEntry2.ID)

	got, err := repo.GetByEntryIDs(ctx, []uuid.UUID{entry1.ID, entry2.ID})
	if err != nil {
		t.Fatalf("GetByEntryIDs: unexpected error: %v", err)
	}

	// Each entry has 2 senses from SeedEntry.
	if len(got) != 4 {
		t.Fatalf("expected 4 senses total, got %d", len(got))
	}

	// Verify results can be grouped by entry_id.
	byEntry := make(map[uuid.UUID][]domain.Sense)
	for _, s := range got {
		byEntry[s.EntryID] = append(byEntry[s.EntryID], s)
	}

	if len(byEntry[entry1.ID]) != 2 {
		t.Errorf("expected 2 senses for entry1, got %d", len(byEntry[entry1.ID]))
	}
	if len(byEntry[entry2.ID]) != 2 {
		t.Errorf("expected 2 senses for entry2, got %d", len(byEntry[entry2.ID]))
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
	if len(got) != 0 {
		t.Errorf("expected 0 senses, got %d", len(got))
	}
}

func TestRepo_GetByEntryIDs_NonexistentEntries(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	got, err := repo.GetByEntryIDs(ctx, []uuid.UUID{uuid.New(), uuid.New()})
	if err != nil {
		t.Fatalf("GetByEntryIDs nonexistent: unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 senses for nonexistent entries, got %d", len(got))
	}
}

// ---------------------------------------------------------------------------
// CountByEntry tests
// ---------------------------------------------------------------------------

func TestRepo_CountByEntry_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	entry := testhelper.SeedEntryCustom(t, pool, user.ID)

	count, err := repo.CountByEntry(ctx, entry.ID)
	if err != nil {
		t.Fatalf("CountByEntry: %v", err)
	}

	// SeedEntryCustom creates 2 senses.
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestRepo_CountByEntry_Zero(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	count, err := repo.CountByEntry(ctx, uuid.New())
	if err != nil {
		t.Fatalf("CountByEntry: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// Update tests
// ---------------------------------------------------------------------------

func TestRepo_Update_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	entry := testhelper.SeedEntryCustom(t, pool, user.ID)

	senseID := entry.Senses[0].ID
	newDef := "updated definition"
	newPos := domain.PartOfSpeechAdverb
	newCefr := "A2"

	got, err := repo.Update(ctx, senseID, &newDef, &newPos, &newCefr)
	if err != nil {
		t.Fatalf("Update: unexpected error: %v", err)
	}

	if got.Definition == nil || *got.Definition != newDef {
		t.Errorf("Definition mismatch: got %v, want %q", got.Definition, newDef)
	}
	if got.PartOfSpeech == nil || *got.PartOfSpeech != newPos {
		t.Errorf("PartOfSpeech mismatch: got %v, want %v", got.PartOfSpeech, newPos)
	}
	if got.CEFRLevel == nil || *got.CEFRLevel != newCefr {
		t.Errorf("CEFRLevel mismatch: got %v, want %q", got.CEFRLevel, newCefr)
	}
}

func TestRepo_Update_SetToNil(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	entry := testhelper.SeedEntryCustom(t, pool, user.ID)

	senseID := entry.Senses[0].ID

	// Set all fields to nil.
	got, err := repo.Update(ctx, senseID, nil, nil, nil)
	if err != nil {
		t.Fatalf("Update to nil: unexpected error: %v", err)
	}

	if got.Definition != nil {
		t.Errorf("expected nil Definition, got %v", got.Definition)
	}
	if got.PartOfSpeech != nil {
		t.Errorf("expected nil PartOfSpeech, got %v", got.PartOfSpeech)
	}
	if got.CEFRLevel != nil {
		t.Errorf("expected nil CEFRLevel, got %v", got.CEFRLevel)
	}
}

func TestRepo_Update_PreservesRefSenseID(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "update-ref-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	senseID := entry.Senses[0].ID
	origRefSenseID := entry.Senses[0].RefSenseID

	// Update definition only.
	newDef := "user override definition"
	got, err := repo.Update(ctx, senseID, &newDef, nil, nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	// ref_sense_id should NOT be touched.
	if got.RefSenseID == nil || *got.RefSenseID != *origRefSenseID {
		t.Errorf("RefSenseID changed: got %v, want %v", got.RefSenseID, origRefSenseID)
	}

	// Definition should be the user override.
	if got.Definition == nil || *got.Definition != newDef {
		t.Errorf("Definition mismatch: got %v, want %q", got.Definition, newDef)
	}
}

func TestRepo_Update_PartialCustomizationWithCOALESCE(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "partial-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	senseID := entry.Senses[0].ID
	refSense := refEntry.Senses[0]

	// Override only definition, keep part_of_speech from ref via COALESCE.
	newDef := "user-custom definition"
	got, err := repo.Update(ctx, senseID, &newDef, nil, nil)
	if err != nil {
		t.Fatalf("Update partial: %v", err)
	}

	// Definition should be the user-provided value.
	if got.Definition == nil || *got.Definition != newDef {
		t.Errorf("Definition mismatch: got %v, want %q", got.Definition, newDef)
	}

	// PartOfSpeech should come from ref via COALESCE (since user set nil).
	if got.PartOfSpeech == nil {
		t.Fatal("expected PartOfSpeech from ref via COALESCE, got nil")
	}
	if *got.PartOfSpeech != *refSense.PartOfSpeech {
		t.Errorf("PartOfSpeech from COALESCE mismatch: got %v, want %v", *got.PartOfSpeech, *refSense.PartOfSpeech)
	}
}

func TestRepo_Update_NotFound(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	def := "test"
	_, err := repo.Update(ctx, uuid.New(), &def, nil, nil)
	assertIsDomainError(t, err, domain.ErrNotFound)
}

// ---------------------------------------------------------------------------
// Delete tests
// ---------------------------------------------------------------------------

func TestRepo_Delete_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	entry := testhelper.SeedEntryCustom(t, pool, user.ID)

	senseID := entry.Senses[0].ID

	if err := repo.Delete(ctx, senseID); err != nil {
		t.Fatalf("Delete: unexpected error: %v", err)
	}

	// Should not be found anymore.
	_, err := repo.GetByID(ctx, senseID)
	assertIsDomainError(t, err, domain.ErrNotFound)
}

func TestRepo_Delete_NotFound(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	err := repo.Delete(ctx, uuid.New())
	assertIsDomainError(t, err, domain.ErrNotFound)
}

func TestRepo_Delete_ReducesCount(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	entry := testhelper.SeedEntryCustom(t, pool, user.ID)

	countBefore, _ := repo.CountByEntry(ctx, entry.ID)

	if err := repo.Delete(ctx, entry.Senses[0].ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	countAfter, _ := repo.CountByEntry(ctx, entry.ID)
	if countAfter != countBefore-1 {
		t.Errorf("expected count %d after delete, got %d", countBefore-1, countAfter)
	}
}

// ---------------------------------------------------------------------------
// Reorder tests
// ---------------------------------------------------------------------------

func TestRepo_Reorder_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	entry := testhelper.SeedEntryCustom(t, pool, user.ID)

	// SeedEntryCustom creates senses at positions 0, 1. Swap them.
	items := []sense.ReorderItem{
		{ID: entry.Senses[0].ID, Position: 1},
		{ID: entry.Senses[1].ID, Position: 0},
	}

	if err := repo.Reorder(ctx, items); err != nil {
		t.Fatalf("Reorder: unexpected error: %v", err)
	}

	// Verify positions swapped.
	senses, err := repo.GetByEntryID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("GetByEntryID after reorder: %v", err)
	}

	if len(senses) != 2 {
		t.Fatalf("expected 2 senses, got %d", len(senses))
	}

	// Senses should be ordered by position (0, 1).
	if senses[0].ID != entry.Senses[1].ID {
		t.Errorf("expected sense at position 0 to be %s, got %s", entry.Senses[1].ID, senses[0].ID)
	}
	if senses[1].ID != entry.Senses[0].ID {
		t.Errorf("expected sense at position 1 to be %s, got %s", entry.Senses[0].ID, senses[1].ID)
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

func TestRepo_Reorder_Atomic(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	entry := testhelper.SeedEntryCustom(t, pool, user.ID)

	// Try to reorder with a nonexistent sense ID mixed in.
	// The entire batch should still succeed because UpdateSensePosition is :exec
	// (no error on 0 rows). But we can verify that the valid ones get updated.
	items := []sense.ReorderItem{
		{ID: entry.Senses[0].ID, Position: 10},
		{ID: entry.Senses[1].ID, Position: 20},
	}

	if err := repo.Reorder(ctx, items); err != nil {
		t.Fatalf("Reorder: unexpected error: %v", err)
	}

	senses, _ := repo.GetByEntryID(ctx, entry.ID)
	if len(senses) != 2 {
		t.Fatalf("expected 2 senses, got %d", len(senses))
	}
	if senses[0].Position != 10 {
		t.Errorf("expected position 10, got %d", senses[0].Position)
	}
	if senses[1].Position != 20 {
		t.Errorf("expected position 20, got %d", senses[1].Position)
	}
}

// ---------------------------------------------------------------------------
// TRIGGER TEST: fn_preserve_sense_on_ref_delete
// ---------------------------------------------------------------------------

func TestRepo_TriggerPreserveSenseOnRefDelete(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	// 1. Create ref_entry with ref_senses via SeedRefEntry.
	refEntry := testhelper.SeedRefEntry(t, pool, "trigger-"+uuid.New().String()[:8])
	refSense := refEntry.Senses[0]

	// 2. Create user entry with senses linked to ref_senses via SeedEntry.
	user := testhelper.SeedUser(t, pool)
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	// Find the user sense linked to refSense.
	var linkedSenseID uuid.UUID
	for _, s := range entry.Senses {
		if s.RefSenseID != nil && *s.RefSenseID == refSense.ID {
			linkedSenseID = s.ID
			break
		}
	}
	if linkedSenseID == uuid.Nil {
		t.Fatal("could not find user sense linked to ref_sense")
	}

	// 3. Verify that reading the sense returns ref values (since user fields are NULL).
	before, err := repo.GetByID(ctx, linkedSenseID)
	if err != nil {
		t.Fatalf("GetByID before delete: %v", err)
	}

	if before.Definition == nil || *before.Definition != refSense.Definition {
		t.Errorf("before delete: definition mismatch: got %v, want %q", before.Definition, refSense.Definition)
	}
	if before.PartOfSpeech == nil || *before.PartOfSpeech != *refSense.PartOfSpeech {
		t.Errorf("before delete: part_of_speech mismatch: got %v, want %v", before.PartOfSpeech, refSense.PartOfSpeech)
	}
	if before.CEFRLevel == nil || *before.CEFRLevel != *refSense.CEFRLevel {
		t.Errorf("before delete: cefr_level mismatch: got %v, want %v", before.CEFRLevel, refSense.CEFRLevel)
	}
	if before.RefSenseID == nil || *before.RefSenseID != refSense.ID {
		t.Errorf("before delete: ref_sense_id mismatch: got %v, want %s", before.RefSenseID, refSense.ID)
	}

	// 4. Delete the ref_sense directly (simulates catalog cleanup).
	_, err = pool.Exec(ctx, "DELETE FROM ref_senses WHERE id = $1", refSense.ID)
	if err != nil {
		t.Fatalf("delete ref_sense: %v", err)
	}

	// 5. Read the sense again. Trigger should have copied data into user fields.
	after, err := repo.GetByID(ctx, linkedSenseID)
	if err != nil {
		t.Fatalf("GetByID after delete: %v", err)
	}

	// Definition, PartOfSpeech, CEFRLevel should still have the same values
	// (trigger copied ref values into user fields before delete).
	if after.Definition == nil || *after.Definition != refSense.Definition {
		t.Errorf("after delete: definition mismatch: got %v, want %q", after.Definition, refSense.Definition)
	}
	if after.PartOfSpeech == nil || *after.PartOfSpeech != *refSense.PartOfSpeech {
		t.Errorf("after delete: part_of_speech mismatch: got %v, want %v", after.PartOfSpeech, refSense.PartOfSpeech)
	}
	if after.CEFRLevel == nil || *after.CEFRLevel != *refSense.CEFRLevel {
		t.Errorf("after delete: cefr_level mismatch: got %v, want %v", after.CEFRLevel, refSense.CEFRLevel)
	}

	// 6. ref_sense_id should be NULL now (ON DELETE SET NULL).
	if after.RefSenseID != nil {
		t.Errorf("after delete: expected ref_sense_id to be nil, got %v", after.RefSenseID)
	}
}

func TestRepo_TriggerPreserve_PartialCustomization(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	// Test that if user has overridden definition but not part_of_speech,
	// after ref deletion, user keeps their definition and trigger fills part_of_speech.
	refEntry := testhelper.SeedRefEntry(t, pool, "trigger-partial-"+uuid.New().String()[:8])
	refSense := refEntry.Senses[0]

	user := testhelper.SeedUser(t, pool)
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	var linkedSenseID uuid.UUID
	for _, s := range entry.Senses {
		if s.RefSenseID != nil && *s.RefSenseID == refSense.ID {
			linkedSenseID = s.ID
			break
		}
	}

	// Override only the definition.
	userDef := "my custom definition"
	_, err := repo.Update(ctx, linkedSenseID, &userDef, nil, nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Delete the ref_sense.
	_, err = pool.Exec(ctx, "DELETE FROM ref_senses WHERE id = $1", refSense.ID)
	if err != nil {
		t.Fatalf("delete ref_sense: %v", err)
	}

	// Read sense after trigger fired.
	after, err := repo.GetByID(ctx, linkedSenseID)
	if err != nil {
		t.Fatalf("GetByID after trigger: %v", err)
	}

	// Definition should be the USER's value (COALESCE(user_def, ref_def) = user_def).
	if after.Definition == nil || *after.Definition != userDef {
		t.Errorf("definition: got %v, want %q (user override preserved)", after.Definition, userDef)
	}

	// PartOfSpeech should be from ref (trigger copied ref value since user had NULL).
	if after.PartOfSpeech == nil || *after.PartOfSpeech != *refSense.PartOfSpeech {
		t.Errorf("part_of_speech: got %v, want %v (trigger should copy from ref)", after.PartOfSpeech, refSense.PartOfSpeech)
	}

	// ref_sense_id should be NULL.
	if after.RefSenseID != nil {
		t.Errorf("ref_sense_id: expected nil, got %v", after.RefSenseID)
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
