// Package sense implements the Sense repository using PostgreSQL.
// Read queries use raw SQL with COALESCE + LEFT JOIN ref_senses to resolve
// inherited fields. Write queries use sqlc-generated code.
package sense

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
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/sense/sqlc"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// Repo provides sense persistence backed by PostgreSQL.
type Repo struct {
	pool *pgxpool.Pool
	txm  *postgres.TxManager
}

// New creates a new sense repository.
func New(pool *pgxpool.Pool, txm *postgres.TxManager) *Repo {
	return &Repo{pool: pool, txm: txm}
}

// ---------------------------------------------------------------------------
// Raw SQL for COALESCE read queries
// ---------------------------------------------------------------------------

const getByEntryIDSQL = `
SELECT
    s.id, s.entry_id,
    COALESCE(s.definition, rs.definition) AS definition,
    COALESCE(s.part_of_speech, rs.part_of_speech) AS part_of_speech,
    COALESCE(s.cefr_level, rs.cefr_level) AS cefr_level,
    s.source_slug, s.position, s.ref_sense_id, s.created_at
FROM senses s
LEFT JOIN ref_senses rs ON s.ref_sense_id = rs.id
WHERE s.entry_id = $1
ORDER BY s.position`

const getByEntryIDsSQL = `
SELECT
    s.id, s.entry_id,
    COALESCE(s.definition, rs.definition) AS definition,
    COALESCE(s.part_of_speech, rs.part_of_speech) AS part_of_speech,
    COALESCE(s.cefr_level, rs.cefr_level) AS cefr_level,
    s.source_slug, s.position, s.ref_sense_id, s.created_at
FROM senses s
LEFT JOIN ref_senses rs ON s.ref_sense_id = rs.id
WHERE s.entry_id = ANY($1::uuid[])
ORDER BY s.entry_id, s.position`

const getByIDSQL = `
SELECT
    s.id, s.entry_id,
    COALESCE(s.definition, rs.definition) AS definition,
    COALESCE(s.part_of_speech, rs.part_of_speech) AS part_of_speech,
    COALESCE(s.cefr_level, rs.cefr_level) AS cefr_level,
    s.source_slug, s.position, s.ref_sense_id, s.created_at
FROM senses s
LEFT JOIN ref_senses rs ON s.ref_sense_id = rs.id
WHERE s.id = $1`

const getByIDForUserSQL = `
SELECT
    s.id, s.entry_id,
    COALESCE(s.definition, rs.definition) AS definition,
    COALESCE(s.part_of_speech, rs.part_of_speech) AS part_of_speech,
    COALESCE(s.cefr_level, rs.cefr_level) AS cefr_level,
    s.source_slug, s.position, s.ref_sense_id, s.created_at
FROM senses s
LEFT JOIN ref_senses rs ON s.ref_sense_id = rs.id
JOIN entries e ON e.id = s.entry_id
WHERE s.id = $1 AND e.user_id = $2 AND e.deleted_at IS NULL`

// ---------------------------------------------------------------------------
// Read operations
// ---------------------------------------------------------------------------

// GetByEntryID returns all senses for an entry with COALESCE-resolved fields,
// ordered by position.
func (r *Repo) GetByEntryID(ctx context.Context, entryID uuid.UUID) ([]domain.Sense, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	rows, err := querier.Query(ctx, getByEntryIDSQL, entryID)
	if err != nil {
		return nil, fmt.Errorf("get senses by entry_id: %w", err)
	}
	defer rows.Close()

	senses, err := scanSenses(rows)
	if err != nil {
		return nil, fmt.Errorf("get senses by entry_id: %w", err)
	}

	return senses, nil
}

// GetByEntryIDs returns senses for multiple entries (batch for DataLoader).
// Results include EntryID for grouping by the caller.
func (r *Repo) GetByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]domain.Sense, error) {
	if len(entryIDs) == 0 {
		return []domain.Sense{}, nil
	}

	querier := postgres.QuerierFromCtx(ctx, r.pool)

	rows, err := querier.Query(ctx, getByEntryIDsSQL, entryIDs)
	if err != nil {
		return nil, fmt.Errorf("get senses by entry_ids: %w", err)
	}
	defer rows.Close()

	senses, err := scanSenses(rows)
	if err != nil {
		return nil, fmt.Errorf("get senses by entry_ids: %w", err)
	}

	return senses, nil
}

// GetByID returns a single sense with COALESCE-resolved fields.
func (r *Repo) GetByID(ctx context.Context, senseID uuid.UUID) (*domain.Sense, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	row := querier.QueryRow(ctx, getByIDSQL, senseID)

	sense, err := scanSenseRow(row)
	if err != nil {
		return nil, mapError(err, "sense", senseID)
	}

	return &sense, nil
}

