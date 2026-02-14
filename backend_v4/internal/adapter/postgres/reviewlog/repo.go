// Package reviewlog implements the ReviewLog repository using PostgreSQL.
// Simple CRUD queries use sqlc; queries requiring JOINs (CountToday,
// GetStreakDays, GetByCardIDs) use raw SQL.
package reviewlog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	postgres "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/reviewlog/sqlc"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// DayReviewCount holds the date and number of reviews for that day.
type DayReviewCount struct {
	Date  time.Time // date only (midnight)
	Count int
}

// ReviewLogWithCardID wraps a domain.ReviewLog with its CardID for batch queries.
type ReviewLogWithCardID struct {
	CardID uuid.UUID
	domain.ReviewLog
}

// Repo provides review log persistence backed by PostgreSQL.
type Repo struct {
	pool *pgxpool.Pool
}

// New creates a new review log repository.
func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// ---------------------------------------------------------------------------
// Raw SQL for complex queries requiring JOINs
// ---------------------------------------------------------------------------

const countTodaySQL = `
SELECT count(*) FROM review_logs rl
JOIN cards c ON rl.card_id = c.id
WHERE c.user_id = $1 AND rl.reviewed_at >= $2`

const getStreakDaysSQL = `
SELECT
    date_trunc('day', rl.reviewed_at AT TIME ZONE $2)::date AS review_date,
    count(*) AS review_count
FROM review_logs rl
JOIN cards c ON rl.card_id = c.id
WHERE c.user_id = $1
GROUP BY review_date
ORDER BY review_date DESC
LIMIT $3`

const getByCardIDsSQL = `
SELECT rl.id, rl.card_id, rl.grade, rl.prev_state, rl.duration_ms, rl.reviewed_at
FROM review_logs rl
WHERE rl.card_id = ANY($1::uuid[])
ORDER BY rl.card_id, rl.reviewed_at DESC`

// ---------------------------------------------------------------------------
// Read operations
// ---------------------------------------------------------------------------

