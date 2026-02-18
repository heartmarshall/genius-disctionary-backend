// Package example implements the Example repository using PostgreSQL.
// Read queries use raw SQL with COALESCE + LEFT JOIN ref_examples to resolve
// inherited fields. Write queries use sqlc-generated code.
package example

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
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/example/sqlc"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// Repo provides example persistence backed by PostgreSQL.
type Repo struct {
	pool *pgxpool.Pool
	txm  *postgres.TxManager
}

// New creates a new example repository.
func New(pool *pgxpool.Pool, txm *postgres.TxManager) *Repo {
	return &Repo{pool: pool, txm: txm}
}

// ---------------------------------------------------------------------------
// Raw SQL for COALESCE read queries
// ---------------------------------------------------------------------------

const getBySenseIDSQL = `
SELECT
    e.id, e.sense_id,
    COALESCE(e.sentence, re.sentence) AS sentence,
    COALESCE(e.translation, re.translation) AS translation,
    e.source_slug, e.position, e.ref_example_id, e.created_at
FROM examples e
LEFT JOIN ref_examples re ON e.ref_example_id = re.id
WHERE e.sense_id = $1
ORDER BY e.position`

const getBySenseIDsSQL = `
SELECT
    e.id, e.sense_id,
    COALESCE(e.sentence, re.sentence) AS sentence,
    COALESCE(e.translation, re.translation) AS translation,
    e.source_slug, e.position, e.ref_example_id, e.created_at
FROM examples e
LEFT JOIN ref_examples re ON e.ref_example_id = re.id
WHERE e.sense_id = ANY($1::uuid[])
ORDER BY e.sense_id, e.position`

const getByIDSQL = `
SELECT
    e.id, e.sense_id,
    COALESCE(e.sentence, re.sentence) AS sentence,
    COALESCE(e.translation, re.translation) AS translation,
    e.source_slug, e.position, e.ref_example_id, e.created_at
FROM examples e
LEFT JOIN ref_examples re ON e.ref_example_id = re.id
WHERE e.id = $1`

const getByIDForUserSQL = `
SELECT
    ex.id, ex.sense_id,
    COALESCE(ex.sentence, re.sentence) AS sentence,
    COALESCE(ex.translation, re.translation) AS translation,
    ex.source_slug, ex.position, ex.ref_example_id, ex.created_at
FROM examples ex
LEFT JOIN ref_examples re ON ex.ref_example_id = re.id
JOIN senses s ON s.id = ex.sense_id
JOIN entries e ON e.id = s.entry_id
WHERE ex.id = $1 AND e.user_id = $2 AND e.deleted_at IS NULL`

// ---------------------------------------------------------------------------
// Read operations
// ---------------------------------------------------------------------------

// GetBySenseID returns all examples for a sense with COALESCE-resolved fields,
// ordered by position.
func (r *Repo) GetBySenseID(ctx context.Context, senseID uuid.UUID) ([]domain.Example, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	rows, err := querier.Query(ctx, getBySenseIDSQL, senseID)
	if err != nil {
		return nil, fmt.Errorf("get examples by sense_id: %w", err)
	}
	defer rows.Close()

	examples, err := scanExamples(rows)
	if err != nil {
		return nil, fmt.Errorf("get examples by sense_id: %w", err)
	}

	return examples, nil
}

// GetBySenseIDs returns examples for multiple senses (batch for DataLoader).
// Results include SenseID in domain.Example for grouping by the caller.
func (r *Repo) GetBySenseIDs(ctx context.Context, senseIDs []uuid.UUID) ([]domain.Example, error) {
	if len(senseIDs) == 0 {
		return []domain.Example{}, nil
	}

	querier := postgres.QuerierFromCtx(ctx, r.pool)

	rows, err := querier.Query(ctx, getBySenseIDsSQL, senseIDs)
	if err != nil {
		return nil, fmt.Errorf("get examples by sense_ids: %w", err)
	}
	defer rows.Close()

	examples, err := scanExamples(rows)
	if err != nil {
		return nil, fmt.Errorf("get examples by sense_ids: %w", err)
	}

	return examples, nil
}

