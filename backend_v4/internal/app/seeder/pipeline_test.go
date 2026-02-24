package seeder

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// mockRepo records calls to verify pipeline behavior.
type mockRepo struct {
	mu sync.Mutex

	entriesInserted        int
	sensesInserted         int
	translationsInserted   int
	examplesInserted       int
	pronunciationsInserted int
	relationsInserted      int
	coverageInserted       int
	metadataUpdated        int
	dataSourcesUpserted    bool

	bulkInsertEntriesErr        error
	bulkInsertSensesErr         error
	bulkInsertTranslationsErr   error
	bulkInsertExamplesErr       error
	bulkInsertPronunciationsErr error
	bulkInsertRelationsErr      error
	bulkInsertCoverageErr       error
	bulkUpdateMetadataErr       error
	upsertDataSourcesErr        error

	normalizedTexts     map[string]bool
	entryIDMap          map[string]uuid.UUID
	senseIDMap          map[uuid.UUID]uuid.UUID
	getAllTextsErr       error
	getEntryIDsErr      error
	getFirstSenseIDsErr error

	callLog []string
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		normalizedTexts: make(map[string]bool),
		entryIDMap:      make(map[string]uuid.UUID),
		senseIDMap:      make(map[uuid.UUID]uuid.UUID),
	}
}

func (m *mockRepo) logCall(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callLog = append(m.callLog, name)
}

func (m *mockRepo) BulkInsertEntries(_ context.Context, entries []domain.RefEntry) (int, error) {
	m.logCall("BulkInsertEntries")
	if m.bulkInsertEntriesErr != nil {
		return 0, m.bulkInsertEntriesErr
	}
	m.mu.Lock()
	m.entriesInserted += len(entries)
	m.mu.Unlock()
	return len(entries), nil
}

func (m *mockRepo) BulkInsertSenses(_ context.Context, senses []domain.RefSense) (int, error) {
	m.logCall("BulkInsertSenses")
	if m.bulkInsertSensesErr != nil {
		return 0, m.bulkInsertSensesErr
	}
	m.mu.Lock()
	m.sensesInserted += len(senses)
	m.mu.Unlock()
	return len(senses), nil
}

func (m *mockRepo) BulkInsertTranslations(_ context.Context, translations []domain.RefTranslation) (int, error) {
	m.logCall("BulkInsertTranslations")
	if m.bulkInsertTranslationsErr != nil {
		return 0, m.bulkInsertTranslationsErr
	}
	m.mu.Lock()
	m.translationsInserted += len(translations)
	m.mu.Unlock()
	return len(translations), nil
}

func (m *mockRepo) BulkInsertExamples(_ context.Context, examples []domain.RefExample) (int, error) {
	m.logCall("BulkInsertExamples")
	if m.bulkInsertExamplesErr != nil {
		return 0, m.bulkInsertExamplesErr
	}
	m.mu.Lock()
	m.examplesInserted += len(examples)
	m.mu.Unlock()
	return len(examples), nil
}

func (m *mockRepo) BulkInsertPronunciations(_ context.Context, pronunciations []domain.RefPronunciation) (int, error) {
	m.logCall("BulkInsertPronunciations")
	if m.bulkInsertPronunciationsErr != nil {
		return 0, m.bulkInsertPronunciationsErr
	}
	m.mu.Lock()
	m.pronunciationsInserted += len(pronunciations)
	m.mu.Unlock()
	return len(pronunciations), nil
}

func (m *mockRepo) BulkInsertRelations(_ context.Context, relations []domain.RefWordRelation) (int, error) {
	m.logCall("BulkInsertRelations")
	if m.bulkInsertRelationsErr != nil {
		return 0, m.bulkInsertRelationsErr
	}
	m.mu.Lock()
	m.relationsInserted += len(relations)
	m.mu.Unlock()
	return len(relations), nil
}

func (m *mockRepo) BulkInsertCoverage(_ context.Context, coverage []domain.RefEntrySourceCoverage) (int, error) {
	m.logCall("BulkInsertCoverage")
	if m.bulkInsertCoverageErr != nil {
		return 0, m.bulkInsertCoverageErr
	}
	m.mu.Lock()
	m.coverageInserted += len(coverage)
	m.mu.Unlock()
	return len(coverage), nil
}

func (m *mockRepo) BulkUpdateEntryMetadata(_ context.Context, updates []domain.EntryMetadataUpdate) (int, error) {
	m.logCall("BulkUpdateEntryMetadata")
	if m.bulkUpdateMetadataErr != nil {
		return 0, m.bulkUpdateMetadataErr
	}
	m.mu.Lock()
	m.metadataUpdated += len(updates)
	m.mu.Unlock()
	return len(updates), nil
}

