// Package refentry implements the Reference Catalog repository using PostgreSQL.
// It manages 6 tables (ref_entries + 5 child tables) as a single aggregate.
// The catalog is immutable: no Update/Delete operations are exposed.
package refentry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	postgres "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/refentry/sqlc"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// Repo provides reference catalog persistence backed by PostgreSQL.
type Repo struct {
	pool *pgxpool.Pool
	txm  *postgres.TxManager
}

// New creates a new reference catalog repository.
func New(pool *pgxpool.Pool, txm *postgres.TxManager) *Repo {
	return &Repo{pool: pool, txm: txm}
}

// ---------------------------------------------------------------------------
// Read operations
// ---------------------------------------------------------------------------

// GetFullTreeByID returns a ref_entry with full tree (senses -> translations, examples;
// pronunciations; images). Returns domain.ErrNotFound if not found.
func (r *Repo) GetFullTreeByID(ctx context.Context, id uuid.UUID) (*domain.RefEntry, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.GetRefEntryByID(ctx, id)
	if err != nil {
		return nil, mapError(err, "ref_entry", id)
	}

	entry := toDomainRefEntry(fromGetByID(row))

	if err := r.loadFullTree(ctx, q, &entry); err != nil {
		return nil, err
	}

	return &entry, nil
}

// GetFullTreeByText returns a ref_entry by normalized text with full tree
// (senses -> translations, examples; pronunciations; images).
// Returns domain.ErrNotFound if not found.
func (r *Repo) GetFullTreeByText(ctx context.Context, textNormalized string) (*domain.RefEntry, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.GetRefEntryByNormalizedText(ctx, textNormalized)
	if err != nil {
		return nil, mapError(err, "ref_entry", uuid.Nil)
	}

	entry := toDomainRefEntry(fromGetByText(row))

	if err := r.loadFullTree(ctx, q, &entry); err != nil {
		return nil, err
	}

	return &entry, nil
}

