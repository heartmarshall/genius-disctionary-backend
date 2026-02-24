package refentry

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	postgres "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/refentry/sqlc"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// ---------------------------------------------------------------------------
// Batch insert methods (pgx.Batch API)
// ---------------------------------------------------------------------------

// BulkInsertEntries inserts ref_entries using pgx.Batch. Existing entries
// (by text_normalized) are skipped via ON CONFLICT DO NOTHING.
// Returns the number of actually inserted rows.
func (r *Repo) BulkInsertEntries(ctx context.Context, entries []domain.RefEntry) (int, error) {
	if len(entries) == 0 {
		return 0, nil
	}

	batch := &pgx.Batch{}
	for _, e := range entries {
		batch.Queue(
			`INSERT INTO ref_entries (id, text, text_normalized, frequency_rank, cefr_level, is_core_lexicon, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 ON CONFLICT (text_normalized) DO NOTHING`,
			e.ID, e.Text, e.TextNormalized,
			domain.IntPtrToInt32Ptr(e.FrequencyRank),
			e.CEFRLevel,
			e.IsCoreLexicon,
			e.CreatedAt,
		)
	}

	return r.sendBatchExec(ctx, batch)
}

// BulkInsertSenses inserts ref_senses using pgx.Batch.
// Existing senses (by id) are skipped via ON CONFLICT DO NOTHING.
func (r *Repo) BulkInsertSenses(ctx context.Context, senses []domain.RefSense) (int, error) {
	if len(senses) == 0 {
		return 0, nil
	}

	batch := &pgx.Batch{}
	for _, s := range senses {
		var pos *string
		if s.PartOfSpeech != nil {
			p := string(*s.PartOfSpeech)
			pos = &p
		}

		batch.Queue(
			`INSERT INTO ref_senses (id, ref_entry_id, definition, part_of_speech, cefr_level, notes, source_slug, position, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 ON CONFLICT (id) DO NOTHING`,
			s.ID, s.RefEntryID, s.Definition, pos, s.CEFRLevel, s.Notes, s.SourceSlug, s.Position, s.CreatedAt,
		)
	}

	return r.sendBatchExec(ctx, batch)
}

// BulkInsertTranslations inserts ref_translations using pgx.Batch.
// Existing translations (by id) are skipped via ON CONFLICT DO NOTHING.
func (r *Repo) BulkInsertTranslations(ctx context.Context, translations []domain.RefTranslation) (int, error) {
	if len(translations) == 0 {
		return 0, nil
	}

	batch := &pgx.Batch{}
	for _, tr := range translations {
		batch.Queue(
			`INSERT INTO ref_translations (id, ref_sense_id, text, source_slug, position)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (id) DO NOTHING`,
			tr.ID, tr.RefSenseID, tr.Text, tr.SourceSlug, tr.Position,
		)
	}

	return r.sendBatchExec(ctx, batch)
}

// BulkInsertExamples inserts ref_examples using pgx.Batch.
// Existing examples (by id) are skipped via ON CONFLICT DO NOTHING.
func (r *Repo) BulkInsertExamples(ctx context.Context, examples []domain.RefExample) (int, error) {
	if len(examples) == 0 {
		return 0, nil
	}

	batch := &pgx.Batch{}
	for _, ex := range examples {
		batch.Queue(
			`INSERT INTO ref_examples (id, ref_sense_id, sentence, translation, source_slug, position)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 ON CONFLICT (id) DO NOTHING`,
			ex.ID, ex.RefSenseID, ex.Sentence, ex.Translation, ex.SourceSlug, ex.Position,
		)
	}

	return r.sendBatchExec(ctx, batch)
}

// BulkInsertPronunciations inserts ref_pronunciations using pgx.Batch.
// Existing pronunciations (by id) are skipped via ON CONFLICT DO NOTHING.
func (r *Repo) BulkInsertPronunciations(ctx context.Context, pronunciations []domain.RefPronunciation) (int, error) {
	if len(pronunciations) == 0 {
		return 0, nil
	}

	batch := &pgx.Batch{}
	for _, p := range pronunciations {
		batch.Queue(
			`INSERT INTO ref_pronunciations (id, ref_entry_id, transcription, audio_url, region, source_slug)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 ON CONFLICT (id) DO NOTHING`,
			p.ID, p.RefEntryID, ptrStringToString(p.Transcription), p.AudioURL, p.Region, p.SourceSlug,
		)
	}

	return r.sendBatchExec(ctx, batch)
}

