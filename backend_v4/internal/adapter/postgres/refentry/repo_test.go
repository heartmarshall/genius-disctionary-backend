package refentry_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	postgres "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/refentry"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/testhelper"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// newRepo sets up a test DB and returns a ready Repo + pool.
func newRepo(t *testing.T) (*refentry.Repo, *pgxpool.Pool) {
	t.Helper()
	pool := testhelper.SetupTestDB(t)
	txm := postgres.NewTxManager(pool)
	return refentry.New(pool, txm), pool
}

// buildRefEntry creates a fully populated domain.RefEntry for testing.
func buildRefEntry(text string) domain.RefEntry {
	now := time.Now().UTC().Truncate(time.Microsecond)
	entryID := uuid.New()

	posNoun := domain.PartOfSpeechNoun
	posVerb := domain.PartOfSpeechVerb
	cefrB1 := "B1"

	sense1ID := uuid.New()
	sense2ID := uuid.New()

	exTranslation := "Example translation"

	return domain.RefEntry{
		ID:             entryID,
		Text:           text,
		TextNormalized: domain.NormalizeText(text),
		CreatedAt:      now,
		Senses: []domain.RefSense{
			{
				ID:           sense1ID,
				RefEntryID:   entryID,
				Definition:   "a thing used to test",
				PartOfSpeech: &posNoun,
				CEFRLevel:    &cefrB1,
				SourceSlug:   "test-source",
				Position:     0,
				CreatedAt:    now,
				Translations: []domain.RefTranslation{
					{
						ID:         uuid.New(),
						RefSenseID: sense1ID,
						Text:       "translation-1",
						SourceSlug: "test-source",
						Position:   0,
					},
					{
						ID:         uuid.New(),
						RefSenseID: sense1ID,
						Text:       "translation-2",
						SourceSlug: "test-source",
						Position:   1,
					},
				},
				Examples: []domain.RefExample{
					{
						ID:          uuid.New(),
						RefSenseID:  sense1ID,
						Sentence:    "This is a test sentence.",
						Translation: &exTranslation,
						SourceSlug:  "test-source",
						Position:    0,
					},
				},
			},
			{
				ID:           sense2ID,
				RefEntryID:   entryID,
				Definition:   "to verify something works",
				PartOfSpeech: &posVerb,
				CEFRLevel:    nil,
				SourceSlug:   "test-source",
				Position:     1,
				CreatedAt:    now,
				Translations: []domain.RefTranslation{
					{
						ID:         uuid.New(),
						RefSenseID: sense2ID,
						Text:       "translation-3",
						SourceSlug: "test-source",
						Position:   0,
					},
				},
				Examples: []domain.RefExample{},
			},
		},
		Pronunciations: []domain.RefPronunciation{
			{
				ID:            uuid.New(),
				RefEntryID:    entryID,
				Transcription: ptrStr("/tEst/"),
				AudioURL:      ptrStr("https://example.com/audio/test-us.mp3"),
				Region:        ptrStr("us"),
				SourceSlug:    "test-source",
			},
			{
				ID:            uuid.New(),
				RefEntryID:    entryID,
				Transcription: ptrStr("/test/"),
				AudioURL:      ptrStr("https://example.com/audio/test-uk.mp3"),
				Region:        ptrStr("uk"),
				SourceSlug:    "test-source",
			},
		},
		Images: []domain.RefImage{
			{
				ID:         uuid.New(),
				RefEntryID: entryID,
				URL:        "https://example.com/images/test.jpg",
				Caption:    ptrStr("A test image"),
				SourceSlug: "test-source",
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Create tests
// ---------------------------------------------------------------------------

func TestRepo_Create_FullTree(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	input := buildRefEntry("Abundance-" + uuid.New().String()[:8])

	got, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	// Verify entry.
	if got.ID != input.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, input.ID)
	}
	if got.Text != input.Text {
		t.Errorf("Text mismatch: got %q, want %q", got.Text, input.Text)
	}
	if got.TextNormalized != input.TextNormalized {
		t.Errorf("TextNormalized mismatch: got %q, want %q", got.TextNormalized, input.TextNormalized)
	}

	// Verify senses.
	if len(got.Senses) != 2 {
		t.Fatalf("expected 2 senses, got %d", len(got.Senses))
	}
	if got.Senses[0].Definition != "a thing used to test" {
		t.Errorf("Sense[0].Definition mismatch: got %q", got.Senses[0].Definition)
	}
	if got.Senses[0].PartOfSpeech == nil || *got.Senses[0].PartOfSpeech != domain.PartOfSpeechNoun {
		t.Errorf("Sense[0].PartOfSpeech mismatch: got %v", got.Senses[0].PartOfSpeech)
	}
	if got.Senses[0].CEFRLevel == nil || *got.Senses[0].CEFRLevel != "B1" {
		t.Errorf("Sense[0].CEFRLevel mismatch: got %v", got.Senses[0].CEFRLevel)
	}
	if got.Senses[1].PartOfSpeech == nil || *got.Senses[1].PartOfSpeech != domain.PartOfSpeechVerb {
		t.Errorf("Sense[1].PartOfSpeech mismatch: got %v", got.Senses[1].PartOfSpeech)
	}
	if got.Senses[1].CEFRLevel != nil {
		t.Errorf("Sense[1].CEFRLevel should be nil, got %v", *got.Senses[1].CEFRLevel)
	}

	// Verify translations.
	if len(got.Senses[0].Translations) != 2 {
		t.Fatalf("expected 2 translations for sense[0], got %d", len(got.Senses[0].Translations))
	}
	if got.Senses[0].Translations[0].Text != "translation-1" {
		t.Errorf("Translation[0].Text mismatch: got %q", got.Senses[0].Translations[0].Text)
	}
	if len(got.Senses[1].Translations) != 1 {
		t.Fatalf("expected 1 translation for sense[1], got %d", len(got.Senses[1].Translations))
	}

	// Verify examples.
	if len(got.Senses[0].Examples) != 1 {
		t.Fatalf("expected 1 example for sense[0], got %d", len(got.Senses[0].Examples))
	}
	if got.Senses[0].Examples[0].Sentence != "This is a test sentence." {
		t.Errorf("Example[0].Sentence mismatch: got %q", got.Senses[0].Examples[0].Sentence)
	}
	if got.Senses[0].Examples[0].Translation == nil || *got.Senses[0].Examples[0].Translation != "Example translation" {
		t.Errorf("Example[0].Translation mismatch: got %v", got.Senses[0].Examples[0].Translation)
	}
	if len(got.Senses[1].Examples) != 0 {
		t.Fatalf("expected 0 examples for sense[1], got %d", len(got.Senses[1].Examples))
	}

	// Verify pronunciations.
	if len(got.Pronunciations) != 2 {
		t.Fatalf("expected 2 pronunciations, got %d", len(got.Pronunciations))
	}

	// Verify images.
	if len(got.Images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(got.Images))
	}
	if got.Images[0].URL != "https://example.com/images/test.jpg" {
		t.Errorf("Image[0].URL mismatch: got %q", got.Images[0].URL)
	}
	if got.Images[0].Caption == nil || *got.Images[0].Caption != "A test image" {
		t.Errorf("Image[0].Caption mismatch: got %v", got.Images[0].Caption)
	}
}

func TestRepo_Create_DuplicateText(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	text := "Duplicate-" + uuid.New().String()[:8]
	input1 := buildRefEntry(text)
	if _, err := repo.Create(ctx, input1); err != nil {
		t.Fatalf("Create first: %v", err)
	}

	// Second insert with same normalized text should fail.
	input2 := buildRefEntry(text)
	_, err := repo.Create(ctx, input2)
	assertIsDomainError(t, err, domain.ErrAlreadyExists)
}

func TestRepo_Create_AtomicRollback(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	text := "Rollback-" + uuid.New().String()[:8]
	input := buildRefEntry(text)

	// Set an invalid ref_sense_id on a translation to cause a FK error during creation.
	input.Senses[0].Translations[0].RefSenseID = uuid.New() // bogus sense ID
	// The sense ID in the Insert call uses sense.ID, not translation.RefSenseID,
	// so let's instead make a sense reference an invalid entry ID to trigger FK error.
	// Actually, the Create method uses entry.ID and sense.ID directly, so let me
	// instead cause an error by having a duplicate sense ID.
	input.Senses[1].ID = input.Senses[0].ID // duplicate PK

	_, err := repo.Create(ctx, input)
	if err == nil {
		t.Fatal("expected error from Create with duplicate sense ID")
	}

	// Verify entry was rolled back.
	var count int
	err = pool.QueryRow(ctx,
		`SELECT count(*) FROM ref_entries WHERE text_normalized = $1`,
		input.TextNormalized,
	).Scan(&count)
	if err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 ref_entries after rollback, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// GetByID tests
// ---------------------------------------------------------------------------

func TestRepo_GetByID_FullTree(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	seeded := testhelper.SeedRefEntry(t, pool, "GetByID-"+uuid.New().String()[:8])

	got, err := repo.GetByID(ctx, seeded.ID)
	if err != nil {
		t.Fatalf("GetByID: unexpected error: %v", err)
	}

	// Verify entry fields.
	if got.ID != seeded.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, seeded.ID)
	}
	if got.Text != seeded.Text {
		t.Errorf("Text mismatch: got %q, want %q", got.Text, seeded.Text)
	}
	if got.TextNormalized != seeded.TextNormalized {
		t.Errorf("TextNormalized mismatch: got %q, want %q", got.TextNormalized, seeded.TextNormalized)
	}

	// Verify senses loaded (SeedRefEntry creates 2 senses).
	if len(got.Senses) != 2 {
		t.Fatalf("expected 2 senses, got %d", len(got.Senses))
	}
	// Each sense has 2 translations.
	if len(got.Senses[0].Translations) != 2 {
		t.Errorf("expected 2 translations for sense[0], got %d", len(got.Senses[0].Translations))
	}
	if len(got.Senses[1].Translations) != 2 {
		t.Errorf("expected 2 translations for sense[1], got %d", len(got.Senses[1].Translations))
	}
	// Each sense has 2 examples.
	if len(got.Senses[0].Examples) != 2 {
		t.Errorf("expected 2 examples for sense[0], got %d", len(got.Senses[0].Examples))
	}
	if len(got.Senses[1].Examples) != 2 {
		t.Errorf("expected 2 examples for sense[1], got %d", len(got.Senses[1].Examples))
	}

	// Verify pronunciations loaded (SeedRefEntry creates 2).
	if len(got.Pronunciations) != 2 {
		t.Fatalf("expected 2 pronunciations, got %d", len(got.Pronunciations))
	}

	// Verify images (SeedRefEntry creates 0 images).
	if got.Images == nil {
		t.Fatal("expected non-nil Images slice")
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
// GetByNormalizedText tests
// ---------------------------------------------------------------------------

func TestRepo_GetByNormalizedText_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	seeded := testhelper.SeedRefEntry(t, pool, "ByText-"+uuid.New().String()[:8])

	got, err := repo.GetByNormalizedText(ctx, seeded.TextNormalized)
	if err != nil {
		t.Fatalf("GetByNormalizedText: unexpected error: %v", err)
	}

	if got.ID != seeded.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, seeded.ID)
	}
}

