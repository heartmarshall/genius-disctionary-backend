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
    date_trunc('day', rl.reviewed_at)::date AS review_date,
    count(*) AS review_count
FROM review_logs rl
JOIN cards c ON rl.card_id = c.id
WHERE c.user_id = $1 AND rl.reviewed_at >= $2
GROUP BY review_date
ORDER BY review_date DESC
LIMIT $3`

const getByCardIDsSQL = `
SELECT rl.id, rl.card_id, rl.grade, rl.prev_state, rl.duration_ms, rl.reviewed_at
FROM review_logs rl
WHERE rl.card_id = ANY($1::uuid[])
ORDER BY rl.card_id, rl.reviewed_at DESC`

const countByCardIDSQL = `SELECT count(*) FROM review_logs WHERE card_id = $1`

// countNewTodaySQL depends on the JSON key "state" in cardSnapshotJSON.
// If you rename cardSnapshotJSON.State's json tag, update this query too.
const countNewTodaySQL = `
SELECT count(*) FROM review_logs rl
JOIN cards c ON rl.card_id = c.id
WHERE c.user_id = $1 AND rl.reviewed_at >= $2
AND rl.prev_state IS NOT NULL
AND rl.prev_state->>'state' = 'NEW'`

const getStatsByCardIDSQL = `
SELECT
    count(*) AS total,
    count(*) FILTER (WHERE grade = 'AGAIN') AS again_count,
    count(*) FILTER (WHERE grade = 'HARD') AS hard_count,
    count(*) FILTER (WHERE grade = 'GOOD') AS good_count,
    count(*) FILTER (WHERE grade = 'EASY') AS easy_count,
    avg(duration_ms) FILTER (WHERE duration_ms IS NOT NULL) AS avg_duration_ms