// BulkInsertRelations inserts ref_word_relations using pgx.Batch.
// Existing relations (by unique constraint) are skipped via ON CONFLICT DO NOTHING.
func (r *Repo) BulkInsertRelations(ctx context.Context, relations []domain.RefWordRelation) (int, error) {
	if len(relations) == 0 {
		return 0, nil
	}

	batch := &pgx.Batch{}
	for _, rel := range relations {
		batch.Queue(
			`INSERT INTO ref_word_relations (id, source_entry_id, target_entry_id, relation_type, source_slug, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 ON CONFLICT ON CONSTRAINT uq_ref_word_relations DO NOTHING`,
			rel.ID, rel.SourceEntryID, rel.TargetEntryID, rel.RelationType, rel.SourceSlug, rel.CreatedAt,
		)
	}

	return r.sendBatchExec(ctx, batch)
}

// BulkInsertCoverage inserts ref_entry_source_coverage using pgx.Batch.
// On conflict (same entry + source), updates status and fetched_at.
func (r *Repo) BulkInsertCoverage(ctx context.Context, coverage []domain.RefEntrySourceCoverage) (int, error) {
	if len(coverage) == 0 {
		return 0, nil
	}

	batch := &pgx.Batch{}
	for _, c := range coverage {
		batch.Queue(
			`INSERT INTO ref_entry_source_coverage (ref_entry_id, source_slug, status, dataset_version, fetched_at)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (ref_entry_id, source_slug) DO UPDATE
			 SET status = EXCLUDED.status, fetched_at = EXCLUDED.fetched_at`,
			c.RefEntryID, c.SourceSlug, c.Status, nilIfEmpty(c.DatasetVersion), c.FetchedAt,
		)
	}

	return r.sendBatchExec(ctx, batch)
}

// ---------------------------------------------------------------------------
// Replace methods (for LLM enrichment)
// ---------------------------------------------------------------------------