// Search performs fuzzy search by text_normalized using pg_trgm.
// Empty query returns empty result without a DB query.
func (r *Repo) Search(ctx context.Context, query string, limit int) ([]domain.RefEntry, error) {
	if query == "" {
		return []domain.RefEntry{}, nil
	}

	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	rows, err := q.SearchRefEntries(ctx, sqlc.SearchRefEntriesParams{
		Query: query,
		Lim:   int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("search ref_entries: %w", err)
	}

	entries := make([]domain.RefEntry, len(rows))
	for i, row := range rows {
		entries[i] = toDomainRefEntry(fromSearch(row))
	}

	return entries, nil
}

// ---------------------------------------------------------------------------
// Write operations
// ---------------------------------------------------------------------------

// CreateWithTree inserts a ref_entry and all children in one transaction.
// Returns the created domain.RefEntry with all IDs populated.
func (r *Repo) CreateWithTree(ctx context.Context, entry *domain.RefEntry) (*domain.RefEntry, error) {
	var result domain.RefEntry

	err := r.txm.RunInTx(ctx, func(txCtx context.Context) error {
		q := sqlc.New(postgres.QuerierFromCtx(txCtx, r.pool))

		// Insert ref_entry.
		reRow, err := q.InsertRefEntry(txCtx, sqlc.InsertRefEntryParams{
			ID:             entry.ID,
			Text:           entry.Text,
			TextNormalized: entry.TextNormalized,
			FrequencyRank:  intPtrToPgInt4(entry.FrequencyRank),
			CefrLevel:      ptrStringToPgText(entry.CEFRLevel),
			IsCoreLexicon:  boolToPgBool(entry.IsCoreLexicon),
			CreatedAt:      entry.CreatedAt,
		})
		if err != nil {
			return mapError(err, "ref_entry", entry.ID)
		}
		result = toDomainRefEntry(fromInsert(reRow))

		// Insert senses with their children.
		result.Senses = make([]domain.RefSense, len(entry.Senses))
		for i, sense := range entry.Senses {
			senseRow, err := q.InsertRefSense(txCtx, sqlc.InsertRefSenseParams{
				ID:           sense.ID,
				RefEntryID:   entry.ID,
				Definition:   stringToPgText(sense.Definition),
				PartOfSpeech: partOfSpeechToSqlc(sense.PartOfSpeech),
				CefrLevel:    ptrStringToPgText(sense.CEFRLevel),
				Notes:        ptrStringToPgText(sense.Notes),
				SourceSlug:   sense.SourceSlug,
				Position:     int32(sense.Position),
				CreatedAt:    sense.CreatedAt,
			})
			if err != nil {
				return mapError(err, "ref_sense", sense.ID)
			}

			domSense := toDomainRefSense(senseRow)

			// Insert translations for this sense.
			domSense.Translations = make([]domain.RefTranslation, len(sense.Translations))
			for j, tr := range sense.Translations {
				trRow, err := q.InsertRefTranslation(txCtx, sqlc.InsertRefTranslationParams{
					ID:         tr.ID,
					RefSenseID: sense.ID,
					Text:       tr.Text,
					SourceSlug: tr.SourceSlug,
					Position:   int32(tr.Position),
				})
				if err != nil {
					return mapError(err, "ref_translation", tr.ID)
				}
				domSense.Translations[j] = toDomainRefTranslation(trRow)
			}

			// Insert examples for this sense.
			domSense.Examples = make([]domain.RefExample, len(sense.Examples))
			for j, ex := range sense.Examples {
				exRow, err := q.InsertRefExample(txCtx, sqlc.InsertRefExampleParams{
					ID:          ex.ID,
					RefSenseID:  sense.ID,
					Sentence:    ex.Sentence,
					Translation: ptrStringToPgText(ex.Translation),
					SourceSlug:  ex.SourceSlug,
					Position:    int32(ex.Position),
				})
				if err != nil {
					return mapError(err, "ref_example", ex.ID)
				}
				domSense.Examples[j] = toDomainRefExample(exRow)
			}

			result.Senses[i] = domSense
		}

		// Insert pronunciations.
		result.Pronunciations = make([]domain.RefPronunciation, len(entry.Pronunciations))
		for i, pron := range entry.Pronunciations {
			pronRow, err := q.InsertRefPronunciation(txCtx, sqlc.InsertRefPronunciationParams{
				ID:            pron.ID,
				RefEntryID:    entry.ID,
				Transcription: ptrStringToString(pron.Transcription),
				AudioUrl:      ptrStringToPgText(pron.AudioURL),
				Region:        ptrStringToPgText(pron.Region),
				SourceSlug:    pron.SourceSlug,
			})
			if err != nil {
				return mapError(err, "ref_pronunciation", pron.ID)
			}
			result.Pronunciations[i] = toDomainRefPronunciation(pronRow)
		}

		// Insert images.
		result.Images = make([]domain.RefImage, len(entry.Images))
		for i, img := range entry.Images {
			imgRow, err := q.InsertRefImage(txCtx, sqlc.InsertRefImageParams{
				ID:         img.ID,
				RefEntryID: entry.ID,
				Url:        img.URL,
				Caption:    ptrStringToPgText(img.Caption),
				SourceSlug: img.SourceSlug,
			})
			if err != nil {
				return mapError(err, "ref_image", img.ID)
			}
			result.Images[i] = toDomainRefImage(imgRow)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetOrCreate performs an upsert: INSERT ON CONFLICT DO NOTHING, then SELECT.
// Returns the ref_entry (new or existing) without the full tree.
// Concurrent callers with the same text all succeed and return the same entry.
func (r *Repo) GetOrCreate(ctx context.Context, id uuid.UUID, text, textNormalized string) (domain.RefEntry, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	// Attempt insert; if text_normalized already exists, does nothing.
	if err := q.UpsertRefEntry(ctx, sqlc.UpsertRefEntryParams{
		ID:             id,
		Text:           text,
		TextNormalized: textNormalized,
	}); err != nil {
		return domain.RefEntry{}, mapError(err, "ref_entry", id)
	}

	// Always select to get the definitive row (new or existing).
	row, err := q.GetRefEntryByNormalizedText(ctx, textNormalized)
	if err != nil {
		return domain.RefEntry{}, mapError(err, "ref_entry", id)
	}

	return toDomainRefEntry(fromGetByText(row)), nil
}

// ---------------------------------------------------------------------------
// Batch operations (for DataLoaders)
// ---------------------------------------------------------------------------

// GetRefSensesByIDs returns ref_senses for the given IDs.
func (r *Repo) GetRefSensesByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.RefSense, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	rows, err := q.GetRefSensesByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get ref_senses by ids: %w", err)
	}

	senses := make([]domain.RefSense, len(rows))
	for i, row := range rows {
		senses[i] = toDomainRefSense(row)
	}

	return senses, nil
}

// GetRefTranslationsByIDs returns ref_translations for the given IDs.
func (r *Repo) GetRefTranslationsByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.RefTranslation, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	rows, err := q.GetRefTranslationsByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get ref_translations by ids: %w", err)
	}

	translations := make([]domain.RefTranslation, len(rows))
	for i, row := range rows {
		translations[i] = toDomainRefTranslation(row)
	}

	return translations, nil
}

// GetRefExamplesByIDs returns ref_examples for the given IDs.
func (r *Repo) GetRefExamplesByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.RefExample, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	rows, err := q.GetRefExamplesByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get ref_examples by ids: %w", err)
	}

	examples := make([]domain.RefExample, len(rows))
	for i, row := range rows {
		examples[i] = toDomainRefExample(row)
	}

	return examples, nil
}

