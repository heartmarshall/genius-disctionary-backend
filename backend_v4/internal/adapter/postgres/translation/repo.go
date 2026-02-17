// Package translation implements the Translation repository using PostgreSQL.
// Read queries use raw SQL with COALESCE + LEFT JOIN ref_translations to resolve
// inherited text. Write queries use sqlc-generated code.
package translation

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	postgres "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/translation/sqlc"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// Repo provides translation persistence backed by PostgreSQL.
type Repo struct {
	pool *pgxpool.Pool
	txm  *postgres.TxManager
}

// New creates a new translation repository.
func New(pool *pgxpool.Pool, txm *postgres.TxManager) *Repo {
	return &Repo{pool: pool, txm: txm}
}

// ---------------------------------------------------------------------------
// Raw SQL for COALESCE read queries
// ---------------------------------------------------------------------------

const getBySenseIDSQL = `
SELECT
    t.id, t.sense_id,
    COALESCE(t.text, rt.text) AS text,
    t.source_slug, t.position, t.ref_translation_id
FROM translations t
LEFT JOIN ref_translations rt ON t.ref_translation_id = rt.id
WHERE t.sense_id = $1
ORDER BY t.position`

const getBySenseIDsSQL = `
SELECT
    t.id, t.sense_id,
    COALESCE(t.text, rt.text) AS text,
    t.source_slug, t.position, t.ref_translation_id
FROM translations t
LEFT JOIN ref_translations rt ON t.ref_translation_id = rt.id
WHERE t.sense_id = ANY($1::uuid[])
ORDER BY t.sense_id, t.position`

const getByIDSQL = `
SELECT
    t.id, t.sense_id,
    COALESCE(t.text, rt.text) AS text,
    t.source_slug, t.position, t.ref_translation_id
FROM translations t
LEFT JOIN ref_translations rt ON t.ref_translation_id = rt.id
WHERE t.id = $1`

// ---------------------------------------------------------------------------
// Read operations
// ---------------------------------------------------------------------------

// GetBySenseID returns all translations for a sense with COALESCE-resolved text,
// ordered by position.
func (r *Repo) GetBySenseID(ctx context.Context, senseID uuid.UUID) ([]domain.Translation, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	rows, err := querier.Query(ctx, getBySenseIDSQL, senseID)
	if err != nil {
		return nil, fmt.Errorf("get translations by sense_id: %w", err)
	}
	defer rows.Close()

	translations, err := scanTranslations(rows)
	if err != nil {
		return nil, fmt.Errorf("get translations by sense_id: %w", err)
	}

	return translations, nil
}

// GetBySenseIDs returns translations for multiple senses (batch for DataLoader).
// Results include SenseID in domain.Translation for grouping by the caller.
func (r *Repo) GetBySenseIDs(ctx context.Context, senseIDs []uuid.UUID) ([]domain.Translation, error) {
	if len(senseIDs) == 0 {
		return []domain.Translation{}, nil
	}

	querier := postgres.QuerierFromCtx(ctx, r.pool)

	rows, err := querier.Query(ctx, getBySenseIDsSQL, senseIDs)
	if err != nil {
		return nil, fmt.Errorf("get translations by sense_ids: %w", err)
	}
	defer rows.Close()

	translations, err := scanTranslations(rows)
	if err != nil {
		return nil, fmt.Errorf("get translations by sense_ids: %w", err)
	}

	return translations, nil
}

// GetByID returns a single translation with COALESCE-resolved text.
func (r *Repo) GetByID(ctx context.Context, translationID uuid.UUID) (*domain.Translation, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	row := querier.QueryRow(ctx, getByIDSQL, translationID)

	tr, err := scanTranslationRow(row)
	if err != nil {
		return nil, mapError(err, "translation", translationID)
	}

	return &tr, nil
}

// CountBySense returns the number of translations for a sense.
func (r *Repo) CountBySense(ctx context.Context, senseID uuid.UUID) (int, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	count, err := q.CountBySense(ctx, senseID)
	if err != nil {
		return 0, fmt.Errorf("count translations: %w", err)
	}

	return int(count), nil
}

// ---------------------------------------------------------------------------
// Write operations
// ---------------------------------------------------------------------------

// CreateFromRef creates a translation linked to a reference translation. The text
// field stays NULL -- COALESCE picks up the ref value.
// Position is auto-calculated as MAX(position)+1.
func (r *Repo) CreateFromRef(ctx context.Context, senseID, refTranslationID uuid.UUID, sourceSlug string) (*domain.Translation, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.CreateTranslationFromRef(ctx, sqlc.CreateTranslationFromRefParams{
		ID:               uuid.New(),
		SenseID:          senseID,
		RefTranslationID: pgtype.UUID{Bytes: refTranslationID, Valid: true},
		SourceSlug:       sourceSlug,
	})
	if err != nil {
		return nil, mapError(err, "translation", uuid.Nil)
	}

	// The sqlc RETURNING gives raw values (without COALESCE). Re-read with COALESCE.
	return r.GetByID(ctx, row.ID)
}

// CreateCustom creates a custom translation with user-provided text (no ref link).
// Position is auto-calculated as MAX(position)+1.
func (r *Repo) CreateCustom(ctx context.Context, senseID uuid.UUID, text string, sourceSlug string) (*domain.Translation, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.CreateTranslationCustom(ctx, sqlc.CreateTranslationCustomParams{
		ID:         uuid.New(),
		SenseID:    senseID,
		Text:       pgtype.Text{String: text, Valid: true},
		SourceSlug: sourceSlug,
	})
	if err != nil {
		return nil, mapError(err, "translation", uuid.Nil)
	}

	tr := toDomainTranslation(row)
	return &tr, nil
}

