// Package pronunciation implements the Pronunciation M2M repository using PostgreSQL.
// It links ref_pronunciations to entries via the entry_pronunciations join table.
// Write queries use sqlc-generated code. Read queries use raw SQL with JOIN.
package pronunciation

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
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/pronunciation/sqlc"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// PronunciationWithEntryID is the batch result type for GetByEntryIDs.
// It embeds domain.RefPronunciation and adds EntryID for grouping by the caller.
type PronunciationWithEntryID struct {
	EntryID uuid.UUID
	domain.RefPronunciation
}

// Repo provides pronunciation M2M persistence backed by PostgreSQL.
type Repo struct {
	pool *pgxpool.Pool
}

// New creates a new pronunciation repository.
func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// ---------------------------------------------------------------------------
// Raw SQL for JOIN read queries
// ---------------------------------------------------------------------------

const getByEntryIDSQL = `
SELECT
    rp.id, rp.ref_entry_id, rp.transcription, rp.audio_url, rp.region, rp.source_slug
FROM entry_pronunciations ep
JOIN ref_pronunciations rp ON ep.ref_pronunciation_id = rp.id
WHERE ep.entry_id = $1`

const getByEntryIDsSQL = `
SELECT
    ep.entry_id,
    rp.id, rp.ref_entry_id, rp.transcription, rp.audio_url, rp.region, rp.source_slug
FROM entry_pronunciations ep
JOIN ref_pronunciations rp ON ep.ref_pronunciation_id = rp.id
WHERE ep.entry_id = ANY($1::uuid[])
ORDER BY ep.entry_id`

// ---------------------------------------------------------------------------
// Read operations
// ---------------------------------------------------------------------------

// GetByEntryID returns all ref_pronunciations linked to an entry via the M2M table.
// Returns an empty slice (not nil) when no pronunciations are linked.
func (r *Repo) GetByEntryID(ctx context.Context, entryID uuid.UUID) ([]domain.RefPronunciation, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	rows, err := querier.Query(ctx, getByEntryIDSQL, entryID)
	if err != nil {
		return nil, fmt.Errorf("get pronunciations by entry_id: %w", err)
	}
	defer rows.Close()

	result, err := scanPronunciations(rows)
	if err != nil {
		return nil, fmt.Errorf("get pronunciations by entry_id: %w", err)
	}

	return result, nil
}

// GetByEntryIDs returns pronunciations for multiple entries (batch for DataLoader).
// Results include EntryID for grouping by the caller.
func (r *Repo) GetByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]PronunciationWithEntryID, error) {
	if len(entryIDs) == 0 {
		return []PronunciationWithEntryID{}, nil
	}

	querier := postgres.QuerierFromCtx(ctx, r.pool)

	rows, err := querier.Query(ctx, getByEntryIDsSQL, entryIDs)
	if err != nil {
		return nil, fmt.Errorf("get pronunciations by entry_ids: %w", err)
	}
	defer rows.Close()

	result, err := scanPronunciationsWithEntryID(rows)
	if err != nil {
		return nil, fmt.Errorf("get pronunciations by entry_ids: %w", err)
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// Write operations
// ---------------------------------------------------------------------------

// Link creates an M2M link between an entry and a ref_pronunciation.
// Idempotent: linking the same pair twice is NOT an error (ON CONFLICT DO NOTHING).
func (r *Repo) Link(ctx context.Context, entryID, refPronunciationID uuid.UUID) error {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	err := q.LinkPronunciation(ctx, sqlc.LinkPronunciationParams{
		EntryID:            entryID,
		RefPronunciationID: refPronunciationID,
	})
	if err != nil {
		return mapError(err, "entry_pronunciation", entryID)
	}

	return nil
}

// Unlink removes the M2M link between an entry and a ref_pronunciation.
// Not an error if the link does not exist (0 rows affected is OK).
func (r *Repo) Unlink(ctx context.Context, entryID, refPronunciationID uuid.UUID) error {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	err := q.UnlinkPronunciation(ctx, sqlc.UnlinkPronunciationParams{
		EntryID:            entryID,
		RefPronunciationID: refPronunciationID,
	})
	if err != nil {
		return mapError(err, "entry_pronunciation", entryID)
	}

	return nil
}

// UnlinkAll removes all M2M links for an entry.
// Not an error if the entry has no links.
func (r *Repo) UnlinkAll(ctx context.Context, entryID uuid.UUID) error {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	err := q.UnlinkAllPronunciations(ctx, entryID)
	if err != nil {
		return mapError(err, "entry_pronunciation", entryID)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Row scanning helpers
// ---------------------------------------------------------------------------

// scanPronunciations scans multiple rows from GetByEntryID into domain.RefPronunciation slices.
func scanPronunciations(rows pgx.Rows) ([]domain.RefPronunciation, error) {
	var result []domain.RefPronunciation
	for rows.Next() {
		p, err := scanPronunciationFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if result == nil {
		result = []domain.RefPronunciation{}
	}

	return result, nil
}

// scanPronunciationFromRows scans a single row from pgx.Rows into a domain.RefPronunciation.
func scanPronunciationFromRows(rows pgx.Rows) (domain.RefPronunciation, error) {
	var (
		id            uuid.UUID
		refEntryID    uuid.UUID
		transcription string
		audioURL      pgtype.Text
		region        pgtype.Text
		sourceSlug    string
	)

	if err := rows.Scan(&id, &refEntryID, &transcription, &audioURL, &region, &sourceSlug); err != nil {
		return domain.RefPronunciation{}, err
	}

	return buildDomainPronunciation(id, refEntryID, transcription, audioURL, region, sourceSlug), nil
}

// scanPronunciationsWithEntryID scans multiple rows from GetByEntryIDs into PronunciationWithEntryID slices.
func scanPronunciationsWithEntryID(rows pgx.Rows) ([]PronunciationWithEntryID, error) {
	var result []PronunciationWithEntryID
	for rows.Next() {
		var (
			entryID       uuid.UUID
			id            uuid.UUID
			refEntryID    uuid.UUID
			transcription string
			audioURL      pgtype.Text
			region        pgtype.Text
			sourceSlug    string
		)

		if err := rows.Scan(&entryID, &id, &refEntryID, &transcription, &audioURL, &region, &sourceSlug); err != nil {
			return nil, err
		}

		result = append(result, PronunciationWithEntryID{
			EntryID:          entryID,
			RefPronunciation: buildDomainPronunciation(id, refEntryID, transcription, audioURL, region, sourceSlug),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if result == nil {
		result = []PronunciationWithEntryID{}
	}

	return result, nil
}

// buildDomainPronunciation constructs a domain.RefPronunciation from scanned values.
func buildDomainPronunciation(id, refEntryID uuid.UUID, transcription string, audioURL, region pgtype.Text, sourceSlug string) domain.RefPronunciation {
	p := domain.RefPronunciation{
		ID:         id,
		RefEntryID: refEntryID,
		SourceSlug: sourceSlug,
	}

	// transcription is NOT NULL in DB, but *string in domain.
	p.Transcription = &transcription

	if audioURL.Valid {
		p.AudioURL = &audioURL.String
	}

	if region.Valid {
		p.Region = &region.String
	}

	return p
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