// GetRefPronunciationsByIDs returns ref_pronunciations for the given IDs.
func (r *Repo) GetRefPronunciationsByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.RefPronunciation, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	rows, err := q.GetRefPronunciationsByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get ref_pronunciations by ids: %w", err)
	}

	pronunciations := make([]domain.RefPronunciation, len(rows))
	for i, row := range rows {
		pronunciations[i] = toDomainRefPronunciation(row)
	}

	return pronunciations, nil
}

// GetRefImagesByIDs returns ref_images for the given IDs.
func (r *Repo) GetRefImagesByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.RefImage, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	rows, err := q.GetRefImagesByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get ref_images by ids: %w", err)
	}

	images := make([]domain.RefImage, len(rows))
	for i, row := range rows {
		images[i] = toDomainRefImage(row)
	}

	return images, nil
}

// ---------------------------------------------------------------------------
// Internal: load full tree for GetByID
// ---------------------------------------------------------------------------

// loadFullTree populates senses (with translations, examples), pronunciations,
// and images on the given entry.
func (r *Repo) loadFullTree(ctx context.Context, q *sqlc.Queries, entry *domain.RefEntry) error {
	// Load senses.
	senseRows, err := q.GetRefSensesByEntryID(ctx, entry.ID)
	if err != nil {
		return fmt.Errorf("load ref_senses for entry %s: %w", entry.ID, err)
	}

	if len(senseRows) > 0 {
		// Collect sense IDs for batch loading translations & examples.
		senseIDs := make([]uuid.UUID, len(senseRows))
		for i, s := range senseRows {
			senseIDs[i] = s.ID
		}

		// Batch load translations and examples.
		trRows, err := q.GetRefTranslationsBySenseIDs(ctx, senseIDs)
		if err != nil {
			return fmt.Errorf("load ref_translations for entry %s: %w", entry.ID, err)
		}

		exRows, err := q.GetRefExamplesBySenseIDs(ctx, senseIDs)
		if err != nil {
			return fmt.Errorf("load ref_examples for entry %s: %w", entry.ID, err)
		}

		// Group translations by sense ID.
		trBySense := make(map[uuid.UUID][]domain.RefTranslation)
		for _, tr := range trRows {
			dt := toDomainRefTranslation(tr)
			trBySense[dt.RefSenseID] = append(trBySense[dt.RefSenseID], dt)
		}

		// Group examples by sense ID.
		exBySense := make(map[uuid.UUID][]domain.RefExample)
		for _, ex := range exRows {
			de := toDomainRefExample(ex)
			exBySense[de.RefSenseID] = append(exBySense[de.RefSenseID], de)
		}

		// Build senses with their children.
		entry.Senses = make([]domain.RefSense, len(senseRows))
		for i, s := range senseRows {
			ds := toDomainRefSense(s)
			if trs, ok := trBySense[ds.ID]; ok {
				ds.Translations = trs
			} else {
				ds.Translations = []domain.RefTranslation{}
			}
			if exs, ok := exBySense[ds.ID]; ok {
				ds.Examples = exs
			} else {
				ds.Examples = []domain.RefExample{}
			}
			entry.Senses[i] = ds
		}
	} else {
		entry.Senses = []domain.RefSense{}
	}

	// Load pronunciations.
	pronRows, err := q.GetRefPronunciationsByEntryID(ctx, entry.ID)
	if err != nil {
		return fmt.Errorf("load ref_pronunciations for entry %s: %w", entry.ID, err)
	}

	entry.Pronunciations = make([]domain.RefPronunciation, len(pronRows))
	for i, p := range pronRows {
		entry.Pronunciations[i] = toDomainRefPronunciation(p)
	}

	// Load images.
	imgRows, err := q.GetRefImagesByEntryID(ctx, entry.ID)
	if err != nil {
		return fmt.Errorf("load ref_images for entry %s: %w", entry.ID, err)
	}

	entry.Images = make([]domain.RefImage, len(imgRows))
	for i, img := range imgRows {
		entry.Images[i] = toDomainRefImage(img)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Error mapping
// ---------------------------------------------------------------------------

// mapError converts pgx/pgconn errors into domain errors.
func mapError(err error, entity string, id uuid.UUID) error {
	if err == nil {
		return nil
	}

	// context errors pass through as-is
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s %s: %w", entity, id, err)
	}

	// pgx.ErrNoRows -> domain.ErrNotFound
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s %s: %w", entity, id, domain.ErrNotFound)
	}

	// PgError codes
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return fmt.Errorf("%s %s: %w", entity, id, domain.ErrAlreadyExists)
		case "23503": // foreign_key_violation
			return fmt.Errorf("%s %s: %w", entity, id, domain.ErrNotFound)
		case "23514": // check_violation
			return fmt.Errorf("%s %s: %w", entity, id, domain.ErrValidation)
		}
	}

	// Everything else: wrap with context
	return fmt.Errorf("%s %s: %w", entity, id, err)
}

