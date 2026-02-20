package refentry_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/testhelper"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// ---------------------------------------------------------------------------
// BulkInsertEntries
// ---------------------------------------------------------------------------

func TestRepo_BulkInsertEntries_Basic(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	entries := []domain.RefEntry{
		makeRefEntry("bulk-entry-1-" + uuid.New().String()[:8]),
		makeRefEntry("bulk-entry-2-" + uuid.New().String()[:8]),
	}

	inserted, err := repo.BulkInsertEntries(ctx, entries)
	if err != nil {
		t.Fatalf("BulkInsertEntries: %v", err)
	}
	if inserted != 2 {
		t.Errorf("expected 2 inserted, got %d", inserted)
	}
}

func TestRepo_BulkInsertEntries_Idempotent(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	entries := []domain.RefEntry{
		makeRefEntry("bulk-idem-" + uuid.New().String()[:8]),
	}

	inserted1, err := repo.BulkInsertEntries(ctx, entries)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if inserted1 != 1 {
		t.Errorf("first: expected 1 inserted, got %d", inserted1)
	}

	// Re-insert with same text_normalized — should skip.
	entries[0].ID = uuid.New() // different ID, same text_normalized
	inserted2, err := repo.BulkInsertEntries(ctx, entries)
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}
	if inserted2 != 0 {
		t.Errorf("second: expected 0 inserted (idempotent), got %d", inserted2)
	}
}

func TestRepo_BulkInsertEntries_Empty(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	inserted, err := repo.BulkInsertEntries(ctx, nil)
	if err != nil {
		t.Fatalf("BulkInsertEntries empty: %v", err)
	}
	if inserted != 0 {
		t.Errorf("expected 0, got %d", inserted)
	}
}

func TestRepo_BulkInsertEntries_WithMetadata(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	rank := 42
	cefr := "B1"
	entry := makeRefEntry("bulk-meta-" + uuid.New().String()[:8])
	entry.FrequencyRank = &rank
	entry.CEFRLevel = &cefr
	entry.IsCoreLexicon = true

	inserted, err := repo.BulkInsertEntries(ctx, []domain.RefEntry{entry})
	if err != nil {
		t.Fatalf("BulkInsertEntries with metadata: %v", err)
	}
	if inserted != 1 {
		t.Fatalf("expected 1 inserted, got %d", inserted)
	}

	// Verify metadata by reading back.
	got, err := repo.GetFullTreeByText(ctx, entry.TextNormalized)
	if err != nil {
		t.Fatalf("GetFullTreeByText: %v", err)
	}
	if got.FrequencyRank == nil || *got.FrequencyRank != 42 {
		t.Errorf("FrequencyRank mismatch: got %v", got.FrequencyRank)
	}
	if got.CEFRLevel == nil || *got.CEFRLevel != "B1" {
		t.Errorf("CEFRLevel mismatch: got %v", got.CEFRLevel)
	}
	if !got.IsCoreLexicon {
		t.Error("IsCoreLexicon should be true")
	}
}

// ---------------------------------------------------------------------------
// BulkInsertSenses
// ---------------------------------------------------------------------------

func TestRepo_BulkInsertSenses_Basic(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	seeded := testhelper.SeedRefEntry(t, pool, "bulk-senses-"+uuid.New().String()[:8])

	senses := []domain.RefSense{
		{
			ID:         uuid.New(),
			RefEntryID: seeded.ID,
			Definition: "new sense definition",
			SourceSlug: "test-source",
			Position:   10,
			CreatedAt:  time.Now().UTC().Truncate(time.Microsecond),
		},
	}

	inserted, err := repo.BulkInsertSenses(ctx, senses)
	if err != nil {
		t.Fatalf("BulkInsertSenses: %v", err)
	}
	if inserted != 1 {
		t.Errorf("expected 1 inserted, got %d", inserted)
	}
}

func TestRepo_BulkInsertSenses_Idempotent(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	seeded := testhelper.SeedRefEntry(t, pool, "bulk-senses-idem-"+uuid.New().String()[:8])

	senses := []domain.RefSense{
		{
			ID:         uuid.New(),
			RefEntryID: seeded.ID,
			Definition: "idem sense",
			SourceSlug: "test-source",
			Position:   10,
			CreatedAt:  time.Now().UTC().Truncate(time.Microsecond),
		},
	}

	if _, err := repo.BulkInsertSenses(ctx, senses); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	inserted2, err := repo.BulkInsertSenses(ctx, senses)
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}
	if inserted2 != 0 {
		t.Errorf("expected 0 inserted (idempotent), got %d", inserted2)
	}
}