// Update modifies the translation text. ref_translation_id is NOT touched
// (preserves origin link). Returns the translation with COALESCE-resolved text.
func (r *Repo) Update(ctx context.Context, translationID uuid.UUID, text string) (*domain.Translation, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	_, err := q.UpdateTranslation(ctx, sqlc.UpdateTranslationParams{
		ID:   translationID,
		Text: pgtype.Text{String: text, Valid: true},
	})
	if err != nil {
		return nil, mapError(err, "translation", translationID)
	}

	// Re-read with COALESCE to return resolved values.
	return r.GetByID(ctx, translationID)
}

// Delete removes a translation by ID. Returns domain.ErrNotFound if 0 rows affected.
func (r *Repo) Delete(ctx context.Context, translationID uuid.UUID) error {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	n, err := q.DeleteTranslation(ctx, translationID)
	if err != nil {
		return mapError(err, "translation", translationID)
	}
	if n == 0 {
		return fmt.Errorf("translation %s: %w", translationID, domain.ErrNotFound)
	}

	return nil
}

// Reorder updates positions for a batch of translations atomically within a transaction.
func (r *Repo) Reorder(ctx context.Context, items []domain.ReorderItem) error {
	if len(items) == 0 {
		return nil
	}

	return r.txm.RunInTx(ctx, func(txCtx context.Context) error {
		q := sqlc.New(postgres.QuerierFromCtx(txCtx, r.pool))

		for _, item := range items {
			if err := q.UpdateTranslationPosition(txCtx, sqlc.UpdateTranslationPositionParams{
				ID:       item.ID,
				Position: int32(item.Position),
			}); err != nil {
				return mapError(err, "translation", item.ID)
			}
		}

		return nil
	})
}

// ---------------------------------------------------------------------------
// Row scanning helpers
// ---------------------------------------------------------------------------

// scanTranslations scans multiple rows from a COALESCE query into domain.Translation slices.
func scanTranslations(rows pgx.Rows) ([]domain.Translation, error) {
	var translations []domain.Translation
	for rows.Next() {
		tr, err := scanTranslationFromRows(rows)
		if err != nil {
			return nil, err
		}
		translations = append(translations, tr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if translations == nil {
		translations = []domain.Translation{}
	}

	return translations, nil
}

// scanTranslationFromRows scans a single row from pgx.Rows into a domain.Translation.
func scanTranslationFromRows(rows pgx.Rows) (domain.Translation, error) {
	var (
		id               uuid.UUID
		senseID          uuid.UUID
		text             pgtype.Text
		sourceSlug       string
		position         int32
		refTranslationID pgtype.UUID
	)

	if err := rows.Scan(&id, &senseID, &text, &sourceSlug, &position, &refTranslationID); err != nil {
		return domain.Translation{}, err
	}

	return buildDomainTranslation(id, senseID, text, sourceSlug, position, refTranslationID), nil
}

// scanTranslationRow scans a single pgx.Row into a domain.Translation.
func scanTranslationRow(row pgx.Row) (domain.Translation, error) {
	var (
		id               uuid.UUID
		senseID          uuid.UUID
		text             pgtype.Text
		sourceSlug       string
		position         int32
		refTranslationID pgtype.UUID
	)

	if err := row.Scan(&id, &senseID, &text, &sourceSlug, &position, &refTranslationID); err != nil {
		return domain.Translation{}, err
	}

	return buildDomainTranslation(id, senseID, text, sourceSlug, position, refTranslationID), nil
}

// buildDomainTranslation constructs a domain.Translation from scanned values.
func buildDomainTranslation(id, senseID uuid.UUID, text pgtype.Text, sourceSlug string, position int32, refTranslationID pgtype.UUID) domain.Translation {
	tr := domain.Translation{
		ID:         id,
		SenseID:    senseID,
		SourceSlug: sourceSlug,
		Position:   int(position),
	}

	if text.Valid {
		tr.Text = &text.String
	}

	if refTranslationID.Valid {
		rid := uuid.UUID(refTranslationID.Bytes)
		tr.RefTranslationID = &rid
	}

	return tr
}

// ---------------------------------------------------------------------------
// Mapping helpers: sqlc -> domain (for write query results)
// ---------------------------------------------------------------------------

// toDomainTranslation converts a sqlc.Translation to a domain.Translation (raw values, no COALESCE).
func toDomainTranslation(row sqlc.Translation) domain.Translation {
	tr := domain.Translation{
		ID:         row.ID,
		SenseID:    row.SenseID,
		SourceSlug: row.SourceSlug,
		Position:   int(row.Position),
	}

	if row.Text.Valid {
		tr.Text = &row.Text.String
	}

	if row.RefTranslationID.Valid {
		rid := uuid.UUID(row.RefTranslationID.Bytes)
		tr.RefTranslationID = &rid
	}

	return tr
}

// ---------------------------------------------------------------------------
// Error mapping
// ---------------------------------------------------------------------------

// mapError converts pgx/pgconn errors into domain errors.
func mapError(err error, entity string, id uuid.UUID) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s %s: %w", entity, id, err)
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s %s: %w", entity, id, domain.ErrNotFound)
	}

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

	return fmt.Errorf("%s %s: %w", entity, id, err)
}