func (m *mockRepo) GetEntryIDsByNormalizedTexts(_ context.Context, texts []string) (map[string]uuid.UUID, error) {
	m.logCall("GetEntryIDsByNormalizedTexts")
	if m.getEntryIDsErr != nil {
		return nil, m.getEntryIDsErr
	}
	result := make(map[string]uuid.UUID)
	for _, t := range texts {
		if id, ok := m.entryIDMap[t]; ok {
			result[t] = id
		}
	}
	return result, nil
}

func (m *mockRepo) GetAllNormalizedTexts(_ context.Context) (map[string]bool, error) {
	m.logCall("GetAllNormalizedTexts")
	if m.getAllTextsErr != nil {
		return nil, m.getAllTextsErr
	}
	return m.normalizedTexts, nil
}

func (m *mockRepo) GetFirstSenseIDsByEntryIDs(_ context.Context, entryIDs []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	m.logCall("GetFirstSenseIDsByEntryIDs")
	if m.getFirstSenseIDsErr != nil {
		return nil, m.getFirstSenseIDsErr
	}
	result := make(map[uuid.UUID]uuid.UUID)
	for _, id := range entryIDs {
		if senseID, ok := m.senseIDMap[id]; ok {
			result[id] = senseID
		}
	}
	return result, nil
}

func (m *mockRepo) GetPronunciationIPAsByEntryIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]map[string]bool, error) {
	m.logCall("GetPronunciationIPAsByEntryIDs")
	return map[uuid.UUID]map[string]bool{}, nil
}

func (m *mockRepo) UpsertDataSources(_ context.Context, _ []domain.RefDataSource) error {
	m.logCall("UpsertDataSources")
	if m.upsertDataSourcesErr != nil {
		return m.upsertDataSourcesErr
	}
	m.mu.Lock()
	m.dataSourcesUpserted = true
	m.mu.Unlock()
	return nil
}

