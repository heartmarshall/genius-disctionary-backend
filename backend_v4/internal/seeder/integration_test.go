//go:build integration

package seeder

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	postgres "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/refentry"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/testhelper"
	"github.com/heartmarshall/myenglish-backend/internal/provider"
	"github.com/heartmarshall/myenglish-backend/internal/service/refcatalog"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func integrationLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func testConfig() Config {
	return Config{
		WiktionaryPath:     "testdata/wiktionary_sample.jsonl",
		NGSLPath:           "testdata/ngsl_sample.csv",
		NAWLPath:           "testdata/nawl_sample.csv",
		CMUPath:            "testdata/cmu_sample.dict",
		WordNetPath:        "testdata/wordnet_sample.json",
		TatoebaPath:        "testdata/tatoeba_sample.tsv",
		TopN:               100,
		BatchSize:          100,
		MaxExamplesPerWord: 5,
	}
}

func setupPipeline(t *testing.T) (*Pipeline, *refentry.Repo, *pgxpool.Pool, *postgres.TxManager) {
	t.Helper()
	pool := testhelper.SetupTestDB(t)
	txm := postgres.NewTxManager(pool)
	repo := refentry.New(pool, txm)
	cfg := testConfig()
	p := NewPipeline(integrationLogger(), repo, cfg)
	return p, repo, pool, txm
}

// cleanSeederData removes all data inserted by the seeder to ensure test isolation.
func cleanSeederData(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	// Delete in reverse FK order.
	tables := []string{
		"ref_entry_source_coverage",
		"ref_word_relations",
		"ref_examples",
		"ref_translations",
		"ref_pronunciations",
		"ref_senses",
		"ref_entries",
		"ref_data_sources",
	}
	for _, table := range tables {
		_, err := pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s", table))
		require.NoError(t, err, "clean table %s", table)
	}
}

// Mock providers for Test 2.

type failDictProvider struct{}

func (f *failDictProvider) FetchEntry(_ context.Context, word string) (*provider.DictionaryResult, error) {
	return nil, fmt.Errorf("should not be called: dict provider hit for %q", word)
}

type noopTransProvider struct{}

