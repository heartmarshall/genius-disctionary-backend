package seeder

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"time"

	"github.com/google/uuid"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/app/seeder/cmu"
	"github.com/heartmarshall/myenglish-backend/internal/app/seeder/ngsl"
	"github.com/heartmarshall/myenglish-backend/internal/app/seeder/tatoeba"
	"github.com/heartmarshall/myenglish-backend/internal/app/seeder/wiktionary"
	"github.com/heartmarshall/myenglish-backend/internal/app/seeder/wordnet"
)

// allPhases defines the canonical execution order.
var allPhases = []string{"wiktionary", "ngsl", "cmu", "wordnet", "tatoeba"}

// knownDataSources returns the 8 predefined data sources.
func knownDataSources() []domain.RefDataSource {
	now := time.Now()
	return []domain.RefDataSource{
		{Slug: "freedict", Name: "Free Dictionary API", Description: "FreeDictionary API definitions", SourceType: "definitions", IsActive: true, DatasetVersion: "v1", CreatedAt: now, UpdatedAt: now},
		{Slug: "translate", Name: "Google Translate", Description: "Google Translate translations", SourceType: "translations", IsActive: true, DatasetVersion: "v1", CreatedAt: now, UpdatedAt: now},
		{Slug: "wiktionary", Name: "Wiktionary (Kaikki)", Description: "Kaikki JSONL dump of English Wiktionary", SourceType: "definitions", IsActive: true, DatasetVersion: "v1", CreatedAt: now, UpdatedAt: now},
		{Slug: "ngsl", Name: "New General Service List", Description: "NGSL frequency-ranked core vocabulary", SourceType: "metadata", IsActive: true, DatasetVersion: "v1", CreatedAt: now, UpdatedAt: now},
		{Slug: "nawl", Name: "New Academic Word List", Description: "NAWL academic vocabulary list", SourceType: "metadata", IsActive: true, DatasetVersion: "v1", CreatedAt: now, UpdatedAt: now},
		{Slug: "cmu", Name: "CMU Pronouncing Dictionary", Description: "ARPAbet-to-IPA pronunciation data", SourceType: "pronunciations", IsActive: true, DatasetVersion: "v1", CreatedAt: now, UpdatedAt: now},
		{Slug: "wordnet", Name: "Open English WordNet", Description: "GWN-LMF semantic relations", SourceType: "relations", IsActive: true, DatasetVersion: "v1", CreatedAt: now, UpdatedAt: now},
		{Slug: "tatoeba", Name: "Tatoeba", Description: "EN-RU sentence pairs", SourceType: "examples", IsActive: true, DatasetVersion: "v1", CreatedAt: now, UpdatedAt: now},
	}
}

// PhaseResult holds the outcome of a single pipeline phase.
type PhaseResult struct {
	Inserted int
	Updated  int
	Skipped  int
	Errors   int
	Duration time.Duration
	Err      error
}

// Pipeline orchestrates the 5-phase seeding process.
type Pipeline struct {
	log     *slog.Logger
	repo    RefEntryBulkRepo
	cfg     Config
	results map[string]PhaseResult
}

// NewPipeline creates a new Pipeline.
func NewPipeline(log *slog.Logger, repo RefEntryBulkRepo, cfg Config) *Pipeline {
	return &Pipeline{
		log:     log,
		repo:    repo,
		cfg:     cfg,
		results: make(map[string]PhaseResult),
	}
}

// Results returns phase results after Run completes.
func (p *Pipeline) Results() map[string]PhaseResult {
	return p.results
}

// HasErrors returns true if any phase recorded errors.
func (p *Pipeline) HasErrors() bool {
	for _, r := range p.results {
		if r.Err != nil || r.Errors > 0 {
			return true
		}
	}
	return false
}