// ReplaceEntryContent replaces all senses, translations, and examples for a ref entry.
// Strategy: delete old child data, insert new. User senses with ref_sense_id pointing
// to deleted senses get SET NULL (by FK constraint ON DELETE SET NULL).
// This is acceptable because LLM data is strictly better quality.
func (r *Repo) ReplaceEntryContent(ctx context.Context, entryID uuid.UUID, senses []domain.RefSense, translations []domain.RefTranslation, examples []domain.RefExample) error {
	return r.txm.RunInTx(ctx, func(txCtx context.Context) error {
		q := postgres.QuerierFromCtx(txCtx, r.pool)

		// 1. Delete existing senses (cascades to ref_translations, ref_examples via FK).
		if _, err := q.Exec(txCtx, `DELETE FROM ref_senses WHERE ref_entry_id = $1`, entryID); err != nil {
			return fmt.Errorf("delete old senses: %w", err)
		}

		// 2. Insert new senses.
		if len(senses) > 0 {
			batch := &pgx.Batch{}
			for _, s := range senses {
				var pos *string
				if s.PartOfSpeech != nil {
					p := string(*s.PartOfSpeech)
					pos = &p
				}
				batch.Queue(
					`INSERT INTO ref_senses (id, ref_entry_id, definition, part_of_speech, cefr_level, notes, source_slug, position, created_at)
					 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
					s.ID, s.RefEntryID, s.Definition, pos, s.CEFRLevel, s.Notes, s.SourceSlug, s.Position, s.CreatedAt,
				)
			}
			results := q.SendBatch(txCtx, batch)
			for range senses {
				if _, err := results.Exec(); err != nil {
					results.Close()
					return fmt.Errorf("insert sense: %w", err)
				}
			}
			results.Close()
		}

		// 3. Insert new translations.
		if len(translations) > 0 {
			batch := &pgx.Batch{}
			for _, tr := range translations {
				batch.Queue(
					`INSERT INTO ref_translations (id, ref_sense_id, text, source_slug, position)
					 VALUES ($1, $2, $3, $4, $5)`,
					tr.ID, tr.RefSenseID, tr.Text, tr.SourceSlug, tr.Position,
				)
			}
			results := q.SendBatch(txCtx, batch)
			for range translations {
				if _, err := results.Exec(); err != nil {
					results.Close()
					return fmt.Errorf("insert translation: %w", err)
				}
			}
			results.Close()
		}

		// 4. Insert new examples.
		if len(examples) > 0 {
			batch := &pgx.Batch{}
			for _, ex := range examples {
				batch.Queue(
					`INSERT INTO ref_examples (id, ref_sense_id, sentence, translation, source_slug, position)
					 VALUES ($1, $2, $3, $4, $5, $6)`,
					ex.ID, ex.RefSenseID, ex.Sentence, ex.Translation, ex.SourceSlug, ex.Position,
				)
			}
			results := q.SendBatch(txCtx, batch)
			for range examples {
				if _, err := results.Exec(); err != nil {
					results.Close()
					return fmt.Errorf("insert example: %w", err)
				}
			}
			results.Close()
		}

		// 5. Upsert source coverage for 'llm'.
		now := time.Now()
		_, err := q.Exec(txCtx,
			`INSERT INTO ref_entry_source_coverage (ref_entry_id, source_slug, status, fetched_at)
			 VALUES ($1, 'llm', 'fetched', $2)
			 ON CONFLICT (ref_entry_id, source_slug) DO UPDATE
			 SET status = 'fetched', fetched_at = EXCLUDED.fetched_at`,
			entryID, now,
		)
		if err != nil {
			return fmt.Errorf("upsert coverage: %w", err)
		}

		return nil
	})
}

// ---------------------------------------------------------------------------
// Batch update methods
// ---------------------------------------------------------------------------

// BulkUpdateEntryMetadata updates frequency_rank, cefr_level, is_core_lexicon
// on ref_entries matched by text_normalized. Returns the number of updated rows.
func (r *Repo) BulkUpdateEntryMetadata(ctx context.Context, updates []domain.EntryMetadataUpdate) (int, error) {
	if len(updates) == 0 {
		return 0, nil
	}

	batch := &pgx.Batch{}
	for _, u := range updates {
		batch.Queue(
			`UPDATE ref_entries
			 SET frequency_rank = COALESCE($1, frequency_rank),
			     cefr_level = COALESCE($2, cefr_level),
			     is_core_lexicon = COALESCE($3, is_core_lexicon)
			 WHERE text_normalized = $4`,
			domain.IntPtrToInt32Ptr(u.FrequencyRank),
			u.CEFRLevel,
			u.IsCoreLexicon,
			u.TextNormalized,
		)
	}

	return r.sendBatchExec(ctx, batch)
}

// ---------------------------------------------------------------------------
// Lookup methods (for seeder pipeline)
// ---------------------------------------------------------------------------

// GetEntryIDsByNormalizedTexts returns a map of text_normalized → UUID
// for all matching entries.
func (r *Repo) GetEntryIDsByNormalizedTexts(ctx context.Context, texts []string) (map[string]uuid.UUID, error) {
	if len(texts) == 0 {
		return map[string]uuid.UUID{}, nil
	}

	q := postgres.QuerierFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT text_normalized, id FROM ref_entries WHERE text_normalized = ANY($1)`,
		texts,
	)
	if err != nil {
		return nil, fmt.Errorf("get entry IDs by texts: %w", err)
	}
	defer rows.Close()

	result := make(map[string]uuid.UUID, len(texts))
	for rows.Next() {
		var text string
		var id uuid.UUID
		if err := rows.Scan(&text, &id); err != nil {
			return nil, fmt.Errorf("scan entry ID: %w", err)
		}
		result[text] = id
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate entry IDs: %w", err)
	}

	return result, nil
}

// GetAllNormalizedTexts returns the full set of text_normalized values
// for all ref_entries. Used for filtering datasets (e.g., WordNet, Tatoeba).
func (r *Repo) GetAllNormalizedTexts(ctx context.Context) (map[string]bool, error) {
	q := postgres.QuerierFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx, `SELECT text_normalized FROM ref_entries`)
	if err != nil {
		return nil, fmt.Errorf("get all normalized texts: %w", err)
	}
	defer rows.Close()

	result := make(map[string]bool)
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return nil, fmt.Errorf("scan normalized text: %w", err)
		}
		result[text] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate normalized texts: %w", err)
	}

	return result, nil
}

// GetFirstSenseIDsByEntryIDs returns a map of entry_id → first sense UUID
// (the sense with the lowest position). Used by Tatoeba phase to attach examples.
func (r *Repo) GetFirstSenseIDsByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	if len(entryIDs) == 0 {
		return map[uuid.UUID]uuid.UUID{}, nil
	}

	q := postgres.QuerierFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT DISTINCT ON (ref_entry_id) ref_entry_id, id
		 FROM ref_senses
		 WHERE ref_entry_id = ANY($1)
		 ORDER BY ref_entry_id, position`,
		entryIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("get first sense IDs: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID]uuid.UUID, len(entryIDs))
	for rows.Next() {
		var entryID, senseID uuid.UUID
		if err := rows.Scan(&entryID, &senseID); err != nil {
			return nil, fmt.Errorf("scan first sense ID: %w", err)
		}
		result[entryID] = senseID
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate first sense IDs: %w", err)
	}

	return result, nil
}

// GetPronunciationIPAsByEntryIDs returns a map of entry_id → set of existing IPA transcriptions.
// Used by CMU phase to skip duplicates already inserted by Wiktionary.
func (r *Repo) GetPronunciationIPAsByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) (map[uuid.UUID]map[string]bool, error) {
	if len(entryIDs) == 0 {
		return map[uuid.UUID]map[string]bool{}, nil
	}

	q := postgres.QuerierFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT ref_entry_id, transcription FROM ref_pronunciations WHERE ref_entry_id = ANY($1)`,
		entryIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("get pronunciation IPAs: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID]map[string]bool, len(entryIDs))
	for rows.Next() {
		var entryID uuid.UUID
		var ipa string
		if err := rows.Scan(&entryID, &ipa); err != nil {
			return nil, fmt.Errorf("scan pronunciation IPA: %w", err)
		}
		if result[entryID] == nil {
			result[entryID] = make(map[string]bool)
		}
		result[entryID][ipa] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pronunciation IPAs: %w", err)
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// Registry method
// ---------------------------------------------------------------------------

// UpsertDataSources inserts or updates data source records.
// On conflict (slug), updates version and timestamp.
func (r *Repo) UpsertDataSources(ctx context.Context, sources []domain.RefDataSource) error {
	if len(sources) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, s := range sources {
		batch.Queue(
			`INSERT INTO ref_data_sources (slug, name, description, source_type, is_active, dataset_version, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 ON CONFLICT (slug) DO UPDATE
			 SET name = EXCLUDED.name,
			     description = EXCLUDED.description,
			     source_type = EXCLUDED.source_type,
			     is_active = EXCLUDED.is_active,
			     dataset_version = EXCLUDED.dataset_version,
			     updated_at = EXCLUDED.updated_at`,
			s.Slug, s.Name, nilIfEmpty(s.Description), s.SourceType, s.IsActive,
			nilIfEmpty(s.DatasetVersion), s.CreatedAt, s.UpdatedAt,
		)
	}

	q := postgres.QuerierFromCtx(ctx, r.pool)
	results := q.SendBatch(ctx, batch)
	defer results.Close()

	for range sources {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("upsert data source: %w", err)
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Read methods (for GraphQL resolvers)
// ---------------------------------------------------------------------------

// GetRelationsByEntryID returns all word relations where the entry is either
// source or target. Both directions are returned for symmetric queries.
func (r *Repo) GetRelationsByEntryID(ctx context.Context, entryID uuid.UUID) ([]domain.RefWordRelation, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	rows, err := q.GetRelationsByEntryID(ctx, entryID)
	if err != nil {
		return nil, fmt.Errorf("get relations by entry ID: %w", err)
	}

	relations := make([]domain.RefWordRelation, len(rows))
	for i, row := range rows {
		relations[i] = toDomainRefWordRelation(row)
	}

	return relations, nil
}

// GetAllDataSources returns all active data sources.
func (r *Repo) GetAllDataSources(ctx context.Context) ([]domain.RefDataSource, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	rows, err := q.GetAllDataSources(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all data sources: %w", err)
	}

	sources := make([]domain.RefDataSource, len(rows))
	for i, row := range rows {
		sources[i] = toDomainRefDataSource(row)
	}

	return sources, nil
}

// GetDataSourceBySlug returns a single data source by its slug.
// Returns domain.ErrNotFound if not found.
func (r *Repo) GetDataSourceBySlug(ctx context.Context, slug string) (*domain.RefDataSource, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.GetDataSourceBySlug(ctx, slug)
	if err != nil {
		return nil, mapError(err, "ref_data_source", uuid.Nil)
	}

	ds := toDomainRefDataSource(row)
	return &ds, nil
}

// GetCoverageByEntryID returns source coverage records for a single entry.
func (r *Repo) GetCoverageByEntryID(ctx context.Context, entryID uuid.UUID) ([]domain.RefEntrySourceCoverage, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	rows, err := q.GetCoverageByEntryID(ctx, entryID)
	if err != nil {
		return nil, fmt.Errorf("get coverage by entry ID: %w", err)
	}

	coverage := make([]domain.RefEntrySourceCoverage, len(rows))
	for i, row := range rows {
		coverage[i] = toDomainRefCoverage(row)
	}

	return coverage, nil
}

// GetCoverageByEntryIDs returns source coverage records for multiple entries,
// grouped by entry ID. Designed for DataLoader batch loading.
func (r *Repo) GetCoverageByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) (map[uuid.UUID][]domain.RefEntrySourceCoverage, error) {
	if len(entryIDs) == 0 {
		return map[uuid.UUID][]domain.RefEntrySourceCoverage{}, nil
	}

	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	rows, err := q.GetCoverageByEntryIDs(ctx, entryIDs)
	if err != nil {
		return nil, fmt.Errorf("get coverage by entry IDs: %w", err)
	}

	result := make(map[uuid.UUID][]domain.RefEntrySourceCoverage, len(entryIDs))
	for _, row := range rows {
		c := toDomainRefCoverage(row)
		result[c.RefEntryID] = append(result[c.RefEntryID], c)
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// Internal: batch execution helper
// ---------------------------------------------------------------------------

// sendBatchExec sends a pgx.Batch and counts affected rows from Exec results.
func (r *Repo) sendBatchExec(ctx context.Context, batch *pgx.Batch) (int, error) {
	q := postgres.QuerierFromCtx(ctx, r.pool)
	results := q.SendBatch(ctx, batch)
	defer results.Close()

	var inserted int
	for range batch.Len() {
		tag, err := results.Exec()
		if err != nil {
			return inserted, fmt.Errorf("batch exec: %w", err)
		}
		inserted += int(tag.RowsAffected())
	}

	return inserted, nil
}

// ---------------------------------------------------------------------------
// Domain converters for new types
// ---------------------------------------------------------------------------

func toDomainRefWordRelation(row sqlc.RefWordRelation) domain.RefWordRelation {
	r := domain.RefWordRelation{
		ID:            row.ID,
		SourceEntryID: row.SourceEntryID,
		TargetEntryID: row.TargetEntryID,
		RelationType:  row.RelationType,
		SourceSlug:    row.SourceSlug,
	}
	if row.CreatedAt != nil {
		r.CreatedAt = *row.CreatedAt
	}
	return r
}

func toDomainRefDataSource(row sqlc.RefDataSource) domain.RefDataSource {
	ds := domain.RefDataSource{
		Slug:           row.Slug,
		Name:           row.Name,
		Description:    pgTextToString(row.Description),
		SourceType:     row.SourceType,
		IsActive:       pgBoolValue(row.IsActive),
		DatasetVersion: pgTextToString(row.DatasetVersion),
	}
	if row.CreatedAt != nil {
		ds.CreatedAt = *row.CreatedAt
	}
	if row.UpdatedAt != nil {
		ds.UpdatedAt = *row.UpdatedAt
	}
	return ds
}

func toDomainRefCoverage(row sqlc.RefEntrySourceCoverage) domain.RefEntrySourceCoverage {
	c := domain.RefEntrySourceCoverage{
		RefEntryID:     row.RefEntryID,
		SourceSlug:     row.SourceSlug,
		Status:         row.Status,
		DatasetVersion: pgTextToString(row.DatasetVersion),
	}
	if row.FetchedAt != nil {
		c.FetchedAt = *row.FetchedAt
	}
	return c
}

// nilIfEmpty returns nil if s is empty, otherwise a pointer to s.
// Used for nullable TEXT columns where empty string means NULL.
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

