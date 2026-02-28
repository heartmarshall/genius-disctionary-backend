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

// ReplaceEntryContent upserts senses, translations, and examples for a ref entry.
// Strategy: position-based matching — existing rows are updated in-place (preserving
// UUIDs and FK references from user senses), new rows are inserted, excess old rows
// are deleted. This keeps user FK links valid so they see enriched data via COALESCE.
func (r *Repo) ReplaceEntryContent(ctx context.Context, entryID uuid.UUID, senses []domain.RefSense, translations []domain.RefTranslation, examples []domain.RefExample) error {
	return r.txm.RunInTx(ctx, func(txCtx context.Context) error {
		q := postgres.QuerierFromCtx(txCtx, r.pool)

		// 1. Fetch existing senses ordered by position.
		oldSenses, err := r.fetchOldSenseIDs(txCtx, q, entryID)
		if err != nil {
			return err
		}

		// 2. Group new translations/examples by the mapper-generated sense ID.
		trBySense := groupTranslationsBySense(translations)
		exBySense := groupExamplesBySense(examples)

		minSenses := min(len(oldSenses), len(senses))

		// 3. UPDATE matched senses in-place (preserves UUID → user FK stays valid).
		for i := range minSenses {
			oldID := oldSenses[i]
			ns := senses[i]

			if err := r.updateSense(txCtx, q, oldID, ns); err != nil {
				return err
			}
			if err := r.upsertTranslations(txCtx, q, oldID, trBySense[ns.ID]); err != nil {
				return err
			}
			if err := r.upsertExamples(txCtx, q, oldID, exBySense[ns.ID]); err != nil {
				return err
			}
		}

		// 4. INSERT excess new senses (+ their children).
		for i := minSenses; i < len(senses); i++ {
			ns := senses[i]
			if err := r.insertSense(txCtx, q, ns); err != nil {
				return err
			}
			if err := r.insertTranslations(txCtx, q, trBySense[ns.ID]); err != nil {
				return err
			}
			if err := r.insertExamples(txCtx, q, exBySense[ns.ID]); err != nil {
				return err
			}
		}

		// 5. DELETE excess old senses (FK ON DELETE SET NULL preserves user data).
		if len(oldSenses) > len(senses) {
			excessIDs := oldSenses[minSenses:]
			if _, err := q.Exec(txCtx,
				`DELETE FROM ref_senses WHERE id = ANY($1)`, excessIDs,
			); err != nil {
				return fmt.Errorf("delete excess senses: %w", err)
			}
		}

		// 6. Upsert source coverage for 'llm'.
		now := time.Now()
		_, err = q.Exec(txCtx,
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
// Upsert helpers (position-based matching)
// ---------------------------------------------------------------------------

// fetchOldSenseIDs returns existing sense UUIDs for an entry, ordered by position.
func (r *Repo) fetchOldSenseIDs(ctx context.Context, q postgres.Querier, entryID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := q.Query(ctx,
		`SELECT id FROM ref_senses WHERE ref_entry_id = $1 ORDER BY position`, entryID,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch old senses: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan old sense id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// updateSense updates an existing ref_sense row in-place, preserving its UUID.
func (r *Repo) updateSense(ctx context.Context, q postgres.Querier, oldID uuid.UUID, ns domain.RefSense) error {
	var pos *string
	if ns.PartOfSpeech != nil {
		p := string(*ns.PartOfSpeech)
		pos = &p
	}
	_, err := q.Exec(ctx,
		`UPDATE ref_senses SET definition = $1, part_of_speech = $2, cefr_level = $3,
		 notes = $4, source_slug = $5, position = $6 WHERE id = $7`,
		ns.Definition, pos, ns.CEFRLevel, ns.Notes, ns.SourceSlug, ns.Position, oldID,
	)
	if err != nil {
		return fmt.Errorf("update sense: %w", err)
	}
	return nil
}

// insertSense inserts a new ref_sense row.
func (r *Repo) insertSense(ctx context.Context, q postgres.Querier, s domain.RefSense) error {
	var pos *string
	if s.PartOfSpeech != nil {
		p := string(*s.PartOfSpeech)
		pos = &p
	}
	_, err := q.Exec(ctx,
		`INSERT INTO ref_senses (id, ref_entry_id, definition, part_of_speech, cefr_level, notes, source_slug, position, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		s.ID, s.RefEntryID, s.Definition, pos, s.CEFRLevel, s.Notes, s.SourceSlug, s.Position, s.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert sense: %w", err)
	}
	return nil
}

// upsertTranslations applies position-based upsert for translations within a sense.
func (r *Repo) upsertTranslations(ctx context.Context, q postgres.Querier, senseID uuid.UUID, newTrs []domain.RefTranslation) error {
	// Fetch existing translation IDs ordered by position.
	oldIDs, err := r.fetchChildIDs(ctx, q, "ref_translations", "ref_sense_id", senseID)
	if err != nil {
		return fmt.Errorf("fetch old translations: %w", err)
	}

	minTrs := min(len(oldIDs), len(newTrs))

	// UPDATE matched translations.
	for i := range minTrs {
		_, err := q.Exec(ctx,
			`UPDATE ref_translations SET text = $1, source_slug = $2, position = $3 WHERE id = $4`,
			newTrs[i].Text, newTrs[i].SourceSlug, newTrs[i].Position, oldIDs[i],
		)
		if err != nil {
			return fmt.Errorf("update translation: %w", err)
		}
	}

	// INSERT excess new translations (re-parent to existing sense).
	for i := minTrs; i < len(newTrs); i++ {
		tr := newTrs[i]
		_, err := q.Exec(ctx,
			`INSERT INTO ref_translations (id, ref_sense_id, text, source_slug, position)
			 VALUES ($1, $2, $3, $4, $5)`,
			tr.ID, senseID, tr.Text, tr.SourceSlug, tr.Position,
		)
		if err != nil {
			return fmt.Errorf("insert translation: %w", err)
		}
	}

	// DELETE excess old translations.
	if len(oldIDs) > len(newTrs) {
		excessIDs := oldIDs[minTrs:]
		if _, err := q.Exec(ctx,
			`DELETE FROM ref_translations WHERE id = ANY($1)`, excessIDs,
		); err != nil {
			return fmt.Errorf("delete excess translations: %w", err)
		}
	}

	return nil
}

// upsertExamples applies position-based upsert for examples within a sense.
func (r *Repo) upsertExamples(ctx context.Context, q postgres.Querier, senseID uuid.UUID, newExs []domain.RefExample) error {
	// Fetch existing example IDs ordered by position.
	oldIDs, err := r.fetchChildIDs(ctx, q, "ref_examples", "ref_sense_id", senseID)
	if err != nil {
		return fmt.Errorf("fetch old examples: %w", err)
	}

	minExs := min(len(oldIDs), len(newExs))

	// UPDATE matched examples.
	for i := range minExs {
		_, err := q.Exec(ctx,
			`UPDATE ref_examples SET sentence = $1, translation = $2, source_slug = $3, position = $4 WHERE id = $5`,
			newExs[i].Sentence, newExs[i].Translation, newExs[i].SourceSlug, newExs[i].Position, oldIDs[i],
		)
		if err != nil {
			return fmt.Errorf("update example: %w", err)
		}
	}

	// INSERT excess new examples (re-parent to existing sense).
	for i := minExs; i < len(newExs); i++ {
		ex := newExs[i]
		_, err := q.Exec(ctx,
			`INSERT INTO ref_examples (id, ref_sense_id, sentence, translation, source_slug, position)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			ex.ID, senseID, ex.Sentence, ex.Translation, ex.SourceSlug, ex.Position,
		)
		if err != nil {
			return fmt.Errorf("insert example: %w", err)
		}
	}

	// DELETE excess old examples.
	if len(oldIDs) > len(newExs) {
		excessIDs := oldIDs[minExs:]
		if _, err := q.Exec(ctx,
			`DELETE FROM ref_examples WHERE id = ANY($1)`, excessIDs,
		); err != nil {
			return fmt.Errorf("delete excess examples: %w", err)
		}
	}

	return nil
}

// insertTranslations inserts a batch of translations for a newly inserted sense.
func (r *Repo) insertTranslations(ctx context.Context, q postgres.Querier, trs []domain.RefTranslation) error {
	for _, tr := range trs {
		_, err := q.Exec(ctx,
			`INSERT INTO ref_translations (id, ref_sense_id, text, source_slug, position)
			 VALUES ($1, $2, $3, $4, $5)`,
			tr.ID, tr.RefSenseID, tr.Text, tr.SourceSlug, tr.Position,
		)
		if err != nil {
			return fmt.Errorf("insert translation: %w", err)
		}
	}
	return nil
}

// insertExamples inserts a batch of examples for a newly inserted sense.
func (r *Repo) insertExamples(ctx context.Context, q postgres.Querier, exs []domain.RefExample) error {
	for _, ex := range exs {
		_, err := q.Exec(ctx,
			`INSERT INTO ref_examples (id, ref_sense_id, sentence, translation, source_slug, position)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			ex.ID, ex.RefSenseID, ex.Sentence, ex.Translation, ex.SourceSlug, ex.Position,
		)
		if err != nil {
			return fmt.Errorf("insert example: %w", err)
		}
	}
	return nil
}

// fetchChildIDs returns UUIDs of child rows ordered by position.
func (r *Repo) fetchChildIDs(ctx context.Context, q postgres.Querier, table, fkCol string, parentID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := q.Query(ctx,
		fmt.Sprintf(`SELECT id FROM %s WHERE %s = $1 ORDER BY position`, table, fkCol), parentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// groupTranslationsBySense groups translations by their mapper-generated RefSenseID.
func groupTranslationsBySense(trs []domain.RefTranslation) map[uuid.UUID][]domain.RefTranslation {
	m := make(map[uuid.UUID][]domain.RefTranslation, len(trs))
	for _, tr := range trs {
		m[tr.RefSenseID] = append(m[tr.RefSenseID], tr)
	}
	return m
}

// groupExamplesBySense groups examples by their mapper-generated RefSenseID.
func groupExamplesBySense(exs []domain.RefExample) map[uuid.UUID][]domain.RefExample {
	m := make(map[uuid.UUID][]domain.RefExample, len(exs))
	for _, ex := range exs {
		m[ex.RefSenseID] = append(m[ex.RefSenseID], ex)
	}
	return m
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