// Run executes the pipeline. If phases is non-empty, only the listed phases run.
func (p *Pipeline) Run(ctx context.Context, phases []string) error {
	// Step 1: Register data sources.
	if err := p.repo.UpsertDataSources(ctx, knownDataSources()); err != nil {
		return fmt.Errorf("upsert data sources: %w", err)
	}

	// Step 2: Determine which phases to run.
	toRun := allPhases
	if len(phases) > 0 {
		filter := make(map[string]bool, len(phases))
		for _, ph := range phases {
			filter[ph] = true
		}
		var filtered []string
		for _, ph := range allPhases {
			if filter[ph] {
				filtered = append(filtered, ph)
			}
		}
		toRun = filtered
	}

	// Step 3: Execute phases in order.
	for _, phase := range toRun {
		start := time.Now()
		p.log.Info("starting phase", slog.String("phase", phase))

		var result PhaseResult
		switch phase {
		case "wiktionary":
			result = p.runWiktionary(ctx)
		case "ngsl":
			result = p.runNGSL(ctx)
		case "cmu":
			result = p.runCMU(ctx)
		case "wordnet":
			result = p.runWordNet(ctx)
		case "tatoeba":
			result = p.runTatoeba(ctx)
		}
		result.Duration = time.Since(start)
		p.results[phase] = result

		if result.Err != nil {
			p.log.Warn("phase failed",
				slog.String("phase", phase),
				slog.String("error", result.Err.Error()),
				slog.Duration("duration", result.Duration),
			)
		} else {
			p.log.Info("phase completed",
				slog.String("phase", phase),
				slog.Int("inserted", result.Inserted),
				slog.Int("updated", result.Updated),
				slog.Int("skipped", result.Skipped),
				slog.Duration("duration", result.Duration),
			)
		}
	}

	// Step 4: Summary log.
	p.log.Info("pipeline completed", slog.Int("phases_run", len(toRun)))
	return nil
}

// runWiktionary parses and inserts Wiktionary entries in parent→child order.
func (p *Pipeline) runWiktionary(ctx context.Context) PhaseResult {
	if p.cfg.WiktionaryPath == "" {
		return PhaseResult{Skipped: 1, Err: fmt.Errorf("wiktionary path not configured")}
	}

	// Parse NGSL/NAWL first for core words (if available).
	var coreWords map[string]bool
	if p.cfg.NGSLPath != "" && p.cfg.NAWLPath != "" {
		_, cw, err := ngsl.Parse(p.cfg.NGSLPath, p.cfg.NAWLPath)
		if err != nil {
			p.log.Warn("could not parse NGSL/NAWL for core words", slog.String("error", err.Error()))
		} else {
			coreWords = cw
		}
	}

	entries, stats, err := wiktionary.Parse(p.cfg.WiktionaryPath, coreWords, p.cfg.TopN)
	if err != nil {
		return PhaseResult{Err: fmt.Errorf("parse wiktionary: %w", err)}
	}
	p.log.Info("wiktionary parsed", slog.Int("entries", len(entries)), slog.Int("total_lines", stats.TotalLines))

	if p.cfg.DryRun {
		return PhaseResult{Skipped: len(entries)}
	}

	domainData := wiktionary.ToDomainEntries(entries)

	var result PhaseResult

	// Insert in parent→child order: entries → senses → translations → examples → pronunciations.
	inserted, err := batchProcess(domainData.Entries, p.cfg.BatchSize, func(batch []domain.RefEntry) (int, error) {
		return p.repo.BulkInsertEntries(ctx, batch)
	})
	if err != nil {
		return PhaseResult{Err: fmt.Errorf("insert entries: %w", err)}
	}
	result.Inserted += inserted

	inserted, err = batchProcess(domainData.Senses, p.cfg.BatchSize, func(batch []domain.RefSense) (int, error) {
		return p.repo.BulkInsertSenses(ctx, batch)
	})
	if err != nil {
		return PhaseResult{Err: fmt.Errorf("insert senses: %w", err)}
	}
	result.Inserted += inserted

	inserted, err = batchProcess(domainData.Translations, p.cfg.BatchSize, func(batch []domain.RefTranslation) (int, error) {
		return p.repo.BulkInsertTranslations(ctx, batch)
	})
	if err != nil {
		return PhaseResult{Err: fmt.Errorf("insert translations: %w", err)}
	}
	result.Inserted += inserted

	inserted, err = batchProcess(domainData.Examples, p.cfg.BatchSize, func(batch []domain.RefExample) (int, error) {
		return p.repo.BulkInsertExamples(ctx, batch)
	})
	if err != nil {
		return PhaseResult{Err: fmt.Errorf("insert examples: %w", err)}
	}
	result.Inserted += inserted

	inserted, err = batchProcess(domainData.Pronunciations, p.cfg.BatchSize, func(batch []domain.RefPronunciation) (int, error) {
		return p.repo.BulkInsertPronunciations(ctx, batch)
	})
	if err != nil {
		return PhaseResult{Err: fmt.Errorf("insert pronunciations: %w", err)}
	}
	result.Inserted += inserted

	// Record coverage for wiktionary.
	coverage := buildCoverage(domainData.Entries, "wiktionary", "fetched")
	if _, err := batchProcess(coverage, p.cfg.BatchSize, func(batch []domain.RefEntrySourceCoverage) (int, error) {
		return p.repo.BulkInsertCoverage(ctx, batch)
	}); err != nil {
		p.log.Warn("wiktionary coverage insert failed", slog.String("error", err.Error()))
	}

	return result
}