func TestRepo_BulkInsertSenses_FKIntegrity(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	senses := []domain.RefSense{
		{
			ID:         uuid.New(),
			RefEntryID: uuid.New(), // non-existent entry
			Definition: "orphan sense",
			SourceSlug: "test-source",
			Position:   0,
			CreatedAt:  time.Now().UTC().Truncate(time.Microsecond),
		},
	}

	_, err := repo.BulkInsertSenses(ctx, senses)
	if err == nil {
		t.Fatal("expected FK error for non-existent entry")
	}
}

// ---------------------------------------------------------------------------
// BulkInsertTranslations
// ---------------------------------------------------------------------------

func TestRepo_BulkInsertTranslations_Basic(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	seeded := testhelper.SeedRefEntry(t, pool, "bulk-tr-"+uuid.New().String()[:8])

	translations := []domain.RefTranslation{
		{
			ID:         uuid.New(),
			RefSenseID: seeded.Senses[0].ID,
			Text:       "bulk translation",
			SourceSlug: "test-source",
			Position:   10,
		},
	}

	inserted, err := repo.BulkInsertTranslations(ctx, translations)
	if err != nil {
		t.Fatalf("BulkInsertTranslations: %v", err)
	}
	if inserted != 1 {
		t.Errorf("expected 1 inserted, got %d", inserted)
	}
}

func TestRepo_BulkInsertTranslations_Idempotent(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	seeded := testhelper.SeedRefEntry(t, pool, "bulk-tr-idem-"+uuid.New().String()[:8])

	translations := []domain.RefTranslation{
		{
			ID:         uuid.New(),
			RefSenseID: seeded.Senses[0].ID,
			Text:       "idem translation",
			SourceSlug: "test-source",
			Position:   10,
		},
	}

	if _, err := repo.BulkInsertTranslations(ctx, translations); err != nil {
		t.Fatalf("first: %v", err)
	}

	inserted2, err := repo.BulkInsertTranslations(ctx, translations)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if inserted2 != 0 {
		t.Errorf("expected 0 (idempotent), got %d", inserted2)
	}
}

// ---------------------------------------------------------------------------
// BulkInsertExamples
// ---------------------------------------------------------------------------

func TestRepo_BulkInsertExamples_Basic(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	seeded := testhelper.SeedRefEntry(t, pool, "bulk-ex-"+uuid.New().String()[:8])
	tr := "example translation"

	examples := []domain.RefExample{
		{
			ID:          uuid.New(),
			RefSenseID:  seeded.Senses[0].ID,
			Sentence:    "This is a bulk example sentence.",
			Translation: &tr,
			SourceSlug:  "test-source",
			Position:    10,
		},
	}

	inserted, err := repo.BulkInsertExamples(ctx, examples)
	if err != nil {
		t.Fatalf("BulkInsertExamples: %v", err)
	}
	if inserted != 1 {
		t.Errorf("expected 1 inserted, got %d", inserted)
	}
}

func TestRepo_BulkInsertExamples_Idempotent(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	seeded := testhelper.SeedRefEntry(t, pool, "bulk-ex-idem-"+uuid.New().String()[:8])

	examples := []domain.RefExample{
		{
			ID:         uuid.New(),
			RefSenseID: seeded.Senses[0].ID,
			Sentence:   "Idem example.",
			SourceSlug: "test-source",
			Position:   10,
		},
	}

	if _, err := repo.BulkInsertExamples(ctx, examples); err != nil {
		t.Fatalf("first: %v", err)
	}

	inserted2, err := repo.BulkInsertExamples(ctx, examples)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if inserted2 != 0 {
		t.Errorf("expected 0 (idempotent), got %d", inserted2)
	}
}

// ---------------------------------------------------------------------------
// BulkInsertPronunciations
// ---------------------------------------------------------------------------