func TestRepo_GetByNormalizedText_NotFound(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	_, err := repo.GetByNormalizedText(ctx, "nonexistent-"+uuid.New().String()[:8])
	assertIsDomainError(t, err, domain.ErrNotFound)
}

// ---------------------------------------------------------------------------
// GetOrCreate tests
// ---------------------------------------------------------------------------

func TestRepo_GetOrCreate_NewEntry(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	text := "GetOrCreate-New-" + uuid.New().String()[:8]
	normalized := domain.NormalizeText(text)
	id := uuid.New()

	got, err := repo.GetOrCreate(ctx, id, text, normalized)
	if err != nil {
		t.Fatalf("GetOrCreate: unexpected error: %v", err)
	}

	if got.ID != id {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, id)
	}
	if got.TextNormalized != normalized {
		t.Errorf("TextNormalized mismatch: got %q, want %q", got.TextNormalized, normalized)
	}
}

func TestRepo_GetOrCreate_Existing(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	seeded := testhelper.SeedRefEntry(t, pool, "GetOrCreate-Existing-"+uuid.New().String()[:8])

	newID := uuid.New()
	got, err := repo.GetOrCreate(ctx, newID, seeded.Text, seeded.TextNormalized)
	if err != nil {
		t.Fatalf("GetOrCreate: unexpected error: %v", err)
	}

	// Should return the existing entry, not the new one.
	if got.ID != seeded.ID {
		t.Errorf("expected existing ID %s, got %s", seeded.ID, got.ID)
	}
}