func (m *mockRepo) ReplaceEntryContent(_ context.Context, _ uuid.UUID, _ []domain.RefSense, _ []domain.RefTranslation, _ []domain.RefExample) error {
	m.logCall("ReplaceEntryContent")
	return nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestPipeline_DryRunNoRepoWrites(t *testing.T) {
	repo := newMockRepo()
	cfg := Config{
		DryRun:    true,
		BatchSize: 100,
		TopN:      100,
	}

	p := NewPipeline(testLogger(), repo, cfg)
	err := p.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// In dry run, data sources are still registered, but no data writes happen.
	if !repo.dataSourcesUpserted {
		t.Error("expected data sources to be upserted even in dry run")
	}
	if repo.entriesInserted != 0 {
		t.Errorf("expected 0 entries inserted in dry run, got %d", repo.entriesInserted)
	}
	if repo.sensesInserted != 0 {
		t.Errorf("expected 0 senses inserted in dry run, got %d", repo.sensesInserted)
	}
	if repo.metadataUpdated != 0 {
		t.Errorf("expected 0 metadata updates in dry run, got %d", repo.metadataUpdated)
	}
}

func TestPipeline_ErrorIsolation(t *testing.T) {
	// Create Wiktionary JSONL with valid entries to trigger BulkInsertEntries.
	wiktData := `{"word":"hello","pos":"interjection","lang":"English","senses":[{"glosses":["greeting"]}]}` + "\n"
	tmpWikt := createTempFile(t, "wiktionary", wiktData)
	tmpNGSL := createTempFile(t, "ngsl", "word\nhello\n")
	tmpNAWL := createTempFile(t, "nawl", "word\nworld\n")
	tmpCMU := createTempFile(t, "cmu", "")
	tmpWordNet := createTempFile(t, "wordnet", `{"@graph":[]}`)
	tmpTatoeba := createTempFile(t, "tatoeba", "")

	repo := newMockRepo()
	// Make wiktionary's BulkInsertEntries fail.
	repo.bulkInsertEntriesErr = errors.New("wiktionary db error")
	// Set up known words for later phases.
	repo.normalizedTexts = map[string]bool{"hello": true}
	repo.entryIDMap = map[string]uuid.UUID{"hello": uuid.New()}

	cfg := Config{
		WiktionaryPath:     tmpWikt,
		NGSLPath:           tmpNGSL,
		NAWLPath:           tmpNAWL,
		CMUPath:            tmpCMU,
		WordNetPath:        tmpWordNet,
		TatoebaPath:        tmpTatoeba,
		BatchSize:          100,
		TopN:               100,
		MaxExamplesPerWord: 5,
	}

	p := NewPipeline(testLogger(), repo, cfg)
	err := p.Run(context.Background(), nil)

	// Pipeline should still complete (no fatal error), but report phase errors.
	if err != nil {
		t.Fatalf("pipeline should not return fatal error, got: %v", err)
	}

	// Wiktionary should have failed, but other phases should have been attempted.
	results := p.Results()
	if len(results) == 0 {
		t.Fatal("expected phase results to be recorded")
	}

	// Verify wiktionary phase recorded an error.
	wiktResult, ok := results["wiktionary"]
	if !ok {
		t.Fatal("expected wiktionary result")
	}
	if wiktResult.Err == nil {
		t.Error("expected wiktionary phase to have an error")
	}

	// Verify other phases were still attempted (they may have empty data, which is OK).
	for _, phase := range []string{"ngsl", "cmu", "wordnet", "tatoeba"} {
		if _, ok := results[phase]; !ok {
			t.Errorf("expected %s phase to be attempted despite wiktionary failure", phase)
		}
	}
}

func TestPipeline_PhaseFilter(t *testing.T) {
	tmpNGSL := createTempFile(t, "ngsl", "word\nhello\n")
	tmpNAWL := createTempFile(t, "nawl", "word\nworld\n")

	repo := newMockRepo()
	cfg := Config{
		NGSLPath:  tmpNGSL,
		NAWLPath:  tmpNAWL,
		BatchSize: 100,
		TopN:      100,
	}

	p := NewPipeline(testLogger(), repo, cfg)
	err := p.Run(context.Background(), []string{"ngsl"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	results := p.Results()

	// Only ngsl should be in results.
	if _, ok := results["ngsl"]; !ok {
		t.Error("expected ngsl phase to run")
	}
	if _, ok := results["wiktionary"]; ok {
		t.Error("wiktionary should NOT run when filter is ngsl only")
	}
	if _, ok := results["cmu"]; ok {
		t.Error("cmu should NOT run when filter is ngsl only")
	}
}

func TestPipeline_PhaseOrderingDataSourcesFirst(t *testing.T) {
	repo := newMockRepo()
	cfg := Config{
		DryRun:    true,
		BatchSize: 100,
		TopN:      100,
	}

	p := NewPipeline(testLogger(), repo, cfg)
	_ = p.Run(context.Background(), nil)

	// UpsertDataSources should be the first call.
	if len(repo.callLog) == 0 {
		t.Fatal("expected at least one repo call")
	}
	if repo.callLog[0] != "UpsertDataSources" {
		t.Errorf("expected first call to be UpsertDataSources, got %s", repo.callLog[0])
	}
}

func TestBatchProcess(t *testing.T) {
	items := make([]int, 7)
	for i := range items {
		items[i] = i
	}

	var batches [][]int
	total, err := batchProcess(items, 3, func(batch []int) (int, error) {
		batches = append(batches, batch)
		return len(batch), nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 7 {
		t.Errorf("expected total 7, got %d", total)
	}
	if len(batches) != 3 {
		t.Errorf("expected 3 batches, got %d", len(batches))
	}
	// First two batches should be size 3, last should be 1.
	if len(batches[0]) != 3 {
		t.Errorf("expected first batch size 3, got %d", len(batches[0]))
	}
	if len(batches[2]) != 1 {
		t.Errorf("expected last batch size 1, got %d", len(batches[2]))
	}
}

func TestBatchProcess_EmptySlice(t *testing.T) {
	total, err := batchProcess([]int{}, 10, func(batch []int) (int, error) {
		t.Fatal("should not be called for empty input")
		return 0, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0, got %d", total)
	}
}

func TestBatchProcess_ErrorStops(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6}
	callCount := 0
	_, err := batchProcess(items, 2, func(batch []int) (int, error) {
		callCount++
		if callCount == 2 {
			return 0, fmt.Errorf("batch error")
		}
		return len(batch), nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls before error, got %d", callCount)
	}
}

func TestBatchedLookup(t *testing.T) {
	repo := newMockRepo()
	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New()
	repo.entryIDMap = map[string]uuid.UUID{
		"apple":  id1,
		"banana": id2,
		"cherry": id3,
	}

	result, err := batchedLookup(context.Background(), repo, []string{"apple", "banana", "cherry"}, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 results, got %d", len(result))
	}
	if result["apple"] != id1 {
		t.Error("apple ID mismatch")
	}
}

func TestPipeline_CoverageRecording(t *testing.T) {
	// When a phase runs successfully with data, coverage should be recorded.
	tmpNGSL := createTempFile(t, "ngsl", "word\nhello\nworld\n")
	tmpNAWL := createTempFile(t, "nawl", "word\ntest\n")

	repo := newMockRepo()
	cfg := Config{
		NGSLPath:  tmpNGSL,
		NAWLPath:  tmpNAWL,
		BatchSize: 100,
		TopN:      100,
	}

	p := NewPipeline(testLogger(), repo, cfg)
	err := p.Run(context.Background(), []string{"ngsl"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// NGSL phase should update metadata.
	if repo.metadataUpdated == 0 {
		t.Error("expected metadata to be updated for NGSL phase")
	}
}

// createTempFile creates a temporary file with the given content for testing.
func createTempFile(t *testing.T, prefix, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), prefix+"_*")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if content != "" {
		if _, err := f.WriteString(content); err != nil {
			t.Fatalf("write temp file: %v", err)
		}
	}
	f.Close()
	return f.Name()
}