func TestRepo_BulkInsertPronunciations_Basic(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	seeded := testhelper.SeedRefEntry(t, pool, "bulk-pron-"+uuid.New().String()[:8])
	ipa := "/bʌlk/"

	pronunciations := []domain.RefPronunciation{
		{
			ID:            uuid.New(),
			RefEntryID:    seeded.ID,
			Transcription: &ipa,
			SourceSlug:    "test-source",
		},
	}

	inserted, err := repo.BulkInsertPronunciations(ctx, pronunciations)
	if err != nil {
		t.Fatalf("BulkInsertPronunciations: %v", err)
	}
	if inserted != 1 {
		t.Errorf("expected 1 inserted, got %d", inserted)
	}
}

func TestRepo_BulkInsertPronunciations_Idempotent(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	seeded := testhelper.SeedRefEntry(t, pool, "bulk-pron-idem-"+uuid.New().String()[:8])
	ipa := "/aɪdm/"

	pronunciations := []domain.RefPronunciation{
		{
			ID:            uuid.New(),
			RefEntryID:    seeded.ID,
			Transcription: &ipa,
			SourceSlug:    "test-source",
		},
	}

	if _, err := repo.BulkInsertPronunciations(ctx, pronunciations); err != nil {
		t.Fatalf("first: %v", err)
	}

	inserted2, err := repo.BulkInsertPronunciations(ctx, pronunciations)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if inserted2 != 0 {
		t.Errorf("expected 0 (idempotent), got %d", inserted2)
	}
}

// ---------------------------------------------------------------------------
// BulkInsertRelations
// ---------------------------------------------------------------------------

func TestRepo_BulkInsertRelations_Basic(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	entry1 := testhelper.SeedRefEntry(t, pool, "rel-source-"+uuid.New().String()[:8])
	entry2 := testhelper.SeedRefEntry(t, pool, "rel-target-"+uuid.New().String()[:8])

	// Need to seed data source for the FK constraint.
	// "wordnet" is pre-seeded by migration 00013.

	relations := []domain.RefWordRelation{
		{
			ID:            uuid.New(),
			SourceEntryID: entry1.ID,
			TargetEntryID: entry2.ID,
			RelationType:  "synonym",
			SourceSlug:    "wordnet",
			CreatedAt:     time.Now().UTC().Truncate(time.Microsecond),
		},
	}

	inserted, err := repo.BulkInsertRelations(ctx, relations)
	if err != nil {
		t.Fatalf("BulkInsertRelations: %v", err)
	}
	if inserted != 1 {
		t.Errorf("expected 1 inserted, got %d", inserted)
	}
}

func TestRepo_BulkInsertRelations_Idempotent(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	entry1 := testhelper.SeedRefEntry(t, pool, "rel-idem-s-"+uuid.New().String()[:8])
	entry2 := testhelper.SeedRefEntry(t, pool, "rel-idem-t-"+uuid.New().String()[:8])

	relations := []domain.RefWordRelation{
		{
			ID:            uuid.New(),
			SourceEntryID: entry1.ID,
			TargetEntryID: entry2.ID,
			RelationType:  "synonym",
			SourceSlug:    "wordnet",
			CreatedAt:     time.Now().UTC().Truncate(time.Microsecond),
		},
	}

	if _, err := repo.BulkInsertRelations(ctx, relations); err != nil {
		t.Fatalf("first: %v", err)
	}

	// Same source/target/type — should skip.
	relations[0].ID = uuid.New()
	inserted2, err := repo.BulkInsertRelations(ctx, relations)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if inserted2 != 0 {
		t.Errorf("expected 0 (idempotent), got %d", inserted2)
	}
}

// ---------------------------------------------------------------------------
// BulkInsertCoverage
// ---------------------------------------------------------------------------

func TestRepo_BulkInsertCoverage_Basic(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	entry := testhelper.SeedRefEntry(t, pool, "cov-basic-"+uuid.New().String()[:8])

	coverage := []domain.RefEntrySourceCoverage{
		{
			RefEntryID: entry.ID,
			SourceSlug: "wiktionary",
			Status:     "fetched",
			FetchedAt:  time.Now().UTC().Truncate(time.Microsecond),
		},
	}

	inserted, err := repo.BulkInsertCoverage(ctx, coverage)
	if err != nil {
		t.Fatalf("BulkInsertCoverage: %v", err)
	}
	if inserted != 1 {
		t.Errorf("expected 1 inserted, got %d", inserted)
	}
}