// GetByID returns a single example with COALESCE-resolved fields.
func (r *Repo) GetByID(ctx context.Context, exampleID uuid.UUID) (*domain.Example, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	row := querier.QueryRow(ctx, getByIDSQL, exampleID)

	example, err := scanExampleRow(row)
	if err != nil {
		return nil, mapError(err, "example", exampleID)
	}

	return &example, nil
}

// GetByIDForUser returns a single example with COALESCE-resolved fields,
// verifying that the parent entry belongs to the given user (single JOIN query).
// Returns domain.ErrNotFound if the example does not exist or the entry is not owned.
func (r *Repo) GetByIDForUser(ctx context.Context, userID, exampleID uuid.UUID) (*domain.Example, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	row := querier.QueryRow(ctx, getByIDForUserSQL, exampleID, userID)

	example, err := scanExampleRow(row)
	if err != nil {
		return nil, mapError(err, "example", exampleID)
	}

	return &example, nil
}

// CountBySense returns the number of examples for a sense.
func (r *Repo) CountBySense(ctx context.Context, senseID uuid.UUID) (int, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	count, err := q.CountBySense(ctx, senseID)
	if err != nil {
		return 0, fmt.Errorf("count examples: %w", err)
	}

	return int(count), nil
}

// ---------------------------------------------------------------------------
// Write operations
// ---------------------------------------------------------------------------

// CreateFromRef creates an example linked to a reference example. Fields
// sentence/translation stay NULL — COALESCE picks up ref values.
// Position is auto-calculated as MAX(position)+1.
func (r *Repo) CreateFromRef(ctx context.Context, senseID, refExampleID uuid.UUID, sourceSlug string) (*domain.Example, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	now := time.Now().UTC().Truncate(time.Microsecond)
	row, err := q.CreateExampleFromRef(ctx, sqlc.CreateExampleFromRefParams{
		ID:           uuid.New(),
		SenseID:      senseID,
		RefExampleID: pgtype.UUID{Bytes: refExampleID, Valid: true},
		SourceSlug:   sourceSlug,
		CreatedAt:    now,
	})
	if err != nil {
		return nil, mapError(err, "example", uuid.Nil)
	}

	// The sqlc RETURNING gives raw values (without COALESCE). Re-read with COALESCE.
	return r.GetByID(ctx, row.ID)
}

// CreateCustom creates a custom example with user-provided fields (no ref link).
// Position is auto-calculated as MAX(position)+1.
func (r *Repo) CreateCustom(ctx context.Context, senseID uuid.UUID, sentence string, translation *string, sourceSlug string) (*domain.Example, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	now := time.Now().UTC().Truncate(time.Microsecond)
	row, err := q.CreateExampleCustom(ctx, sqlc.CreateExampleCustomParams{
		ID:          uuid.New(),
		SenseID:     senseID,
		Sentence:    pgtype.Text{String: sentence, Valid: true},
		Translation: ptrStringToPgText(translation),
		SourceSlug:  sourceSlug,
		CreatedAt:   now,
	})
	if err != nil {
		return nil, mapError(err, "example", uuid.Nil)
	}

	e := toDomainExample(row)
	return &e, nil
}

// Update modifies example fields. ref_example_id is NOT touched (preserves origin link).
// Returns the example with COALESCE-resolved fields.
func (r *Repo) Update(ctx context.Context, exampleID uuid.UUID, sentence string, translation *string) (*domain.Example, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	_, err := q.UpdateExample(ctx, sqlc.UpdateExampleParams{
		ID:          exampleID,
		Sentence:    pgtype.Text{String: sentence, Valid: true},
		Translation: ptrStringToPgText(translation),
	})
	if err != nil {
		return nil, mapError(err, "example", exampleID)
	}

	// Re-read with COALESCE to return resolved values.
	return r.GetByID(ctx, exampleID)
}

// Delete removes an example by ID. Returns domain.ErrNotFound if 0 rows affected.
func (r *Repo) Delete(ctx context.Context, exampleID uuid.UUID) error {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	n, err := q.DeleteExample(ctx, exampleID)
	if err != nil {
		return mapError(err, "example", exampleID)
	}
	if n == 0 {
		return fmt.Errorf("example %s: %w", exampleID, domain.ErrNotFound)
	}

	return nil
}