func TestRepo_GetOrCreate_Concurrent(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	text := "Concurrent-" + uuid.New().String()[:8]
	normalized := domain.NormalizeText(text)

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	resultIDs := make([]uuid.UUID, goroutines)
	errs := make([]error, goroutines)

	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			id := uuid.New()
			got, err := repo.GetOrCreate(ctx, id, text, normalized)
			resultIDs[i] = got.ID
			errs[i] = err
		}()
	}

	wg.Wait()

	// All goroutines should succeed.
	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: unexpected error: %v", i, err)
		}
	}

	// All should return the same ID.
	firstID := resultIDs[0]
	for i := 1; i < goroutines; i++ {
		if resultIDs[i] != firstID {
			t.Errorf("goroutine %d returned ID %s, expected %s", i, resultIDs[i], firstID)
		}
	}
}

// ---------------------------------------------------------------------------
// Search tests
// ---------------------------------------------------------------------------

func TestRepo_Search_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	// Create entries with similar text.
	suffix := uuid.New().String()[:8]
	testhelper.SeedRefEntry(t, pool, "Elephant-"+suffix)
	testhelper.SeedRefEntry(t, pool, "Elephantine-"+suffix)
	testhelper.SeedRefEntry(t, pool, "Completely-Different-"+suffix)

	results, err := repo.Search(ctx, "elephant-"+suffix, 10)
	if err != nil {
		t.Fatalf("Search: unexpected error: %v", err)
	}

	// Should find at least 1 match (exact or fuzzy).
	if len(results) < 1 {
		t.Fatalf("expected at least 1 search result, got %d", len(results))
	}

	// First result should be the closest match.
	foundElephant := false
	for _, r := range results {
		if r.TextNormalized == domain.NormalizeText("Elephant-"+suffix) ||
			r.TextNormalized == domain.NormalizeText("Elephantine-"+suffix) {
			foundElephant = true
			break
		}
	}
	if !foundElephant {
		t.Error("expected to find elephant-related entry in search results")
	}
}