func TestRepo_BulkInsertCoverage_UpsertOnConflict(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	entry := testhelper.SeedRefEntry(t, pool, "cov-upsert-"+uuid.New().String()[:8])
	now := time.Now().UTC().Truncate(time.Microsecond)

	coverage := []domain.RefEntrySourceCoverage{
		{
			RefEntryID: entry.ID,
			SourceSlug: "ngsl",
			Status:     "fetched",
			FetchedAt:  now,
		},
	}

	if _, err := repo.BulkInsertCoverage(ctx, coverage); err != nil {
		t.Fatalf("first: %v", err)
	}

	// Update status — should succeed via ON CONFLICT DO UPDATE.
	later := now.Add(time.Hour)
	coverage[0].Status = "no_data"
	coverage[0].FetchedAt = later

	inserted2, err := repo.BulkInsertCoverage(ctx, coverage)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if inserted2 != 1 {
		t.Errorf("expected 1 (upsert), got %d", inserted2)
	}

	// Verify the update took effect.
	cov, err := repo.GetCoverageByEntryID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("GetCoverageByEntryID: %v", err)
	}
	found := false
	for _, c := range cov {
		if c.SourceSlug == "ngsl" {
			found = true
			if c.Status != "no_data" {
				t.Errorf("expected status 'no_data', got %q", c.Status)
			}
		}
	}
	if !found {
		t.Error("ngsl coverage record not found")
	}
}

// ---------------------------------------------------------------------------
// BulkUpdateEntryMetadata
// ---------------------------------------------------------------------------

func TestRepo_BulkUpdateEntryMetadata_Basic(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	entry := testhelper.SeedRefEntry(t, pool, "meta-update-"+uuid.New().String()[:8])

	rank := 100
	cefr := "A2"
	isCore := true
	updates := []domain.EntryMetadataUpdate{
		{
			TextNormalized: entry.TextNormalized,
			FrequencyRank:  &rank,
			CEFRLevel:      &cefr,
			IsCoreLexicon:  &isCore,
		},
	}

	updated, err := repo.BulkUpdateEntryMetadata(ctx, updates)
	if err != nil {
		t.Fatalf("BulkUpdateEntryMetadata: %v", err)
	}
	if updated != 1 {
		t.Errorf("expected 1 updated, got %d", updated)
	}

	// Verify by reading back.
	got, err := repo.GetFullTreeByText(ctx, entry.TextNormalized)
	if err != nil {
		t.Fatalf("GetFullTreeByText: %v", err)
	}
	if got.FrequencyRank == nil || *got.FrequencyRank != 100 {
		t.Errorf("FrequencyRank: expected 100, got %v", got.FrequencyRank)
	}
	if got.CEFRLevel == nil || *got.CEFRLevel != "A2" {
		t.Errorf("CEFRLevel: expected A2, got %v", got.CEFRLevel)
	}
	if !got.IsCoreLexicon {
		t.Error("IsCoreLexicon should be true")
	}
}

func TestRepo_BulkUpdateEntryMetadata_NoMatch(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	rank := 1
	updates := []domain.EntryMetadataUpdate{
		{
			TextNormalized: "nonexistent-" + uuid.New().String()[:8],
			FrequencyRank:  &rank,
		},
	}

	updated, err := repo.BulkUpdateEntryMetadata(ctx, updates)
	if err != nil {
		t.Fatalf("BulkUpdateEntryMetadata: %v", err)
	}
	if updated != 0 {
		t.Errorf("expected 0 updated (no match), got %d", updated)
	}
}

// ---------------------------------------------------------------------------
// GetEntryIDsByNormalizedTexts
// ---------------------------------------------------------------------------