func (n *noopTransProvider) FetchTranslations(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Test 1 — Full pipeline run
// ---------------------------------------------------------------------------

func TestIntegration_FullPipelineRun(t *testing.T) {
	p, _, pool, _ := setupPipeline(t)
	cleanSeederData(t, pool)
	ctx := context.Background()

	err := p.Run(ctx, nil)
	require.NoError(t, err, "pipeline should complete without error")

	results := p.Results()
	require.NotEmpty(t, results, "pipeline should produce results")

	// All 5 phases should be present.
	for _, phase := range []string{"wiktionary", "ngsl", "cmu", "wordnet", "tatoeba"} {
		r, ok := results[phase]
		require.True(t, ok, "phase %q should be in results", phase)
		assert.NoError(t, r.Err, "phase %q should not have an error", phase)
	}

	// Wiktionary should have inserted entries.
	wiktResult := results["wiktionary"]
	assert.Greater(t, wiktResult.Inserted, 0, "wiktionary should have Inserted > 0")

	// Overall: at least one phase should have non-zero counts.
	totalInserted := 0
	totalUpdated := 0
	for _, r := range results {
		totalInserted += r.Inserted
		totalUpdated += r.Updated
	}
	assert.Greater(t, totalInserted+totalUpdated, 0, "pipeline should produce non-zero counts")
}

// ---------------------------------------------------------------------------
// Test 2 — GetOrFetchEntry returns seeded data
// ---------------------------------------------------------------------------

func TestIntegration_GetOrFetchEntry_ReturnsSeededData(t *testing.T) {
	p, repo, pool, txm := setupPipeline(t)
	cleanSeederData(t, pool)
	ctx := context.Background()

	// Run full pipeline to seed data.
	err := p.Run(ctx, nil)
	require.NoError(t, err)

	// Create refcatalog service with a mock dict provider that errors if called.
	logger := integrationLogger()
	svc := refcatalog.NewService(logger, repo, txm, &failDictProvider{}, &noopTransProvider{})

	// "house" should exist in the DB from the wiktionary seed.
	entry, err := svc.GetOrFetchEntry(ctx, "house")
	require.NoError(t, err, "GetOrFetchEntry should not error for seeded word")
	require.NotNil(t, entry, "entry should not be nil")

	// Verify entry fields.
	assert.Equal(t, "house", entry.TextNormalized)
	assert.NotEmpty(t, entry.Senses, "entry should have senses")

	// Verify senses come from wiktionary source.
	for _, sense := range entry.Senses {
		assert.Equal(t, "wiktionary", sense.SourceSlug, "senses should have source_slug=wiktionary")
	}

	// The mock dict provider was NOT called -- if it were, it would return an error
	// and the test above (require.NoError) would have failed.
}

// ---------------------------------------------------------------------------
// Test 3 — Idempotency
// ---------------------------------------------------------------------------

func TestIntegration_Idempotency(t *testing.T) {
	_, _, pool, _ := setupPipeline(t)
	cleanSeederData(t, pool)
	ctx := context.Background()

	// First run.
	cfg := testConfig()
	txm := postgres.NewTxManager(pool)
	repo := refentry.New(pool, txm)
	p1 := NewPipeline(integrationLogger(), repo, cfg)
	err := p1.Run(ctx, nil)
	require.NoError(t, err)

	firstResults := p1.Results()
	firstWiktInserted := firstResults["wiktionary"].Inserted
	require.Greater(t, firstWiktInserted, 0, "first run should insert wiktionary entries")

	// Second run -- same config, same data.
	p2 := NewPipeline(integrationLogger(), repo, cfg)
	err = p2.Run(ctx, nil)
	require.NoError(t, err)

	secondResults := p2.Results()
	secondWiktInserted := secondResults["wiktionary"].Inserted
	assert.Equal(t, 0, secondWiktInserted,
		"second run should insert 0 new wiktionary entries due to ON CONFLICT DO NOTHING")
}

// ---------------------------------------------------------------------------
// Test 4 — CEFR metadata enrichment
// ---------------------------------------------------------------------------

func TestIntegration_CEFRMetadataEnrichment(t *testing.T) {
	p, _, pool, _ := setupPipeline(t)
	cleanSeederData(t, pool)
	ctx := context.Background()

	err := p.Run(ctx, nil)
	require.NoError(t, err)

	// "time" is rank 1 in ngsl_sample.csv (first word after header).
	var frequencyRank *int
	var cefrLevel *string
	var isCoreLexicon *bool
	err = pool.QueryRow(ctx,
		`SELECT frequency_rank, cefr_level, is_core_lexicon
		 FROM ref_entries WHERE text_normalized = 'time'`,
	).Scan(&frequencyRank, &cefrLevel, &isCoreLexicon)
	require.NoError(t, err, "should find 'time' in ref_entries")

	require.NotNil(t, frequencyRank, "time should have a frequency_rank")
	assert.Equal(t, 1, *frequencyRank, "time should be rank 1 in NGSL")

	require.NotNil(t, cefrLevel, "time should have a cefr_level")
	assert.Equal(t, "A1", *cefrLevel, "time (rank 1) should be A1")

	require.NotNil(t, isCoreLexicon, "time should have is_core_lexicon set")
	assert.True(t, *isCoreLexicon, "time should be core lexicon")

	// Verify a word near the end of the NGSL list has the expected rank.
	// "warehouse" is the 15th word in ngsl_sample.csv (after header), so rank=15, cefr=A1 (<=500).
	var whRank *int
	var whCefr *string
	err = pool.QueryRow(ctx,
		`SELECT frequency_rank, cefr_level
		 FROM ref_entries WHERE text_normalized = 'warehouse'`,
	).Scan(&whRank, &whCefr)
	require.NoError(t, err, "should find 'warehouse' in ref_entries")
	require.NotNil(t, whRank, "warehouse should have frequency_rank")
	assert.Equal(t, 15, *whRank, "warehouse should be rank 15 in NGSL")
}

// ---------------------------------------------------------------------------
// Test 5 — Source coverage tracking
// ---------------------------------------------------------------------------

func TestIntegration_SourceCoverageTracking(t *testing.T) {
	p, _, pool, _ := setupPipeline(t)
	cleanSeederData(t, pool)
	ctx := context.Background()

	err := p.Run(ctx, nil)
	require.NoError(t, err)

	// Get the ref_entry ID for "house".
	var houseID uuid.UUID
	err = pool.QueryRow(ctx,
		`SELECT id FROM ref_entries WHERE text_normalized = 'house'`,
	).Scan(&houseID)
	require.NoError(t, err, "should find 'house' in ref_entries")

	// Query coverage for "house".
	rows, err := pool.Query(ctx,
		`SELECT source_slug, status FROM ref_entry_source_coverage WHERE ref_entry_id = $1`,
		houseID,
	)
	require.NoError(t, err)
	defer rows.Close()

	coverage := make(map[string]string)
	for rows.Next() {
		var slug, status string
		err := rows.Scan(&slug, &status)
		require.NoError(t, err)
		coverage[slug] = status
	}
	require.NoError(t, rows.Err())

	// "house" should have wiktionary coverage.
	wiktStatus, ok := coverage["wiktionary"]
	assert.True(t, ok, "house should have wiktionary coverage")
	assert.Equal(t, "fetched", wiktStatus, "wiktionary coverage status should be 'fetched'")

	// "house" should have CMU coverage (it's in cmu_sample.dict).
	cmuStatus, ok := coverage["cmu"]
	assert.True(t, ok, "house should have CMU coverage")
	assert.Equal(t, "fetched", cmuStatus, "CMU coverage status should be 'fetched'")
}

// ---------------------------------------------------------------------------
// Test 6 — Word relations
// ---------------------------------------------------------------------------

func TestIntegration_WordRelations(t *testing.T) {
	p, repo, pool, _ := setupPipeline(t)
	cleanSeederData(t, pool)
	ctx := context.Background()

	err := p.Run(ctx, nil)
	require.NoError(t, err)

	// Look up "big" entry ID -- big<->small is an antonym in the WordNet sample.
	var bigID uuid.UUID
	err = pool.QueryRow(ctx,
		`SELECT id FROM ref_entries WHERE text_normalized = 'big'`,
	).Scan(&bigID)
	require.NoError(t, err, "should find 'big' in ref_entries")

	// Get relations for "big".
	relations, err := repo.GetRelationsByEntryID(ctx, bigID)
	require.NoError(t, err)
	require.NotEmpty(t, relations, "big should have at least one relation")

	// Verify there's an antonym relation.
	foundAntonym := false
	for _, rel := range relations {
		if rel.RelationType == "antonym" {
			foundAntonym = true
			break
		}
	}
	assert.True(t, foundAntonym, "big should have an antonym relation (big<->small)")

	// Also check from the "small" side (bidirectional query).
	var smallID uuid.UUID
	err = pool.QueryRow(ctx,
		`SELECT id FROM ref_entries WHERE text_normalized = 'small'`,
	).Scan(&smallID)
	require.NoError(t, err)

	relationsSmall, err := repo.GetRelationsByEntryID(ctx, smallID)
	require.NoError(t, err)
	require.NotEmpty(t, relationsSmall, "small should have at least one relation")

	// Also check run<->walk synonym. Based on WordNet sample, run and walk share synset ewn-move-v-01.
	var runID uuid.UUID
	err = pool.QueryRow(ctx,
		`SELECT id FROM ref_entries WHERE text_normalized = 'run'`,
	).Scan(&runID)
	require.NoError(t, err)

	runRelations, err := repo.GetRelationsByEntryID(ctx, runID)
	require.NoError(t, err)
	require.NotEmpty(t, runRelations, "run should have at least one relation")

	foundSynonym := false
	for _, rel := range runRelations {
		if rel.RelationType == "synonym" {
			foundSynonym = true
			break
		}
	}
	assert.True(t, foundSynonym, "run should have a synonym relation (run<->walk)")
}

// ---------------------------------------------------------------------------
// Test 7 — Single-phase execution
// ---------------------------------------------------------------------------

func TestIntegration_SinglePhaseExecution(t *testing.T) {
	_, _, pool, _ := setupPipeline(t)
	cleanSeederData(t, pool)
	ctx := context.Background()

	cfg := testConfig()
	txm := postgres.NewTxManager(pool)
	repo := refentry.New(pool, txm)

	// First: full pipeline run.
	p1 := NewPipeline(integrationLogger(), repo, cfg)
	err := p1.Run(ctx, nil)
	require.NoError(t, err)

	// Verify NGSL metadata was applied in first run.
	var rank1 *int
	err = pool.QueryRow(ctx,
		`SELECT frequency_rank FROM ref_entries WHERE text_normalized = 'time'`,
	).Scan(&rank1)
	require.NoError(t, err)
	require.NotNil(t, rank1)
	assert.Equal(t, 1, *rank1, "time should have rank 1 after first run")

	// Second run: NGSL only.
	p2 := NewPipeline(integrationLogger(), repo, cfg)
	err = p2.Run(ctx, []string{"ngsl"})
	require.NoError(t, err)

	results2 := p2.Results()

	// NGSL phase should have results.
	ngslResult, ok := results2["ngsl"]
	require.True(t, ok, "ngsl phase should be in results")
	assert.Greater(t, ngslResult.Updated, 0, "ngsl phase should have Updated > 0")

	// Wiktionary should NOT be in the results (was not run).
	_, wiktOK := results2["wiktionary"]
	assert.False(t, wiktOK, "wiktionary should NOT be in results for single-phase run")

	// CMU should NOT be in the results either.
	_, cmuOK := results2["cmu"]
	assert.False(t, cmuOK, "cmu should NOT be in results for single-phase run")

	// Verify NGSL metadata still present after second run.
	var rank2 *int
	err = pool.QueryRow(ctx,
		`SELECT frequency_rank FROM ref_entries WHERE text_normalized = 'time'`,
	).Scan(&rank2)
	require.NoError(t, err)
	require.NotNil(t, rank2, "time should still have frequency_rank after ngsl-only run")
	assert.Equal(t, 1, *rank2, "time should still be rank 1")
}
