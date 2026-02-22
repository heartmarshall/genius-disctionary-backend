//go:build integration

package seeder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"testing"
	"time"

	gqlhandler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	postgres "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/audit"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/card"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/entry"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/example"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/image"
	inboxrepo "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/inbox"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/pronunciation"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/refentry"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/reviewlog"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/sense"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/session"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/testhelper"
	topicrepo "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/topic"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/translation"
	userrepo "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/user"
	"github.com/heartmarshall/myenglish-backend/internal/config"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/provider"
	"github.com/heartmarshall/myenglish-backend/internal/service/content"
	"github.com/heartmarshall/myenglish-backend/internal/service/dictionary"
	inboxsvc "github.com/heartmarshall/myenglish-backend/internal/service/inbox"
	"github.com/heartmarshall/myenglish-backend/internal/service/refcatalog"
	"github.com/heartmarshall/myenglish-backend/internal/service/study"
	topicsvc "github.com/heartmarshall/myenglish-backend/internal/service/topic"
	usersvc "github.com/heartmarshall/myenglish-backend/internal/service/user"
	gqlpkg "github.com/heartmarshall/myenglish-backend/internal/transport/graphql"
	"github.com/heartmarshall/myenglish-backend/internal/transport/graphql/generated"
	"github.com/heartmarshall/myenglish-backend/internal/transport/graphql/resolver"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
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

// ---------------------------------------------------------------------------
// Test 8 — GraphQL resolvers for seeded data
// ---------------------------------------------------------------------------