func TestRepo_GetEntryIDsByNormalizedTexts_Basic(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	entry1 := testhelper.SeedRefEntry(t, pool, "lookup-1-"+uuid.New().String()[:8])
	entry2 := testhelper.SeedRefEntry(t, pool, "lookup-2-"+uuid.New().String()[:8])

	result, err := repo.GetEntryIDsByNormalizedTexts(ctx, []string{
		entry1.TextNormalized,
		entry2.TextNormalized,
		"nonexistent-" + uuid.New().String()[:8],
	})
	if err != nil {
		t.Fatalf("GetEntryIDsByNormalizedTexts: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	if result[entry1.TextNormalized] != entry1.ID {
		t.Errorf("entry1 ID mismatch")
	}
	if result[entry2.TextNormalized] != entry2.ID {
		t.Errorf("entry2 ID mismatch")
	}
}

func TestRepo_GetEntryIDsByNormalizedTexts_Empty(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	result, err := repo.GetEntryIDsByNormalizedTexts(ctx, nil)
	if err != nil {
		t.Fatalf("empty: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d", len(result))
	}
}

// ---------------------------------------------------------------------------
// GetAllNormalizedTexts
// ---------------------------------------------------------------------------

func TestRepo_GetAllNormalizedTexts(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	entry := testhelper.SeedRefEntry(t, pool, "alltext-"+uuid.New().String()[:8])

	result, err := repo.GetAllNormalizedTexts(ctx)
	if err != nil {
		t.Fatalf("GetAllNormalizedTexts: %v", err)
	}

	if !result[entry.TextNormalized] {
		t.Errorf("expected %q in result set", entry.TextNormalized)
	}
}

// ---------------------------------------------------------------------------
// GetFirstSenseIDsByEntryIDs
// ---------------------------------------------------------------------------

func TestRepo_GetFirstSenseIDsByEntryIDs_Basic(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	entry := testhelper.SeedRefEntry(t, pool, "first-sense-"+uuid.New().String()[:8])

	result, err := repo.GetFirstSenseIDsByEntryIDs(ctx, []uuid.UUID{entry.ID})
	if err != nil {
		t.Fatalf("GetFirstSenseIDsByEntryIDs: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}

	// SeedRefEntry creates 2 senses with position 0 and 1.
	// The first sense (position 0) should be returned.
	senseID := result[entry.ID]
	if senseID != entry.Senses[0].ID {
		t.Errorf("expected first sense %s, got %s", entry.Senses[0].ID, senseID)
	}
}

func TestRepo_GetFirstSenseIDsByEntryIDs_Empty(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	result, err := repo.GetFirstSenseIDsByEntryIDs(ctx, nil)
	if err != nil {
		t.Fatalf("empty: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d", len(result))
	}
}

// ---------------------------------------------------------------------------
// UpsertDataSources
// ---------------------------------------------------------------------------

func TestRepo_UpsertDataSources_Basic(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Microsecond)
	sources := []domain.RefDataSource{
		{
			Slug:           "test-source-" + uuid.New().String()[:8],
			Name:           "Test Source",
			Description:    "A test data source",
			SourceType:     "definitions",
			IsActive:       true,
			DatasetVersion: "1.0",
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}

	err := repo.UpsertDataSources(ctx, sources)
	if err != nil {
		t.Fatalf("UpsertDataSources: %v", err)
	}

	// Verify by reading back.
	got, err := repo.GetDataSourceBySlug(ctx, sources[0].Slug)
	if err != nil {
		t.Fatalf("GetDataSourceBySlug: %v", err)
	}
	if got.Name != "Test Source" {
		t.Errorf("Name mismatch: got %q", got.Name)
	}
	if got.DatasetVersion != "1.0" {
		t.Errorf("DatasetVersion mismatch: got %q", got.DatasetVersion)
	}
}

func TestRepo_UpsertDataSources_UpdateOnConflict(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	slug := "test-upsert-" + uuid.New().String()[:8]
	now := time.Now().UTC().Truncate(time.Microsecond)

	sources := []domain.RefDataSource{
		{
			Slug:           slug,
			Name:           "Version 1",
			SourceType:     "definitions",
			IsActive:       true,
			DatasetVersion: "1.0",
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}

	if err := repo.UpsertDataSources(ctx, sources); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	// Update the source.
	later := now.Add(time.Hour)
	sources[0].Name = "Version 2"
	sources[0].DatasetVersion = "2.0"
	sources[0].UpdatedAt = later

	if err := repo.UpsertDataSources(ctx, sources); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	got, err := repo.GetDataSourceBySlug(ctx, slug)
	if err != nil {
		t.Fatalf("GetDataSourceBySlug: %v", err)
	}
	if got.Name != "Version 2" {
		t.Errorf("Name not updated: got %q", got.Name)
	}
	if got.DatasetVersion != "2.0" {
		t.Errorf("DatasetVersion not updated: got %q", got.DatasetVersion)
	}
}

// ---------------------------------------------------------------------------
// GetRelationsByEntryID
// ---------------------------------------------------------------------------

func TestRepo_GetRelationsByEntryID(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	entry1 := testhelper.SeedRefEntry(t, pool, "rel-query-s-"+uuid.New().String()[:8])
	entry2 := testhelper.SeedRefEntry(t, pool, "rel-query-t-"+uuid.New().String()[:8])

	relations := []domain.RefWordRelation{
		{
			ID:            uuid.New(),
			SourceEntryID: entry1.ID,
			TargetEntryID: entry2.ID,
			RelationType:  "synonym",
			SourceSlug:    "wordnet",
			CreatedAt:     time.Now().UTC().Truncate(time.Microsecond),
		},
	}

	if _, err := repo.BulkInsertRelations(ctx, relations); err != nil {
		t.Fatalf("BulkInsertRelations: %v", err)
	}

	// Query from source side.
	got, err := repo.GetRelationsByEntryID(ctx, entry1.ID)
	if err != nil {
		t.Fatalf("GetRelationsByEntryID(source): %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 relation from source, got %d", len(got))
	}

	// Query from target side — should also find the relation.
	got2, err := repo.GetRelationsByEntryID(ctx, entry2.ID)
	if err != nil {
		t.Fatalf("GetRelationsByEntryID(target): %v", err)
	}
	if len(got2) != 1 {
		t.Fatalf("expected 1 relation from target, got %d", len(got2))
	}
}

// ---------------------------------------------------------------------------
// GetAllDataSources
// ---------------------------------------------------------------------------

func TestRepo_GetAllDataSources(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	got, err := repo.GetAllDataSources(ctx)
	if err != nil {
		t.Fatalf("GetAllDataSources: %v", err)
	}

	// Migration 00013 seeds 8 data sources (freedict, translate, wiktionary, ngsl, nawl, cmu, wordnet, tatoeba).
	if len(got) < 7 {
		t.Errorf("expected at least 7 data sources, got %d", len(got))
	}
}

// ---------------------------------------------------------------------------
// GetCoverageByEntryIDs (batch)
// ---------------------------------------------------------------------------

func TestRepo_GetCoverageByEntryIDs(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	entry1 := testhelper.SeedRefEntry(t, pool, "cov-batch-1-"+uuid.New().String()[:8])
	entry2 := testhelper.SeedRefEntry(t, pool, "cov-batch-2-"+uuid.New().String()[:8])
	now := time.Now().UTC().Truncate(time.Microsecond)

	coverage := []domain.RefEntrySourceCoverage{
		{RefEntryID: entry1.ID, SourceSlug: "wiktionary", Status: "fetched", FetchedAt: now},
		{RefEntryID: entry1.ID, SourceSlug: "ngsl", Status: "fetched", FetchedAt: now},
		{RefEntryID: entry2.ID, SourceSlug: "wiktionary", Status: "no_data", FetchedAt: now},
	}

	if _, err := repo.BulkInsertCoverage(ctx, coverage); err != nil {
		t.Fatalf("BulkInsertCoverage: %v", err)
	}

	result, err := repo.GetCoverageByEntryIDs(ctx, []uuid.UUID{entry1.ID, entry2.ID})
	if err != nil {
		t.Fatalf("GetCoverageByEntryIDs: %v", err)
	}

	if len(result[entry1.ID]) != 2 {
		t.Errorf("entry1: expected 2 coverage records, got %d", len(result[entry1.ID]))
	}
	if len(result[entry2.ID]) != 1 {
		t.Errorf("entry2: expected 1 coverage record, got %d", len(result[entry2.ID]))
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func makeRefEntry(text string) domain.RefEntry {
	return domain.RefEntry{
		ID:             uuid.New(),
		Text:           text,
		TextNormalized: domain.NormalizeText(text),
		CreatedAt:      time.Now().UTC().Truncate(time.Microsecond),
	}
}