// runNGSL parses NGSL/NAWL and updates entry metadata.
func (p *Pipeline) runNGSL(ctx context.Context) PhaseResult {
	if p.cfg.NGSLPath == "" || p.cfg.NAWLPath == "" {
		return PhaseResult{Skipped: 1, Err: fmt.Errorf("ngsl/nawl paths not configured")}
	}

	updates, _, err := ngsl.Parse(p.cfg.NGSLPath, p.cfg.NAWLPath)
	if err != nil {
		return PhaseResult{Err: fmt.Errorf("parse ngsl/nawl: %w", err)}
	}
	p.log.Info("ngsl/nawl parsed", slog.Int("updates", len(updates)))

	if p.cfg.DryRun {
		return PhaseResult{Skipped: len(updates)}
	}

	updated, err := batchProcess(updates, p.cfg.BatchSize, func(batch []domain.EntryMetadataUpdate) (int, error) {
		return p.repo.BulkUpdateEntryMetadata(ctx, batch)
	})
	if err != nil {
		return PhaseResult{Err: fmt.Errorf("update metadata: %w", err)}
	}

	return PhaseResult{Updated: updated}
}

// runCMU parses CMU dict and inserts pronunciations for known entries.
func (p *Pipeline) runCMU(ctx context.Context) PhaseResult {
	if p.cfg.CMUPath == "" {
		return PhaseResult{Skipped: 1, Err: fmt.Errorf("cmu path not configured")}
	}

	parsed, err := cmu.Parse(p.cfg.CMUPath)
	if err != nil {
		return PhaseResult{Err: fmt.Errorf("parse cmu: %w", err)}
	}
	p.log.Info("cmu parsed", slog.Int("unique_words", parsed.Stats.UniqueWords))

	if p.cfg.DryRun {
		return PhaseResult{Skipped: parsed.Stats.UniqueWords}
	}

	// Resolve word→entryID.
	words := make([]string, 0, len(parsed.Pronunciations))
	for w := range parsed.Pronunciations {
		words = append(words, w)
	}

	entryIDMap, err := batchedLookup(ctx, p.repo, words, p.cfg.BatchSize)
	if err != nil {
		return PhaseResult{Err: fmt.Errorf("lookup entry IDs: %w", err)}
	}

	pronunciations := parsed.ToDomainPronunciations(entryIDMap)

	// Filter out pronunciations that already exist from Wiktionary (same IPA for same entry).
	entryIDs := make([]uuid.UUID, 0, len(entryIDMap))
	for _, id := range entryIDMap {
		entryIDs = append(entryIDs, id)
	}
	existingIPAs, err := p.repo.GetPronunciationIPAsByEntryIDs(ctx, entryIDs)
	if err != nil {
		return PhaseResult{Err: fmt.Errorf("get existing pronunciations: %w", err)}
	}

	var filtered []domain.RefPronunciation
	skipped := 0
	for _, pr := range pronunciations {
		ipa := ""
		if pr.Transcription != nil {
			ipa = *pr.Transcription
		}
		if existing, ok := existingIPAs[pr.RefEntryID]; ok && existing[ipa] {
			skipped++
			continue
		}
		filtered = append(filtered, pr)
	}
	if skipped > 0 {
		p.log.Info("cmu dedup: skipped pronunciations already present from wiktionary", slog.Int("skipped", skipped))
	}

	inserted, err := batchProcess(filtered, p.cfg.BatchSize, func(batch []domain.RefPronunciation) (int, error) {
		return p.repo.BulkInsertPronunciations(ctx, batch)
	})
	if err != nil {
		return PhaseResult{Err: fmt.Errorf("insert pronunciations: %w", err)}
	}

	// Coverage: "fetched" for words with data, "no_data" for words in CMU but not in our DB.
	var coverage []domain.RefEntrySourceCoverage
	now := time.Now()
	for word, entryID := range entryIDMap {
		if _, ok := parsed.Pronunciations[word]; ok {
			coverage = append(coverage, domain.RefEntrySourceCoverage{
				RefEntryID:     entryID,
				SourceSlug:     "cmu",
				Status:         "fetched",
				DatasetVersion: "v1",
				FetchedAt:      now,
			})
		}
	}
	if _, err := batchProcess(coverage, p.cfg.BatchSize, func(batch []domain.RefEntrySourceCoverage) (int, error) {
		return p.repo.BulkInsertCoverage(ctx, batch)
	}); err != nil {
		p.log.Warn("cmu coverage insert failed", slog.String("error", err.Error()))
	}

	return PhaseResult{Inserted: inserted}
}