func TestRepo_Search_EmptyQuery(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	results, err := repo.Search(ctx, "", 10)
	if err != nil {
		t.Fatalf("Search with empty query: unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results for empty query, got %d", len(results))
	}
}

func TestRepo_Search_NoMatch(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	results, err := repo.Search(ctx, "zzzyyyxxx-nonexistent-"+uuid.New().String()[:8], 10)
	if err != nil {
		t.Fatalf("Search no match: unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// Batch query tests
// ---------------------------------------------------------------------------

func TestRepo_GetRefSensesByIDs(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	seeded := testhelper.SeedRefEntry(t, pool, "BatchSenses-"+uuid.New().String()[:8])

	ids := make([]uuid.UUID, len(seeded.Senses))
	for i, s := range seeded.Senses {
		ids[i] = s.ID
	}

	got, err := repo.GetRefSensesByIDs(ctx, ids)
	if err != nil {
		t.Fatalf("GetRefSensesByIDs: unexpected error: %v", err)
	}

	if len(got) != len(seeded.Senses) {
		t.Fatalf("expected %d senses, got %d", len(seeded.Senses), len(got))
	}

	// Build a set of returned IDs.
	gotIDs := make(map[uuid.UUID]bool)
	for _, s := range got {
		gotIDs[s.ID] = true
	}
	for _, id := range ids {
		if !gotIDs[id] {
			t.Errorf("missing sense ID %s in results", id)
		}
	}
}

func TestRepo_GetRefSensesByIDs_Empty(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	got, err := repo.GetRefSensesByIDs(ctx, []uuid.UUID{})
	if err != nil {
		t.Fatalf("GetRefSensesByIDs empty: unexpected error: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("expected 0 senses, got %d", len(got))
	}
}

func TestRepo_GetRefTranslationsByIDs(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	seeded := testhelper.SeedRefEntry(t, pool, "BatchTranslations-"+uuid.New().String()[:8])

	// Collect all translation IDs.
	var ids []uuid.UUID
	for _, s := range seeded.Senses {
		for _, tr := range s.Translations {
			ids = append(ids, tr.ID)
		}
	}

	got, err := repo.GetRefTranslationsByIDs(ctx, ids)
	if err != nil {
		t.Fatalf("GetRefTranslationsByIDs: unexpected error: %v", err)
	}

	if len(got) != len(ids) {
		t.Fatalf("expected %d translations, got %d", len(ids), len(got))
	}
}

func TestRepo_GetRefExamplesByIDs(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	seeded := testhelper.SeedRefEntry(t, pool, "BatchExamples-"+uuid.New().String()[:8])

	// Collect all example IDs.
	var ids []uuid.UUID
	for _, s := range seeded.Senses {
		for _, ex := range s.Examples {
			ids = append(ids, ex.ID)
		}
	}

	got, err := repo.GetRefExamplesByIDs(ctx, ids)
	if err != nil {
		t.Fatalf("GetRefExamplesByIDs: unexpected error: %v", err)
	}

	if len(got) != len(ids) {
		t.Fatalf("expected %d examples, got %d", len(ids), len(got))
	}
}

func TestRepo_GetRefPronunciationsByIDs(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	seeded := testhelper.SeedRefEntry(t, pool, "BatchPronunciations-"+uuid.New().String()[:8])

	ids := make([]uuid.UUID, len(seeded.Pronunciations))
	for i, p := range seeded.Pronunciations {
		ids[i] = p.ID
	}

	got, err := repo.GetRefPronunciationsByIDs(ctx, ids)
	if err != nil {
		t.Fatalf("GetRefPronunciationsByIDs: unexpected error: %v", err)
	}

	if len(got) != len(ids) {
		t.Fatalf("expected %d pronunciations, got %d", len(ids), len(got))
	}

	// Verify pronunciation fields are correctly mapped.
	for _, p := range got {
		if p.Transcription == nil {
			t.Error("expected non-nil Transcription")
		}
		if p.SourceSlug != "test-source" {
			t.Errorf("SourceSlug mismatch: got %q", p.SourceSlug)
		}
	}
}

func TestRepo_GetRefImagesByIDs(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	// Create an entry with images via the repo.
	input := buildRefEntry("BatchImages-" + uuid.New().String()[:8])

	created, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ids := make([]uuid.UUID, len(created.Images))
	for i, img := range created.Images {
		ids[i] = img.ID
	}

	got, err := repo.GetRefImagesByIDs(ctx, ids)
	if err != nil {
		t.Fatalf("GetRefImagesByIDs: unexpected error: %v", err)
	}

	if len(got) != len(ids) {
		t.Fatalf("expected %d images, got %d", len(ids), len(got))
	}

	// Verify image fields.
	for _, img := range got {
		if img.URL == "" {
			t.Error("expected non-empty URL")
		}
		if img.SourceSlug != "test-source" {
			t.Errorf("SourceSlug mismatch: got %q", img.SourceSlug)
		}
	}
}

func TestRepo_GetRefImagesByIDs_Empty(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	got, err := repo.GetRefImagesByIDs(ctx, []uuid.UUID{})
	if err != nil {
		t.Fatalf("GetRefImagesByIDs empty: unexpected error: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("expected 0 images, got %d", len(got))
	}
}

// ---------------------------------------------------------------------------
// Create + GetByID round-trip test
// ---------------------------------------------------------------------------

func TestRepo_Create_ThenGetByID_RoundTrip(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	input := buildRefEntry("RoundTrip-" + uuid.New().String()[:8])

	created, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID after Create: %v", err)
	}

	// Verify the full tree matches.
	if got.ID != created.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, created.ID)
	}
	if len(got.Senses) != len(created.Senses) {
		t.Fatalf("senses count mismatch: got %d, want %d", len(got.Senses), len(created.Senses))
	}

	for i, wantSense := range created.Senses {
		gotSense := got.Senses[i]
		if gotSense.ID != wantSense.ID {
			t.Errorf("sense[%d] ID mismatch: got %s, want %s", i, gotSense.ID, wantSense.ID)
		}
		if len(gotSense.Translations) != len(wantSense.Translations) {
			t.Errorf("sense[%d] translations count mismatch: got %d, want %d",
				i, len(gotSense.Translations), len(wantSense.Translations))
		}
		if len(gotSense.Examples) != len(wantSense.Examples) {
			t.Errorf("sense[%d] examples count mismatch: got %d, want %d",
				i, len(gotSense.Examples), len(wantSense.Examples))
		}
	}

	if len(got.Pronunciations) != len(created.Pronunciations) {
		t.Errorf("pronunciations count mismatch: got %d, want %d",
			len(got.Pronunciations), len(created.Pronunciations))
	}

	if len(got.Images) != len(created.Images) {
		t.Errorf("images count mismatch: got %d, want %d", len(got.Images), len(created.Images))
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func ptrStr(s string) *string {
	return &s
}

func assertIsDomainError(t *testing.T, err error, target error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error wrapping %v, got nil", target)
	}
	if !errors.Is(err, target) {
		t.Fatalf("expected error wrapping %v, got: %v", target, err)
	}
}