// ---------------------------------------------------------------------------
// Mapping helpers: sqlc -> domain
// ---------------------------------------------------------------------------

// refEntryRow is the common field set returned by all ref_entry queries.
type refEntryRow struct {
	ID             uuid.UUID
	Text           string
	TextNormalized string
	FrequencyRank  pgtype.Int4
	CefrLevel      pgtype.Text
	IsCoreLexicon  pgtype.Bool
	CreatedAt      time.Time
}

func fromGetByID(r sqlc.GetRefEntryByIDRow) refEntryRow {
	return refEntryRow{r.ID, r.Text, r.TextNormalized, r.FrequencyRank, r.CefrLevel, r.IsCoreLexicon, r.CreatedAt}
}

func fromGetByText(r sqlc.GetRefEntryByNormalizedTextRow) refEntryRow {
	return refEntryRow{r.ID, r.Text, r.TextNormalized, r.FrequencyRank, r.CefrLevel, r.IsCoreLexicon, r.CreatedAt}
}

func fromSearch(r sqlc.SearchRefEntriesRow) refEntryRow {
	return refEntryRow{r.ID, r.Text, r.TextNormalized, r.FrequencyRank, r.CefrLevel, r.IsCoreLexicon, r.CreatedAt}
}

func fromInsert(r sqlc.InsertRefEntryRow) refEntryRow {
	return refEntryRow{r.ID, r.Text, r.TextNormalized, r.FrequencyRank, r.CefrLevel, r.IsCoreLexicon, r.CreatedAt}
}

func toDomainRefEntry(row refEntryRow) domain.RefEntry {
	return domain.RefEntry{
		ID:             row.ID,
		Text:           row.Text,
		TextNormalized: row.TextNormalized,
		FrequencyRank:  domain.Int32PtrToIntPtr(pgInt4ToPtr(row.FrequencyRank)),
		CEFRLevel:      pgTextToPtr(row.CefrLevel),
		IsCoreLexicon:  pgBoolValue(row.IsCoreLexicon),
		CreatedAt:      row.CreatedAt,
	}
}