// runWordNet parses WordNet and inserts word relations.
func (p *Pipeline) runWordNet(ctx context.Context) PhaseResult {
	if p.cfg.WordNetPath == "" {
		return PhaseResult{Skipped: 1, Err: fmt.Errorf("wordnet path not configured")}
	}

	// Get all known words from DB.
	knownWords, err := p.repo.GetAllNormalizedTexts(ctx)
	if err != nil {
		return PhaseResult{Err: fmt.Errorf("get known words: %w", err)}
	}

	parsed, err := wordnet.Parse(p.cfg.WordNetPath, knownWords)
	if err != nil {
		return PhaseResult{Err: fmt.Errorf("parse wordnet: %w", err)}
	}
	p.log.Info("wordnet parsed", slog.Int("relations", parsed.Stats.TotalRelations))

	if p.cfg.DryRun {
		return PhaseResult{Skipped: parsed.Stats.TotalRelations}
	}

	// Resolve words to entry IDs.
	wordSet := make(map[string]bool)
	for _, rel := range parsed.Relations {
		wordSet[rel.SourceWord] = true
		wordSet[rel.TargetWord] = true
	}
	words := make([]string, 0, len(wordSet))
	for w := range wordSet {
		words = append(words, w)
	}

	entryIDMap, err := batchedLookup(ctx, p.repo, words, p.cfg.BatchSize)
	if err != nil {
		return PhaseResult{Err: fmt.Errorf("lookup entry IDs: %w", err)}
	}

	relations := parsed.ToDomainRelations(entryIDMap)

	inserted, err := batchProcess(relations, p.cfg.BatchSize, func(batch []domain.RefWordRelation) (int, error) {
		return p.repo.BulkInsertRelations(ctx, batch)
	})
	if err != nil {
		return PhaseResult{Err: fmt.Errorf("insert relations: %w", err)}
	}

	return PhaseResult{Inserted: inserted}
}