// GetByIDForUser returns a single sense with COALESCE-resolved fields,
// verifying that the parent entry belongs to the given user (single JOIN query).
// Returns domain.ErrNotFound if the sense does not exist or the entry is not owned.
func (r *Repo) GetByIDForUser(ctx context.Context, userID, senseID uuid.UUID) (*domain.Sense, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	row := querier.QueryRow(ctx, getByIDForUserSQL, senseID, userID)

	sense, err := scanSenseRow(row)
	if err != nil {
		return nil, mapError(err, "sense", senseID)
	}

	return &sense, nil
}

// CountByEntry returns the number of senses for an entry.
func (r *Repo) CountByEntry(ctx context.Context, entryID uuid.UUID) (int, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	count, err := q.CountByEntry(ctx, entryID)
	if err != nil {
		return 0, fmt.Errorf("count senses: %w", err)
	}

	return int(count), nil
}

// CountByEntryID is an alias for CountByEntry, satisfying the study.senseRepo interface.
func (r *Repo) CountByEntryID(ctx context.Context, entryID uuid.UUID) (int, error) {
	return r.CountByEntry(ctx, entryID)
}

// ---------------------------------------------------------------------------
// Write operations
// ---------------------------------------------------------------------------

// CreateFromRef creates a sense linked to a reference sense. Fields
// definition/part_of_speech/cefr_level stay NULL — COALESCE picks up ref values.
// Position is auto-calculated as MAX(position)+1.
func (r *Repo) CreateFromRef(ctx context.Context, entryID, refSenseID uuid.UUID, sourceSlug string) (*domain.Sense, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	now := time.Now().UTC().Truncate(time.Microsecond)
	row, err := q.CreateSenseFromRef(ctx, sqlc.CreateSenseFromRefParams{
		ID:         uuid.New(),
		EntryID:    entryID,
		RefSenseID: pgtype.UUID{Bytes: refSenseID, Valid: true},
		SourceSlug: sourceSlug,
		CreatedAt:  now,
	})
	if err != nil {
		return nil, mapError(err, "sense", uuid.Nil)
	}

	// The sqlc RETURNING gives raw values (without COALESCE). Re-read with COALESCE.
	return r.GetByID(ctx, row.ID)
}

// CreateCustom creates a custom sense with user-provided fields (no ref link).
// Position is auto-calculated as MAX(position)+1.
func (r *Repo) CreateCustom(ctx context.Context, entryID uuid.UUID, definition *string, partOfSpeech *domain.PartOfSpeech, cefrLevel *string, sourceSlug string) (*domain.Sense, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	now := time.Now().UTC().Truncate(time.Microsecond)
	row, err := q.CreateSenseCustom(ctx, sqlc.CreateSenseCustomParams{
		ID:           uuid.New(),
		EntryID:      entryID,
		Definition:   ptrStringToPgText(definition),
		PartOfSpeech: partOfSpeechToPg(partOfSpeech),
		CefrLevel:    ptrStringToPgText(cefrLevel),
		SourceSlug:   sourceSlug,
		CreatedAt:    now,
	})
	if err != nil {
		return nil, mapError(err, "sense", uuid.Nil)
	}

	s := toDomainSense(row)
	return &s, nil
}

// Update modifies sense fields. ref_sense_id is NOT touched (preserves origin link).
// Returns the sense with COALESCE-resolved fields.
func (r *Repo) Update(ctx context.Context, senseID uuid.UUID, definition *string, partOfSpeech *domain.PartOfSpeech, cefrLevel *string) (*domain.Sense, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	_, err := q.UpdateSense(ctx, sqlc.UpdateSenseParams{
		ID:           senseID,
		Definition:   ptrStringToPgText(definition),
		PartOfSpeech: partOfSpeechToPg(partOfSpeech),
		CefrLevel:    ptrStringToPgText(cefrLevel),
	})
	if err != nil {
		return nil, mapError(err, "sense", senseID)
	}

	// Re-read with COALESCE to return resolved values.
	return r.GetByID(ctx, senseID)
}

// Delete removes a sense by ID. Returns domain.ErrNotFound if 0 rows affected.
func (r *Repo) Delete(ctx context.Context, senseID uuid.UUID) error {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	n, err := q.DeleteSense(ctx, senseID)
	if err != nil {
		return mapError(err, "sense", senseID)
	}
	if n == 0 {
		return fmt.Errorf("sense %s: %w", senseID, domain.ErrNotFound)
	}

	return nil
}

// Reorder updates positions for a batch of senses atomically within a transaction.
func (r *Repo) Reorder(ctx context.Context, items []domain.ReorderItem) error {
	if len(items) == 0 {
		return nil
	}

	return r.txm.RunInTx(ctx, func(txCtx context.Context) error {
		q := sqlc.New(postgres.QuerierFromCtx(txCtx, r.pool))

		for _, item := range items {
			if err := q.UpdateSensePosition(txCtx, sqlc.UpdateSensePositionParams{
				ID:       item.ID,
				Position: int32(item.Position),
			}); err != nil {
				return mapError(err, "sense", item.ID)
			}
		}

		return nil
	})
}