// GetByCardID returns review logs for a card, ordered by reviewed_at DESC,
// with limit/offset pagination.
func (r *Repo) GetByCardID(ctx context.Context, cardID uuid.UUID, limit, offset int) ([]domain.ReviewLog, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	rows, err := q.GetByCardID(ctx, sqlc.GetByCardIDParams{
		CardID: cardID,
		Lim:    int32(limit),
		Off:    int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("get review_logs by card_id: %w", err)
	}

	logs := make([]domain.ReviewLog, len(rows))
	for i, row := range rows {
		rl, err := toDomainReviewLog(row)
		if err != nil {
			return nil, err
		}
		logs[i] = rl
	}

	return logs, nil
}

// GetLastByCardID returns the most recent review log for a card.
// Returns domain.ErrNotFound if no review logs exist for the card.
func (r *Repo) GetLastByCardID(ctx context.Context, cardID uuid.UUID) (domain.ReviewLog, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.GetLastByCardID(ctx, cardID)
	if err != nil {
		return domain.ReviewLog{}, mapError(err, "review_log", cardID)
	}

	return toDomainReviewLog(row)
}

// GetByCardIDs returns review logs for multiple cards (batch for DataLoader).
// Results include CardID for grouping by the caller.
func (r *Repo) GetByCardIDs(ctx context.Context, cardIDs []uuid.UUID) ([]ReviewLogWithCardID, error) {
	if len(cardIDs) == 0 {
		return []ReviewLogWithCardID{}, nil
	}

	querier := postgres.QuerierFromCtx(ctx, r.pool)

	rows, err := querier.Query(ctx, getByCardIDsSQL, cardIDs)
	if err != nil {
		return nil, fmt.Errorf("get review_logs by card_ids: %w", err)
	}
	defer rows.Close()

	var logs []ReviewLogWithCardID
	for rows.Next() {
		var (
			id         uuid.UUID
			cardID     uuid.UUID
			grade      string
			prevState  []byte
			durationMs pgtype.Int4
			reviewedAt time.Time
		)

		if err := rows.Scan(&id, &cardID, &grade, &prevState, &durationMs, &reviewedAt); err != nil {
			return nil, fmt.Errorf("scan review_log: %w", err)
		}

		rl := domain.ReviewLog{
			ID:         id,
			CardID:     cardID,
			Grade:      domain.ReviewGrade(grade),
			ReviewedAt: reviewedAt,
		}

		if durationMs.Valid {
			d := int(durationMs.Int32)
			rl.DurationMs = &d
		}

		ps, err := unmarshalPrevState(prevState)
		if err != nil {
			return nil, fmt.Errorf("review_log %s: %w", id, err)
		}
		rl.PrevState = ps

		logs = append(logs, ReviewLogWithCardID{
			CardID:    cardID,
			ReviewLog: rl,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate review_logs: %w", err)
	}

	if logs == nil {
		logs = []ReviewLogWithCardID{}
	}

	return logs, nil
}

// CountToday returns the count of reviews for a user since dayStart.
// dayStart is already in UTC.
func (r *Repo) CountToday(ctx context.Context, userID uuid.UUID, dayStart time.Time) (int, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	var count int
	if err := querier.QueryRow(ctx, countTodaySQL, userID, dayStart).Scan(&count); err != nil {
		return 0, fmt.Errorf("count today reviews: %w", err)
	}

	return count, nil
}

// GetStreakDays returns daily review counts grouped by day in the given timezone,
// ordered by date DESC, limited to `days` entries.
func (r *Repo) GetStreakDays(ctx context.Context, userID uuid.UUID, timezone string, days int) ([]DayReviewCount, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	rows, err := querier.Query(ctx, getStreakDaysSQL, userID, timezone, days)
	if err != nil {
		return nil, fmt.Errorf("get streak days: %w", err)
	}
	defer rows.Close()

	var counts []DayReviewCount
	for rows.Next() {
		var dc DayReviewCount
		if err := rows.Scan(&dc.Date, &dc.Count); err != nil {
			return nil, fmt.Errorf("scan streak day: %w", err)
		}
		counts = append(counts, dc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate streak days: %w", err)
	}

	if counts == nil {
		counts = []DayReviewCount{}
	}

	return counts, nil
}

// ---------------------------------------------------------------------------
// Write operations
// ---------------------------------------------------------------------------

// Create inserts a new review log and returns the persisted domain.ReviewLog.
func (r *Repo) Create(ctx context.Context, rl domain.ReviewLog) (domain.ReviewLog, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	prevStateBytes, err := marshalPrevState(rl.PrevState)
	if err != nil {
		return domain.ReviewLog{}, fmt.Errorf("review_log marshal prev_state: %w", err)
	}

	var durationMs pgtype.Int4
	if rl.DurationMs != nil {
		durationMs = pgtype.Int4{Int32: int32(*rl.DurationMs), Valid: true}
	}

	row, err := q.CreateReviewLog(ctx, sqlc.CreateReviewLogParams{
		ID:         rl.ID,
		CardID:     rl.CardID,
		Grade:      sqlc.ReviewGrade(rl.Grade),
		PrevState:  prevStateBytes,
		DurationMs: durationMs,
		ReviewedAt: rl.ReviewedAt,
	})
	if err != nil {
		return domain.ReviewLog{}, mapError(err, "review_log", rl.ID)
	}

	return toDomainReviewLog(row)
}

// Delete removes a review log by ID.
// Returns domain.ErrNotFound if 0 rows affected.
func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	rowsAffected, err := q.DeleteReviewLog(ctx, id)
	if err != nil {
		return mapError(err, "review_log", id)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("review_log %s: %w", id, domain.ErrNotFound)
	}

	return nil
}

// ---------------------------------------------------------------------------
// JSONB serialization helpers for CardSnapshot (prev_state)
// ---------------------------------------------------------------------------

// cardSnapshotJSON is an intermediate struct for JSON marshaling of domain.CardSnapshot.
// Domain CardSnapshot has no json tags, so the repo layer handles serialization.
type cardSnapshotJSON struct {
	Status       string  `json:"status"`
	LearningStep int     `json:"learning_step"`
	IntervalDays int     `json:"interval_days"`
	EaseFactor   float64 `json:"ease_factor"`
	NextReviewAt *string `json:"next_review_at,omitempty"`
}

// marshalPrevState converts a *domain.CardSnapshot to JSON bytes for JSONB storage.
// Returns nil for nil input (stored as NULL in DB).
func marshalPrevState(cs *domain.CardSnapshot) ([]byte, error) {
	if cs == nil {
		return nil, nil
	}

	j := cardSnapshotJSON{
		Status:       string(cs.Status),
		LearningStep: cs.LearningStep,
		IntervalDays: cs.IntervalDays,
		EaseFactor:   cs.EaseFactor,
	}

	if cs.NextReviewAt != nil {
		s := cs.NextReviewAt.UTC().Format(time.RFC3339Nano)
		j.NextReviewAt = &s
	}

	return json.Marshal(j)
}

// unmarshalPrevState converts JSON bytes from JSONB storage to a *domain.CardSnapshot.
// Returns nil for nil/empty input (NULL in DB).
func unmarshalPrevState(data []byte) (*domain.CardSnapshot, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var j cardSnapshotJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return nil, fmt.Errorf("unmarshal prev_state: %w", err)
	}

	cs := &domain.CardSnapshot{
		Status:       domain.LearningStatus(j.Status),
		LearningStep: j.LearningStep,
		IntervalDays: j.IntervalDays,
		EaseFactor:   j.EaseFactor,
	}

	if j.NextReviewAt != nil {
		t, err := time.Parse(time.RFC3339Nano, *j.NextReviewAt)
		if err != nil {
			return nil, fmt.Errorf("parse next_review_at: %w", err)
		}
		cs.NextReviewAt = &t
	}

	return cs, nil
}

// ---------------------------------------------------------------------------
// Mapping helpers: sqlc -> domain
// ---------------------------------------------------------------------------

// toDomainReviewLog converts a sqlc.ReviewLog row into a domain.ReviewLog.
func toDomainReviewLog(row sqlc.ReviewLog) (domain.ReviewLog, error) {
	rl := domain.ReviewLog{
		ID:         row.ID,
		CardID:     row.CardID,
		Grade:      domain.ReviewGrade(row.Grade),
		ReviewedAt: row.ReviewedAt,
	}

	if row.DurationMs.Valid {
		d := int(row.DurationMs.Int32)
		rl.DurationMs = &d
	}

	ps, err := unmarshalPrevState(row.PrevState)
	if err != nil {
		return domain.ReviewLog{}, fmt.Errorf("review_log %s: %w", row.ID, err)
	}
	rl.PrevState = ps

	return rl, nil
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