// Reorder updates positions for a batch of examples atomically within a transaction.
func (r *Repo) Reorder(ctx context.Context, items []domain.ReorderItem) error {
	if len(items) == 0 {
		return nil
	}

	return r.txm.RunInTx(ctx, func(txCtx context.Context) error {
		q := sqlc.New(postgres.QuerierFromCtx(txCtx, r.pool))

		for _, item := range items {
			if err := q.UpdateExamplePosition(txCtx, sqlc.UpdateExamplePositionParams{
				ID:       item.ID,
				Position: int32(item.Position),
			}); err != nil {
				return mapError(err, "example", item.ID)
			}
		}

		return nil
	})
}

// ---------------------------------------------------------------------------
// Row scanning helpers
// ---------------------------------------------------------------------------

// scanExamples scans multiple rows from a COALESCE query into domain.Example slices.
func scanExamples(rows pgx.Rows) ([]domain.Example, error) {
	var examples []domain.Example
	for rows.Next() {
		e, err := scanExampleFromRows(rows)
		if err != nil {
			return nil, err
		}
		examples = append(examples, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if examples == nil {
		examples = []domain.Example{}
	}

	return examples, nil
}

// scanExampleFromRows scans a single row from pgx.Rows into a domain.Example.
func scanExampleFromRows(rows pgx.Rows) (domain.Example, error) {
	var (
		id           uuid.UUID
		senseID      uuid.UUID
		sentence     pgtype.Text
		translation  pgtype.Text
		sourceSlug   string
		position     int32
		refExampleID pgtype.UUID
		createdAt    time.Time
	)

	if err := rows.Scan(&id, &senseID, &sentence, &translation, &sourceSlug, &position, &refExampleID, &createdAt); err != nil {
		return domain.Example{}, err
	}

	return buildDomainExample(id, senseID, sentence, translation, sourceSlug, position, refExampleID, createdAt), nil
}

// scanExampleRow scans a single pgx.Row into a domain.Example.
func scanExampleRow(row pgx.Row) (domain.Example, error) {
	var (
		id           uuid.UUID
		senseID      uuid.UUID
		sentence     pgtype.Text
		translation  pgtype.Text
		sourceSlug   string
		position     int32
		refExampleID pgtype.UUID
		createdAt    time.Time
	)

	if err := row.Scan(&id, &senseID, &sentence, &translation, &sourceSlug, &position, &refExampleID, &createdAt); err != nil {
		return domain.Example{}, err
	}

	return buildDomainExample(id, senseID, sentence, translation, sourceSlug, position, refExampleID, createdAt), nil
}

// buildDomainExample constructs a domain.Example from scanned values.
func buildDomainExample(id, senseID uuid.UUID, sentence, translation pgtype.Text, sourceSlug string, position int32, refExampleID pgtype.UUID, createdAt time.Time) domain.Example {
	e := domain.Example{
		ID:         id,
		SenseID:    senseID,
		SourceSlug: sourceSlug,
		Position:   int(position),
		CreatedAt:  createdAt,
	}

	if sentence.Valid {
		e.Sentence = &sentence.String
	}

	if translation.Valid {
		e.Translation = &translation.String
	}

	if refExampleID.Valid {
		rid := uuid.UUID(refExampleID.Bytes)
		e.RefExampleID = &rid
	}

	return e
}

// ---------------------------------------------------------------------------
// Mapping helpers: sqlc -> domain (for write query results)
// ---------------------------------------------------------------------------

// toDomainExample converts a sqlc.Example to a domain.Example (raw values, no COALESCE).
func toDomainExample(row sqlc.Example) domain.Example {
	e := domain.Example{
		ID:         row.ID,
		SenseID:    row.SenseID,
		SourceSlug: row.SourceSlug,
		Position:   int(row.Position),
		CreatedAt:  row.CreatedAt,
	}

	if row.Sentence.Valid {
		e.Sentence = &row.Sentence.String
	}

	if row.Translation.Valid {
		e.Translation = &row.Translation.String
	}

	if row.RefExampleID.Valid {
		rid := uuid.UUID(row.RefExampleID.Bytes)
		e.RefExampleID = &rid
	}

	return e
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

// ---------------------------------------------------------------------------
// pgtype helpers
// ---------------------------------------------------------------------------

// ptrStringToPgText converts a *string to pgtype.Text (nil -> NULL).
func ptrStringToPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}