// ---------------------------------------------------------------------------
// Row scanning helpers
// ---------------------------------------------------------------------------

// scanSenses scans multiple rows from a COALESCE query into domain.Sense slices.
func scanSenses(rows pgx.Rows) ([]domain.Sense, error) {
	var senses []domain.Sense
	for rows.Next() {
		s, err := scanSenseFromRows(rows)
		if err != nil {
			return nil, err
		}
		senses = append(senses, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if senses == nil {
		senses = []domain.Sense{}
	}

	return senses, nil
}

// scanSenseFromRows scans a single row from pgx.Rows into a domain.Sense.
func scanSenseFromRows(rows pgx.Rows) (domain.Sense, error) {
	var (
		id         uuid.UUID
		entryID    uuid.UUID
		definition pgtype.Text
		pos        pgtype.Text
		cefrLevel  pgtype.Text
		sourceSlug string
		position   int32
		refSenseID pgtype.UUID
		createdAt  time.Time
	)

	if err := rows.Scan(&id, &entryID, &definition, &pos, &cefrLevel, &sourceSlug, &position, &refSenseID, &createdAt); err != nil {
		return domain.Sense{}, err
	}

	return buildDomainSense(id, entryID, definition, pos, cefrLevel, sourceSlug, position, refSenseID, createdAt), nil
}

// scanSenseRow scans a single pgx.Row into a domain.Sense.
func scanSenseRow(row pgx.Row) (domain.Sense, error) {
	var (
		id         uuid.UUID
		entryID    uuid.UUID
		definition pgtype.Text
		pos        pgtype.Text
		cefrLevel  pgtype.Text
		sourceSlug string
		position   int32
		refSenseID pgtype.UUID
		createdAt  time.Time
	)

	if err := row.Scan(&id, &entryID, &definition, &pos, &cefrLevel, &sourceSlug, &position, &refSenseID, &createdAt); err != nil {
		return domain.Sense{}, err
	}

	return buildDomainSense(id, entryID, definition, pos, cefrLevel, sourceSlug, position, refSenseID, createdAt), nil
}

// buildDomainSense constructs a domain.Sense from scanned values.
func buildDomainSense(id, entryID uuid.UUID, definition, pos, cefrLevel pgtype.Text, sourceSlug string, position int32, refSenseID pgtype.UUID, createdAt time.Time) domain.Sense {
	s := domain.Sense{
		ID:         id,
		EntryID:    entryID,
		SourceSlug: sourceSlug,
		Position:   int(position),
		CreatedAt:  createdAt,
	}

	if definition.Valid {
		s.Definition = &definition.String
	}

	if pos.Valid {
		p := domain.PartOfSpeech(pos.String)
		s.PartOfSpeech = &p
	}

	if cefrLevel.Valid {
		s.CEFRLevel = &cefrLevel.String
	}

	if refSenseID.Valid {
		rid := uuid.UUID(refSenseID.Bytes)
		s.RefSenseID = &rid
	}

	return s
}

// ---------------------------------------------------------------------------
// Mapping helpers: sqlc -> domain (for write query results)
// ---------------------------------------------------------------------------

// toDomainSense converts a sqlc.Sense to a domain.Sense (raw values, no COALESCE).
func toDomainSense(row sqlc.Sense) domain.Sense {
	s := domain.Sense{
		ID:         row.ID,
		EntryID:    row.EntryID,
		SourceSlug: row.SourceSlug,
		Position:   int(row.Position),
		CreatedAt:  row.CreatedAt,
	}

	if row.Definition.Valid {
		s.Definition = &row.Definition.String
	}

	if row.PartOfSpeech.Valid {
		p := domain.PartOfSpeech(row.PartOfSpeech.PartOfSpeech)
		s.PartOfSpeech = &p
	}

	if row.CefrLevel.Valid {
		s.CEFRLevel = &row.CefrLevel.String
	}

	if row.RefSenseID.Valid {
		rid := uuid.UUID(row.RefSenseID.Bytes)
		s.RefSenseID = &rid
	}

	return s
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

// partOfSpeechToPg converts a *domain.PartOfSpeech to sqlc NullPartOfSpeech.
func partOfSpeechToPg(p *domain.PartOfSpeech) sqlc.NullPartOfSpeech {
	if p == nil {
		return sqlc.NullPartOfSpeech{}
	}
	return sqlc.NullPartOfSpeech{
		PartOfSpeech: sqlc.PartOfSpeech(*p),
		Valid:        true,
	}
}