// runTatoeba parses Tatoeba and inserts examples for known entries.
func (p *Pipeline) runTatoeba(ctx context.Context) PhaseResult {
	if p.cfg.TatoebaPath == "" {
		return PhaseResult{Skipped: 1, Err: fmt.Errorf("tatoeba path not configured")}
	}

	// Get all known words from DB.
	knownWords, err := p.repo.GetAllNormalizedTexts(ctx)
	if err != nil {
		return PhaseResult{Err: fmt.Errorf("get known words: %w", err)}
	}

	parsed, err := tatoeba.Parse(p.cfg.TatoebaPath, knownWords, p.cfg.MaxExamplesPerWord)
	if err != nil {
		return PhaseResult{Err: fmt.Errorf("parse tatoeba: %w", err)}
	}
	p.log.Info("tatoeba parsed", slog.Int("matched_words", parsed.Stats.MatchedWords))

	if p.cfg.DryRun {
		return PhaseResult{Skipped: parsed.Stats.TotalPairs}
	}

	// Resolve words to entry IDs.
	words := make([]string, 0, len(parsed.Sentences))
	for w := range parsed.Sentences {
		words = append(words, w)
	}

	entryIDMap, err := batchedLookup(ctx, p.repo, words, p.cfg.BatchSize)
	if err != nil {
		return PhaseResult{Err: fmt.Errorf("lookup entry IDs: %w", err)}
	}

	// Get first sense IDs for entries (examples need sense FK).
	entryIDs := make([]uuid.UUID, 0, len(entryIDMap))
	for _, id := range entryIDMap {
		entryIDs = append(entryIDs, id)
	}

	senseIDMap, err := p.repo.GetFirstSenseIDsByEntryIDs(ctx, entryIDs)
	if err != nil {
		return PhaseResult{Err: fmt.Errorf("lookup sense IDs: %w", err)}
	}

	examples := parsed.ToDomainExamples(entryIDMap, senseIDMap)

	inserted, err := batchProcess(examples, p.cfg.BatchSize, func(batch []domain.RefExample) (int, error) {
		return p.repo.BulkInsertExamples(ctx, batch)
	})
	if err != nil {
		return PhaseResult{Err: fmt.Errorf("insert examples: %w", err)}
	}

	return PhaseResult{Inserted: inserted}
}

// batchProcess splits items into batches and processes each via fn.
func batchProcess[T any](items []T, batchSize int, fn func([]T) (int, error)) (int, error) {
	if len(items) == 0 {
		return 0, nil
	}
	if batchSize <= 0 {
		batchSize = 500
	}

	total := 0
	for i := 0; i < len(items); i += batchSize {
		end := min(i+batchSize, len(items))
		n, err := fn(items[i:end])
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

// batchedLookup splits a large text slice into chunks and calls GetEntryIDsByNormalizedTexts.
func batchedLookup(ctx context.Context, repo RefEntryBulkRepo, texts []string, batchSize int) (map[string]uuid.UUID, error) {
	if len(texts) == 0 {
		return make(map[string]uuid.UUID), nil
	}
	if batchSize <= 0 {
		batchSize = 500
	}

	result := make(map[string]uuid.UUID, len(texts))
	for i := 0; i < len(texts); i += batchSize {
		end := min(i+batchSize, len(texts))
		batch, err := repo.GetEntryIDsByNormalizedTexts(ctx, texts[i:end])
		if err != nil {
			return nil, err
		}
		maps.Copy(result, batch)
	}
	return result, nil
}

// buildCoverage creates coverage records for inserted entries.
func buildCoverage(entries []domain.RefEntry, sourceSlug, status string) []domain.RefEntrySourceCoverage {
	now := time.Now()
	coverage := make([]domain.RefEntrySourceCoverage, len(entries))
	for i, e := range entries {
		coverage[i] = domain.RefEntrySourceCoverage{
			RefEntryID:     e.ID,
			SourceSlug:     sourceSlug,
			Status:         status,
			DatasetVersion: "v1",
			FetchedAt:      now,
		}
	}
	return coverage
}