// pgInt4ToPtr converts pgtype.Int4 to *int32.
func pgInt4ToPtr(v pgtype.Int4) *int32 {
	if v.Valid {
		return &v.Int32
	}
	return nil
}

// pgBoolValue returns the bool value from pgtype.Bool, defaulting to false.
func pgBoolValue(v pgtype.Bool) bool {
	if v.Valid {
		return v.Bool
	}
	return false
}

// intPtrToPgInt4 converts *int to pgtype.Int4.
func intPtrToPgInt4(v *int) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(*v), Valid: true}
}

// boolToPgBool converts bool to pgtype.Bool.
func boolToPgBool(v bool) pgtype.Bool {
	return pgtype.Bool{Bool: v, Valid: true}
}

func toDomainRefSense(row sqlc.RefSense) domain.RefSense {
	s := domain.RefSense{
		ID:         row.ID,
		RefEntryID: row.RefEntryID,
		Definition: pgTextToString(row.Definition),
		CEFRLevel:  pgTextToPtr(row.CefrLevel),
		Notes:      pgTextToPtr(row.Notes),
		SourceSlug: row.SourceSlug,
		Position:   int(row.Position),
		CreatedAt:  row.CreatedAt,
	}

	if row.PartOfSpeech.Valid {
		pos := domain.PartOfSpeech(row.PartOfSpeech.PartOfSpeech)
		s.PartOfSpeech = &pos
	}

	return s
}

func toDomainRefTranslation(row sqlc.RefTranslation) domain.RefTranslation {
	return domain.RefTranslation{
		ID:         row.ID,
		RefSenseID: row.RefSenseID,
		Text:       row.Text,
		SourceSlug: row.SourceSlug,
		Position:   int(row.Position),
	}
}

func toDomainRefExample(row sqlc.RefExample) domain.RefExample {
	return domain.RefExample{
		ID:          row.ID,
		RefSenseID:  row.RefSenseID,
		Sentence:    row.Sentence,
		Translation: pgTextToPtr(row.Translation),
		SourceSlug:  row.SourceSlug,
		Position:    int(row.Position),
	}
}

func toDomainRefPronunciation(row sqlc.RefPronunciation) domain.RefPronunciation {
	return domain.RefPronunciation{
		ID:            row.ID,
		RefEntryID:    row.RefEntryID,
		Transcription: stringToPtr(row.Transcription),
		AudioURL:      pgTextToPtr(row.AudioUrl),
		Region:        pgTextToPtr(row.Region),
		SourceSlug:    row.SourceSlug,
	}
}

func toDomainRefImage(row sqlc.RefImage) domain.RefImage {
	return domain.RefImage{
		ID:         row.ID,
		RefEntryID: row.RefEntryID,
		URL:        row.Url,
		Caption:    pgTextToPtr(row.Caption),
		SourceSlug: row.SourceSlug,
	}
}

// ---------------------------------------------------------------------------
// pgtype helpers
// ---------------------------------------------------------------------------

// pgTextToString returns the string value or empty string if invalid (NULL).
func pgTextToString(t pgtype.Text) string {
	if t.Valid {
		return t.String
	}
	return ""
}

// pgTextToPtr returns a *string (nil when NULL).
func pgTextToPtr(t pgtype.Text) *string {
	if t.Valid {
		return &t.String
	}
	return nil
}

// stringToPgText converts a Go string to pgtype.Text.
// An empty string is stored as a valid empty text, not NULL.
func stringToPgText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

// ptrStringToPgText converts a *string to pgtype.Text (nil -> NULL).
func ptrStringToPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// stringToPtr converts a non-empty string to *string, empty to nil.
func stringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ptrStringToString converts a *string to string, nil to empty.
func ptrStringToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// partOfSpeechToSqlc converts a domain *PartOfSpeech to sqlc NullPartOfSpeech.
func partOfSpeechToSqlc(pos *domain.PartOfSpeech) sqlc.NullPartOfSpeech {
	if pos == nil {
		return sqlc.NullPartOfSpeech{}
	}
	return sqlc.NullPartOfSpeech{
		PartOfSpeech: sqlc.PartOfSpeech(*pos),
		Valid:        true,
	}
}
