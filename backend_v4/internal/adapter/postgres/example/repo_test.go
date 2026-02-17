package example_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	postgres "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/example"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/testhelper"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// newRepo sets up a test DB and returns a ready Repo + pool.
func newRepo(t *testing.T) (*example.Repo, *pgxpool.Pool) {
	t.Helper()
	pool := testhelper.SetupTestDB(t)
	txm := postgres.NewTxManager(pool)
	return example.New(pool, txm), pool
}

// ---------------------------------------------------------------------------
// CreateFromRef + GetBySenseID tests
// ---------------------------------------------------------------------------

func TestRepo_CreateFromRef_AndGetBySenseID(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "exfromref-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	sense := entry.Senses[0]
	refExample := refEntry.Senses[0].Examples[0]

	got, err := repo.CreateFromRef(ctx, sense.ID, refExample.ID, "test-source")
	if err != nil {
		t.Fatalf("CreateFromRef: unexpected error: %v", err)
	}

	// COALESCE should resolve ref values since user fields are NULL.
	if got.Sentence == nil || *got.Sentence != refExample.Sentence {
		t.Errorf("Sentence mismatch: got %v, want %q", got.Sentence, refExample.Sentence)
	}
	if got.Translation == nil || *got.Translation != *refExample.Translation {
		t.Errorf("Translation mismatch: got %v, want %v", got.Translation, *refExample.Translation)
	}
	if got.RefExampleID == nil || *got.RefExampleID != refExample.ID {
		t.Errorf("RefExampleID mismatch: got %v, want %s", got.RefExampleID, refExample.ID)
	}
	if got.SenseID != sense.ID {
		t.Errorf("SenseID mismatch: got %s, want %s", got.SenseID, sense.ID)
	}
	if got.SourceSlug != "test-source" {
		t.Errorf("SourceSlug mismatch: got %q, want %q", got.SourceSlug, "test-source")
	}

	// Verify via GetBySenseID.
	all, err := repo.GetBySenseID(ctx, sense.ID)
	if err != nil {
		t.Fatalf("GetBySenseID: unexpected error: %v", err)
	}

	// SeedEntry creates 2 examples per sense, plus the one we just created.
	found := false
	for _, ex := range all {
		if ex.ID == got.ID {
			found = true
			if ex.Sentence == nil || *ex.Sentence != refExample.Sentence {
				t.Errorf("GetBySenseID: Sentence mismatch: got %v, want %q", ex.Sentence, refExample.Sentence)
			}
			break
		}
	}
	if !found {
		t.Errorf("created example %s not found in GetBySenseID results", got.ID)
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
	sentence := "The cat sat on the mat."
	translation := "Кот сидел на коврике."

	got, err := repo.CreateCustom(ctx, sense.ID, sentence, &translation, "user")
	if err != nil {
		t.Fatalf("CreateCustom: unexpected error: %v", err)
	}

	if got.Sentence == nil || *got.Sentence != sentence {
		t.Errorf("Sentence mismatch: got %v, want %q", got.Sentence, sentence)
	}
	if got.Translation == nil || *got.Translation != translation {
		t.Errorf("Translation mismatch: got %v, want %q", got.Translation, translation)
	}
	if got.RefExampleID != nil {
		t.Errorf("expected RefExampleID to be nil for custom example, got %v", got.RefExampleID)
	}
	if got.SourceSlug != "user" {
		t.Errorf("SourceSlug mismatch: got %q, want %q", got.SourceSlug, "user")
	}

	// Verify via GetBySenseID.
	all, err := repo.GetBySenseID(ctx, sense.ID)
	if err != nil {
		t.Fatalf("GetBySenseID: unexpected error: %v", err)
	}

	found := false
	for _, ex := range all {
		if ex.ID == got.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("created example %s not found in GetBySenseID results", got.ID)
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
	refEntry := testhelper.SeedRefEntry(t, pool, "getbyid-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	// SeedEntry creates examples linked to ref_examples.
	exampleID := entry.Senses[0].Examples[0].ID

	got, err := repo.GetByID(ctx, exampleID)
	if err != nil {
		t.Fatalf("GetByID: unexpected error: %v", err)
	}

	if got.ID != exampleID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, exampleID)
	}
	// COALESCE should resolve ref values.
	if got.Sentence == nil {
		t.Error("expected Sentence to be resolved from ref, got nil")
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
	entry := testhelper.SeedEntryCustom(t, pool, user.ID)

	exampleID := entry.Senses[0].Examples[0].ID
	newSentence := "updated sentence"
	newTranslation := "updated translation"

	got, err := repo.Update(ctx, exampleID, newSentence, &newTranslation)
	if err != nil {
		t.Fatalf("Update: unexpected error: %v", err)
	}

	if got.Sentence == nil || *got.Sentence != newSentence {
		t.Errorf("Sentence mismatch: got %v, want %q", got.Sentence, newSentence)
	}
	if got.Translation == nil || *got.Translation != newTranslation {
		t.Errorf("Translation mismatch: got %v, want %q", got.Translation, newTranslation)
	}
}

func TestRepo_Update_PreservesRefLink(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "update-ref-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	exampleID := entry.Senses[0].Examples[0].ID
	origRefExampleID := entry.Senses[0].Examples[0].RefExampleID

	// Update sentence only.
	newSentence := "user override sentence"
	got, err := repo.Update(ctx, exampleID, newSentence, nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	// ref_example_id should NOT be touched.
	if got.RefExampleID == nil || *got.RefExampleID != *origRefExampleID {
		t.Errorf("RefExampleID changed: got %v, want %v", got.RefExampleID, origRefExampleID)
	}

	// Sentence should be the user override.
	if got.Sentence == nil || *got.Sentence != newSentence {
		t.Errorf("Sentence mismatch: got %v, want %q", got.Sentence, newSentence)
	}
}

func TestRepo_Update_NotFound(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	sentence := "test"
	_, err := repo.Update(ctx, uuid.New(), sentence, nil)
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

	exampleID := entry.Senses[0].Examples[0].ID

	if err := repo.Delete(ctx, exampleID); err != nil {
		t.Fatalf("Delete: unexpected error: %v", err)
	}

	// Should not be found anymore.
	_, err := repo.GetByID(ctx, exampleID)
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
	refEntry := testhelper.SeedRefEntry(t, pool, "reorder-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	sense := entry.Senses[0]
	// SeedEntry creates 2 examples per sense at positions 0, 1. Swap them.
	items := []domain.ReorderItem{
		{ID: sense.Examples[0].ID, Position: 10},
		{ID: sense.Examples[1].ID, Position: 20},
	}

	if err := repo.Reorder(ctx, items); err != nil {
		t.Fatalf("Reorder: unexpected error: %v", err)
	}

	// Verify positions updated.
	examples, err := repo.GetBySenseID(ctx, sense.ID)
	if err != nil {
		t.Fatalf("GetBySenseID after reorder: %v", err)
	}

	if len(examples) != 2 {
		t.Fatalf("expected 2 examples, got %d", len(examples))
	}

	if examples[0].Position != 10 {
		t.Errorf("expected position 10, got %d", examples[0].Position)
	}
	if examples[1].Position != 20 {
		t.Errorf("expected position 20, got %d", examples[1].Position)
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
// Position auto-increment tests
// ---------------------------------------------------------------------------

func TestRepo_PositionAutoIncrement(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user := testhelper.SeedUser(t, pool)
	refEntry := testhelper.SeedRefEntry(t, pool, "autopos-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	sense := entry.Senses[0]
	// SeedEntry creates 2 examples at positions 0, 1.

	sentence := "third example"
	e3, err := repo.CreateCustom(ctx, sense.ID, sentence, nil, "user")
	if err != nil {
		t.Fatalf("CreateCustom[2]: %v", err)
	}

	if e3.Position != 2 {
		t.Errorf("expected position 2, got %d", e3.Position)
	}

	sentence2 := "fourth example"
	e4, err := repo.CreateCustom(ctx, sense.ID, sentence2, nil, "user")
	if err != nil {
		t.Fatalf("CreateCustom[3]: %v", err)
	}

	if e4.Position != 3 {
		t.Errorf("expected position 3, got %d", e4.Position)
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
	refEntry := testhelper.SeedRefEntry(t, pool, "batch-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	senseIDs := []uuid.UUID{entry.Senses[0].ID, entry.Senses[1].ID}

	got, err := repo.GetBySenseIDs(ctx, senseIDs)
	if err != nil {
		t.Fatalf("GetBySenseIDs: unexpected error: %v", err)
	}

	// Each sense has 2 examples from SeedEntry.
	if len(got) != 4 {
		t.Fatalf("expected 4 examples total, got %d", len(got))
	}

	// Verify results can be grouped by sense_id.
	bySense := make(map[uuid.UUID][]domain.Example)
	for _, ex := range got {
		bySense[ex.SenseID] = append(bySense[ex.SenseID], ex)
	}

	if len(bySense[senseIDs[0]]) != 2 {
		t.Errorf("expected 2 examples for sense[0], got %d", len(bySense[senseIDs[0]]))
	}
	if len(bySense[senseIDs[1]]) != 2 {
		t.Errorf("expected 2 examples for sense[1], got %d", len(bySense[senseIDs[1]]))
	}

	// Verify COALESCE resolved ref sentence.
	if got[0].Sentence == nil {
		t.Error("expected sentence to be resolved from ref, got nil")
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
		t.Errorf("expected 0 examples, got %d", len(got))
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
	refEntry := testhelper.SeedRefEntry(t, pool, "count-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, pool, user.ID, refEntry.ID)

	sense := entry.Senses[0]

	count, err := repo.CountBySense(ctx, sense.ID)
	if err != nil {
		t.Fatalf("CountBySense: %v", err)
	}

	// SeedEntry creates 2 examples per sense.
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