// graphqlQuery sends a GraphQL POST request and returns the decoded response.
func graphqlQuery(t *testing.T, client *http.Client, url, query string, variables map[string]any) map[string]any {
	t.Helper()
	body := map[string]any{"query": query, "variables": variables}
	jsonBody, err := json.Marshal(body)
	require.NoError(t, err)

	resp, err := client.Post(url+"/query", "application/json", bytes.NewReader(jsonBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	var result map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	return result
}

// requireNoGQLErrors asserts that the GraphQL response has no errors.
func requireNoGQLErrors(t *testing.T, result map[string]any) {
	t.Helper()
	if errs, ok := result["errors"]; ok && errs != nil {
		t.Fatalf("unexpected GraphQL errors: %v", errs)
	}
}

// fakeAuthMiddleware injects a fixed user ID into every request context,
// so that resolvers requiring authentication (e.g. searchCatalog) succeed.
func fakeAuthMiddleware(userID uuid.UUID) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := ctxutil.WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// setupGraphQLServer creates a minimal HTTP test server with a GraphQL handler
// backed by real services and repositories against the given database pool.
func setupGraphQLServer(t *testing.T, pool *pgxpool.Pool, txm *postgres.TxManager) *httptest.Server {
	t.Helper()
	logger := integrationLogger()

	// Repositories.
	auditRepo := audit.New(pool)
	cardRepo := card.New(pool)
	entryRepo := entry.New(pool)
	exampleRepo := example.New(pool, txm)
	imageRepo := image.New(pool)
	inboxRepo := inboxrepo.New(pool)
	pronunciationRepo := pronunciation.New(pool)
	refentryRepo := refentry.New(pool, txm)
	reviewlogRepo := reviewlog.New(pool)
	senseRepo := sense.New(pool, txm)
	sessionRepo := session.New(pool)
	topicRepo := topicrepo.New(pool)
	translationRepo := translation.New(pool, txm)
	userRepo := userrepo.New(pool)

	// Services.
	refCatalogService := refcatalog.NewService(logger, refentryRepo, txm, &failDictProvider{}, &noopTransProvider{})

	dictionaryService := dictionary.NewService(
		logger, entryRepo, senseRepo, translationRepo, exampleRepo,
		pronunciationRepo, imageRepo, cardRepo, auditRepo, txm,
		refCatalogService, config.DictionaryConfig{
			MaxEntriesPerUser: 10000,
			DefaultEaseFactor: 2.5,
		},
	)

	contentService := content.NewService(
		logger, entryRepo, senseRepo, translationRepo, exampleRepo,
		imageRepo, auditRepo, txm,
	)

	srsConfig := domain.SRSConfig{
		DefaultEaseFactor:    2.5,
		MinEaseFactor:        1.3,
		MaxIntervalDays:      365,
		GraduatingInterval:   1,
		LearningSteps:        []time.Duration{time.Minute, 10 * time.Minute},
		NewCardsPerDay:       20,
		ReviewsPerDay:        200,
		EasyInterval:         4,
		RelearningSteps:      []time.Duration{10 * time.Minute},
		IntervalModifier:     1.0,
		HardIntervalModifier: 1.2,
		EasyBonus:            1.3,
		LapseNewInterval:     0.0,
		UndoWindowMinutes:    10,
	}

	studyService := study.NewService(
		logger, cardRepo, reviewlogRepo, sessionRepo, entryRepo,
		senseRepo, userRepo, auditRepo, txm, srsConfig,
	)

	topicService := topicsvc.NewService(logger, topicRepo, entryRepo, auditRepo, txm)
	inboxService := inboxsvc.NewService(logger, inboxRepo)
	userService := usersvc.NewService(logger, userRepo, userRepo, auditRepo, txm)

	// Resolver + GraphQL handler.
	res := resolver.NewResolver(
		logger, dictionaryService, contentService, studyService,
		topicService, inboxService, userService, refCatalogService,
	)

	schema := generated.NewExecutableSchema(generated.Config{Resolvers: res})
	gqlSrv := gqlhandler.NewDefaultServer(schema)
	gqlSrv.SetErrorPresenter(gqlpkg.NewErrorPresenter(logger))

	// Wrap with fake auth middleware so searchCatalog gets a user ID.
	fakeUserID := uuid.New()
	handler := fakeAuthMiddleware(fakeUserID)(gqlSrv)

	mux := http.NewServeMux()
	mux.Handle("POST /query", handler)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestIntegration_GraphQL_Resolvers(t *testing.T) {
	p, _, pool, txm := setupPipeline(t)
	cleanSeederData(t, pool)
	ctx := context.Background()

	// Seed the database.
	err := p.Run(ctx, nil)
	require.NoError(t, err, "pipeline should complete without error")

	// Set up the GraphQL server.
	srv := setupGraphQLServer(t, pool, txm)
	client := srv.Client()

	// ---------------------------------------------------------------
	// Sub-test 1: refDataSources returns 8 sources.
	// ---------------------------------------------------------------
	t.Run("refDataSources", func(t *testing.T) {
		query := `query { refDataSources { slug name sourceType isActive } }`
		result := graphqlQuery(t, client, srv.URL, query, nil)
		requireNoGQLErrors(t, result)

		data, ok := result["data"].(map[string]any)
		require.True(t, ok, "expected data object")

		sources, ok := data["refDataSources"].([]any)
		require.True(t, ok, "expected refDataSources array")
		require.Len(t, sources, 8, "expected 8 data sources")

		// Collect slugs.
		slugs := make([]string, 0, len(sources))
		for _, s := range sources {
			src, ok := s.(map[string]any)
			require.True(t, ok)
			slug, ok := src["slug"].(string)
			require.True(t, ok)
			slugs = append(slugs, slug)

			// Verify isActive is true for all sources.
			isActive, ok := src["isActive"].(bool)
			require.True(t, ok)
			assert.True(t, isActive, "source %q should be active", slug)
		}

		sort.Strings(slugs)
		expectedSlugs := []string{"cmu", "freedict", "nawl", "ngsl", "tatoeba", "translate", "wiktionary", "wordnet"}
		assert.Equal(t, expectedSlugs, slugs, "data source slugs should match")
	})

	// ---------------------------------------------------------------
	// Sub-test 2: searchCatalog returns correct data for "house".
	// ---------------------------------------------------------------
	t.Run("searchCatalog", func(t *testing.T) {
		query := `query { searchCatalog(query: "house", limit: 1) { id text frequencyRank cefrLevel isCoreLexicon } }`
		result := graphqlQuery(t, client, srv.URL, query, nil)
		requireNoGQLErrors(t, result)

		data, ok := result["data"].(map[string]any)
		require.True(t, ok, "expected data object")

		entries, ok := data["searchCatalog"].([]any)
		require.True(t, ok, "expected searchCatalog array")
		require.NotEmpty(t, entries, "searchCatalog should return at least 1 result")

		first, ok := entries[0].(map[string]any)
		require.True(t, ok)

		// text should contain "house".
		text, ok := first["text"].(string)
		require.True(t, ok)
		assert.Contains(t, text, "house", "text should contain 'house'")

		// frequencyRank should be 2 (rank 2 in NGSL sample).
		freqRank, ok := first["frequencyRank"].(float64)
		require.True(t, ok, "frequencyRank should be a number")
		assert.Equal(t, float64(2), freqRank, "house should have frequency rank 2")

		// cefrLevel should be "A1".
		cefrLevel, ok := first["cefrLevel"].(string)
		require.True(t, ok, "cefrLevel should be a string")
		assert.Equal(t, "A1", cefrLevel, "house should have CEFR level A1")

		// isCoreLexicon should be true.
		isCore, ok := first["isCoreLexicon"].(bool)
		require.True(t, ok, "isCoreLexicon should be a bool")
		assert.True(t, isCore, "house should be core lexicon")
	})

	// ---------------------------------------------------------------
	// Sub-test 3: refEntryRelations returns WordNet relations.
	// ---------------------------------------------------------------
	t.Run("refEntryRelations", func(t *testing.T) {
		// Get the entry ID for "big" from the database.
		var bigID uuid.UUID
		err := pool.QueryRow(ctx,
			`SELECT id FROM ref_entries WHERE text_normalized = 'big'`,
		).Scan(&bigID)
		require.NoError(t, err, "should find 'big' in ref_entries")

		query := `query($id: UUID!) { refEntryRelations(entryId: $id) { id relationType sourceSlug } }`
		vars := map[string]any{"id": bigID.String()}
		result := graphqlQuery(t, client, srv.URL, query, vars)
		requireNoGQLErrors(t, result)

		data, ok := result["data"].(map[string]any)
		require.True(t, ok, "expected data object")

		relations, ok := data["refEntryRelations"].([]any)
		require.True(t, ok, "expected refEntryRelations array")
		require.NotEmpty(t, relations, "big should have at least one relation")

		// Verify there is an antonym relation.
		foundAntonym := false
		for _, r := range relations {
			rel, ok := r.(map[string]any)
			require.True(t, ok)
			if rel["relationType"] == "antonym" {
				foundAntonym = true
				// sourceSlug should be "wordnet".
				assert.Equal(t, "wordnet", rel["sourceSlug"], "antonym relation should come from wordnet")
			}
		}
		assert.True(t, foundAntonym, "big should have an antonym relation")
	})

	// ---------------------------------------------------------------
	// Sub-test 4: RefEntry.sourceCoverage returns per-source status.
	// ---------------------------------------------------------------
	t.Run("sourceCoverage", func(t *testing.T) {
		query := `query { searchCatalog(query: "house", limit: 1) { id text sourceCoverage { source { slug } status } } }`
		result := graphqlQuery(t, client, srv.URL, query, nil)
		requireNoGQLErrors(t, result)

		data, ok := result["data"].(map[string]any)
		require.True(t, ok, "expected data object")

		entries, ok := data["searchCatalog"].([]any)
		require.True(t, ok, "expected searchCatalog array")
		require.NotEmpty(t, entries, "searchCatalog should return at least 1 result")

		first, ok := entries[0].(map[string]any)
		require.True(t, ok)

		coverageList, ok := first["sourceCoverage"].([]any)
		require.True(t, ok, "expected sourceCoverage array")
		require.NotEmpty(t, coverageList, "house should have source coverage entries")

		// Build slug -> status map.
		coverageMap := make(map[string]string)
		for _, c := range coverageList {
			cov, ok := c.(map[string]any)
			require.True(t, ok)
			src, ok := cov["source"].(map[string]any)
			require.True(t, ok)
			slug, ok := src["slug"].(string)
			require.True(t, ok)
			status, ok := cov["status"].(string)
			require.True(t, ok)
			coverageMap[slug] = status
		}

		// "house" should have wiktionary coverage with "fetched" status.
		wiktStatus, ok := coverageMap["wiktionary"]
		assert.True(t, ok, "house should have wiktionary coverage")
		assert.Equal(t, "fetched", wiktStatus, "wiktionary coverage should be 'fetched'")

		// "house" should have CMU coverage with "fetched" status.
		cmuStatus, ok := coverageMap["cmu"]
		assert.True(t, ok, "house should have CMU coverage")
		assert.Equal(t, "fetched", cmuStatus, "CMU coverage should be 'fetched'")
	})
}