FROM review_logs
WHERE card_id = $1`

const getByPeriodSQL = `
SELECT rl.id, rl.card_id, rl.grade, rl.prev_state, rl.duration_ms, rl.reviewed_at
FROM review_logs rl
JOIN cards c ON rl.card_id = c.id
WHERE c.user_id = $1 AND rl.reviewed_at >= $2 AND rl.reviewed_at <= $3
ORDER BY rl.reviewed_at DESC`

// ---------------------------------------------------------------------------
// Read operations
// ---------------------------------------------------------------------------

// GetByCardID returns review logs for a card, ordered by reviewed_at DESC,
// with limit/offset pagination. Returns logs, total count, and error.
func (r *Repo) GetByCardID(ctx context.Context, cardID uuid.UUID, limit, offset int) ([]*domain.ReviewLog, int, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	// Count total
	var total int
	if err := querier.QueryRow(ctx, countByCardIDSQL, cardID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count review_logs by card_id: %w", err)
	}

	// limit=0 means "no limit" — use a large value for SQL LIMIT
	effectiveLimit := limit
	if effectiveLimit <= 0 {
		effectiveLimit = 2147483647
	}

	q := sqlc.New(querier)
	rows, err := q.GetByCardID(ctx, sqlc.GetByCardIDParams{
		CardID: cardID,
		Lim:    int32(effectiveLimit),
		Off:    int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("get review_logs by card_id: %w", err)
	}

	logs := make([]*domain.ReviewLog, len(rows))
	for i, row := range rows {
		rl, err := toDomainReviewLog(row)
		if err != nil {
			return nil, 0, err
		}
		logs[i] = &rl
	}

	return logs, total, nil
}

// GetLastByCardID returns the most recent review log for a card.
// Returns domain.ErrNotFound if no review logs exist for the card.
func (r *Repo) GetLastByCardID(ctx context.Context, cardID uuid.UUID) (*domain.ReviewLog, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.GetLastByCardID(ctx, cardID)
	if err != nil {
		return nil, mapError(err, "review_log", cardID)
	}

	rl, err := toDomainReviewLog(row)
	if err != nil {
		return nil, err
	}

	return &rl, nil
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

// GetStreakDays returns daily review counts grouped by day,
// ordered by date DESC, limited to `lastNDays` entries.
// dayStart is the start of the current day; the query goes back lastNDays from it.
func (r *Repo) GetStreakDays(ctx context.Context, userID uuid.UUID, dayStart time.Time, lastNDays int) ([]domain.DayReviewCount, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	// dayStart is already the start of the first day. We go back lastNDays from dayStart.
	from := dayStart.AddDate(0, 0, -lastNDays)

	rows, err := querier.Query(ctx, getStreakDaysSQL, userID, from, lastNDays)
	if err != nil {
		return nil, fmt.Errorf("get streak days: %w", err)
	}
	defer rows.Close()

	var counts []domain.DayReviewCount
	for rows.Next() {
		var dc domain.DayReviewCount
		if err := rows.Scan(&dc.Date, &dc.Count); err != nil {
			return nil, fmt.Errorf("scan streak day: %w", err)
		}
		counts = append(counts, dc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate streak days: %w", err)
	}

	if counts == nil {
		counts = []domain.DayReviewCount{}
	}

	return counts, nil
}

// ---------------------------------------------------------------------------
// Write operations
// ---------------------------------------------------------------------------

// Create inserts a new review log and returns the persisted domain.ReviewLog.
func (r *Repo) Create(ctx context.Context, rl *domain.ReviewLog) (*domain.ReviewLog, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	prevStateBytes, err := marshalPrevState(rl.PrevState)
	if err != nil {
		return nil, fmt.Errorf("review_log marshal prev_state: %w", err)
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
		return nil, mapError(err, "review_log", rl.ID)
	}

	result, err := toDomainReviewLog(row)
	if err != nil {
		return nil, err
	}

	return &result, nil
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

// CountNewToday returns the count of reviews for NEW-status cards since dayStart.
func (r *Repo) CountNewToday(ctx context.Context, userID uuid.UUID, dayStart time.Time) (int, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	var count int
	if err := querier.QueryRow(ctx, countNewTodaySQL, userID, dayStart).Scan(&count); err != nil {
		return 0, fmt.Errorf("count new today reviews: %w", err)
	}

	return count, nil
}

// GetStatsByCardID returns aggregated review statistics for a card,
// computed entirely in SQL (no loading of individual rows).
func (r *Repo) GetStatsByCardID(ctx context.Context, cardID uuid.UUID) (domain.ReviewLogAggregation, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)
	var stats domain.ReviewLogAggregation
	var avgDur *float64
	err := querier.QueryRow(ctx, getStatsByCardIDSQL, cardID).Scan(
		&stats.TotalReviews, &stats.AgainCount, &stats.HardCount,
		&stats.GoodCount, &stats.EasyCount, &avgDur,
	)
	if err != nil {
		return domain.ReviewLogAggregation{}, fmt.Errorf("get stats by card_id: %w", err)
	}
	if avgDur != nil {
		v := int(*avgDur)
		stats.AvgDurationMs = &v
	}
	return stats, nil
}

// GetByPeriod returns review logs for a user within a time range,
// ordered by reviewed_at DESC.
func (r *Repo) GetByPeriod(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]*domain.ReviewLog, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	rows, err := querier.Query(ctx, getByPeriodSQL, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("get review_logs by period: %w", err)
	}
	defer rows.Close()

	var logs []*domain.ReviewLog
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

		rl := &domain.ReviewLog{
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

		logs = append(logs, rl)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate review_logs: %w", err)
	}

	if logs == nil {
		logs = []*domain.ReviewLog{}
	}

	return logs, nil
}

// ---------------------------------------------------------------------------
// JSONB serialization helpers for CardSnapshot (prev_state)
// ---------------------------------------------------------------------------

// cardSnapshotJSON is an intermediate struct for JSON marshaling of domain.CardSnapshot.
// Domain CardSnapshot has no json tags, so the repo layer handles serialization.
type cardSnapshotJSON struct {
	State         string  `json:"state"`
	Step          int     `json:"step"`
	Stability     float64 `json:"stability"`
	Difficulty    float64 `json:"difficulty"`
	Due           string  `json:"due"`
	LastReview    *string `json:"last_review,omitempty"`
	Reps          int     `json:"reps"`
	Lapses        int     `json:"lapses"`
	ScheduledDays int     `json:"scheduled_days"`
	ElapsedDays   int     `json:"elapsed_days"`
}

// marshalPrevState converts a *domain.CardSnapshot to JSON bytes for JSONB storage.
// Returns nil for nil input (stored as NULL in DB).
func marshalPrevState(cs *domain.CardSnapshot) ([]byte, error) {
	if cs == nil {
		return nil, nil
	}

	j := cardSnapshotJSON{
		State:         string(cs.State),
		Step:          cs.Step,
		Stability:     cs.Stability,
		Difficulty:    cs.Difficulty,
		Due:           cs.Due.UTC().Format(time.RFC3339Nano),
		Reps:          cs.Reps,
		Lapses:        cs.Lapses,
		ScheduledDays: cs.ScheduledDays,
		ElapsedDays:   cs.ElapsedDays,
	}

	if cs.LastReview != nil {
		s := cs.LastReview.UTC().Format(time.RFC3339Nano)
		j.LastReview = &s
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

	due, err := time.Parse(time.RFC3339Nano, j.Due)
	if err != nil {
		return nil, fmt.Errorf("parse due: %w", err)
	}

	cs := &domain.CardSnapshot{
		State:         domain.CardState(j.State),
		Step:          j.Step,
		Stability:     j.Stability,
		Difficulty:    j.Difficulty,
		Due:           due,
		Reps:          j.Reps,
		Lapses:        j.Lapses,
		ScheduledDays: j.ScheduledDays,
		ElapsedDays:   j.ElapsedDays,
	}

	if j.LastReview != nil {
		t, err := time.Parse(time.RFC3339Nano, *j.LastReview)
		if err != nil {
			return nil, fmt.Errorf("parse last_review: %w", err)
		}
		cs.LastReview = &t
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
