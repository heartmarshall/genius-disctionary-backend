// Package session implements the StudySession repository using PostgreSQL.
// All queries use raw SQL (no sqlc) since the result column is JSONB requiring
// custom marshal/unmarshal logic.
package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	postgres "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// Repo provides study session persistence backed by PostgreSQL.
type Repo struct {
	pool *pgxpool.Pool
}

// New creates a new session repository.
func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// ---------------------------------------------------------------------------
// SQL constants
// ---------------------------------------------------------------------------

const sessionColumns = `id, user_id, status, started_at, finished_at, result, created_at`

const createSQL = `
INSERT INTO study_sessions (id, user_id, status, started_at, created_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING ` + sessionColumns

const getByIDSQL = `
SELECT ` + sessionColumns + `
FROM study_sessions
WHERE id = $1 AND user_id = $2`

const getActiveSQL = `
SELECT ` + sessionColumns + `
FROM study_sessions
WHERE user_id = $1 AND status = 'ACTIVE'`

const finishSQL = `
UPDATE study_sessions
SET status = 'FINISHED', finished_at = now(), result = $3
WHERE id = $1 AND user_id = $2 AND status = 'ACTIVE'
RETURNING ` + sessionColumns

const abandonSQL = `
UPDATE study_sessions
SET status = 'ABANDONED', finished_at = now()
WHERE id = $1 AND user_id = $2 AND status = 'ACTIVE'`

const countByUserIDSQL = `
SELECT count(*) FROM study_sessions WHERE user_id = $1`

const getByUserIDSQL = `
SELECT ` + sessionColumns + `
FROM study_sessions
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3`

// ---------------------------------------------------------------------------
// Read operations
// ---------------------------------------------------------------------------

// GetByID returns a session by primary key filtered by user_id.
// Returns domain.ErrNotFound if the session does not exist or belongs to another user.
func (r *Repo) GetByID(ctx context.Context, userID, sessionID uuid.UUID) (*domain.StudySession, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	row := querier.QueryRow(ctx, getByIDSQL, sessionID, userID)

	session, err := scanSession(row)
	if err != nil {
		return nil, mapError(err, "session", sessionID)
	}

	return session, nil
}

// GetActive returns the current ACTIVE session for a user.
// Returns domain.ErrNotFound if no active session exists.
func (r *Repo) GetActive(ctx context.Context, userID uuid.UUID) (*domain.StudySession, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	row := querier.QueryRow(ctx, getActiveSQL, userID)

	session, err := scanSession(row)
	if err != nil {
		return nil, mapError(err, "session", uuid.Nil)
	}

	return session, nil
}

// GetByUserID returns sessions for a user with pagination (ordered by created_at DESC).
// Returns sessions, total count, and error.
func (r *Repo) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.StudySession, int, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	// Count total
	var total int
	if err := querier.QueryRow(ctx, countByUserIDSQL, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count sessions by user_id: %w", err)
	}

	// Fetch page
	rows, err := querier.Query(ctx, getByUserIDSQL, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("get sessions by user_id: %w", err)
	}
	defer rows.Close()

	sessions, err := scanSessions(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("get sessions by user_id: %w", err)
	}

	return sessions, total, nil
}

// ---------------------------------------------------------------------------
// Write operations
// ---------------------------------------------------------------------------

// Create inserts a new study session and returns the persisted domain.StudySession.
// A unique constraint ensures only one ACTIVE session per user; attempting to create
// a second active session results in domain.ErrAlreadyExists.
func (r *Repo) Create(ctx context.Context, session *domain.StudySession) (*domain.StudySession, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	now := time.Now().UTC().Truncate(time.Microsecond)
	startedAt := session.StartedAt.UTC().Truncate(time.Microsecond)

	row := querier.QueryRow(ctx, createSQL,
		session.ID,
		session.UserID,
		string(session.Status),
		startedAt,
		now,
	)

	created, err := scanSession(row)
	if err != nil {
		return nil, mapError(err, "session", session.ID)
	}

	return created, nil
}

// Finish completes an ACTIVE session by setting its status to FINISHED and storing the result.
// Returns domain.ErrNotFound if the session does not exist, belongs to another user, or is not ACTIVE.
func (r *Repo) Finish(ctx context.Context, userID, sessionID uuid.UUID, result domain.SessionResult) (*domain.StudySession, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	resultBytes, err := marshalResult(&result)
	if err != nil {
		return nil, fmt.Errorf("session %s: marshal result: %w", sessionID, err)
	}

	row := querier.QueryRow(ctx, finishSQL, sessionID, userID, resultBytes)

	finished, err := scanSession(row)
	if err != nil {
		return nil, mapError(err, "session", sessionID)
	}

	return finished, nil
}

// Abandon marks an ACTIVE session as ABANDONED.
// Returns domain.ErrNotFound if the session does not exist, belongs to another user, or is not ACTIVE.
func (r *Repo) Abandon(ctx context.Context, userID, sessionID uuid.UUID) error {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	ct, err := querier.Exec(ctx, abandonSQL, sessionID, userID)
	if err != nil {
		return mapError(err, "session", sessionID)
	}

	if ct.RowsAffected() == 0 {
		return fmt.Errorf("session %s: %w", sessionID, domain.ErrNotFound)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Row scanning helpers
// ---------------------------------------------------------------------------

// scanSession scans a single session row from pgx.Row.
func scanSession(row pgx.Row) (*domain.StudySession, error) {
	var (
		id         uuid.UUID
		userID     uuid.UUID
		status     string
		startedAt  time.Time
		finishedAt *time.Time
		resultJSON []byte
		createdAt  time.Time
	)

	if err := row.Scan(&id, &userID, &status, &startedAt, &finishedAt, &resultJSON, &createdAt); err != nil {
		return nil, err
	}

	session := &domain.StudySession{
		ID:         id,
		UserID:     userID,
		Status:     domain.SessionStatus(status),
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		CreatedAt:  createdAt,
	}

	result, err := unmarshalResult(resultJSON)
	if err != nil {
		return nil, fmt.Errorf("session %s: %w", id, err)
	}
	session.Result = result

	return session, nil
}

// scanSessions scans multiple session rows from pgx.Rows into a []*domain.StudySession slice.
func scanSessions(rows pgx.Rows) ([]*domain.StudySession, error) {
	var sessions []*domain.StudySession
	for rows.Next() {
		var (
			id         uuid.UUID
			userID     uuid.UUID
			status     string
			startedAt  time.Time
			finishedAt *time.Time
			resultJSON []byte
			createdAt  time.Time
		)

		if err := rows.Scan(&id, &userID, &status, &startedAt, &finishedAt, &resultJSON, &createdAt); err != nil {
			return nil, err
		}

		session := &domain.StudySession{
			ID:         id,
			UserID:     userID,
			Status:     domain.SessionStatus(status),
			StartedAt:  startedAt,
			FinishedAt: finishedAt,
			CreatedAt:  createdAt,
		}

		result, err := unmarshalResult(resultJSON)
		if err != nil {
			return nil, fmt.Errorf("session %s: %w", id, err)
		}
		session.Result = result

		sessions = append(sessions, session)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if sessions == nil {
		sessions = []*domain.StudySession{}
	}

	return sessions, nil
}

// ---------------------------------------------------------------------------
// JSONB serialization helpers for SessionResult
// ---------------------------------------------------------------------------

// sessionResultJSON is an intermediate struct for JSON marshaling of domain.SessionResult.
// Domain types have no json tags, so the repo layer handles serialization.
type sessionResultJSON struct {
	TotalReviewed int             `json:"total_reviewed"`
	NewReviewed   int             `json:"new_reviewed"`
	DueReviewed   int             `json:"due_reviewed"`
	GradeCounts   gradeCountsJSON `json:"grade_counts"`
	DurationMs    int64           `json:"duration_ms"`
	AccuracyRate  float64         `json:"accuracy_rate"`
}

type gradeCountsJSON struct {
	Again int `json:"again"`
	Hard  int `json:"hard"`
	Good  int `json:"good"`
	Easy  int `json:"easy"`
}

// marshalResult converts a *domain.SessionResult to JSON bytes for JSONB storage.
// Returns nil for nil input (stored as NULL in DB).
func marshalResult(r *domain.SessionResult) ([]byte, error) {
	if r == nil {
		return nil, nil
	}

	j := sessionResultJSON{
		TotalReviewed: r.TotalReviewed,
		NewReviewed:   r.NewReviewed,
		DueReviewed:   r.DueReviewed,
		GradeCounts: gradeCountsJSON{
			Again: r.GradeCounts.Again,
			Hard:  r.GradeCounts.Hard,
			Good:  r.GradeCounts.Good,
			Easy:  r.GradeCounts.Easy,
		},
		DurationMs:   r.DurationMs,
		AccuracyRate: r.AccuracyRate,
	}

	return json.Marshal(j)
}

// unmarshalResult converts JSON bytes from JSONB storage to a *domain.SessionResult.
// Returns nil for nil/empty input (NULL in DB).
func unmarshalResult(data []byte) (*domain.SessionResult, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var j sessionResultJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return nil, fmt.Errorf("unmarshal session result: %w", err)
	}

	return &domain.SessionResult{
		TotalReviewed: j.TotalReviewed,
		NewReviewed:   j.NewReviewed,
		DueReviewed:   j.DueReviewed,
		GradeCounts: domain.GradeCounts{
			Again: j.GradeCounts.Again,
			Hard:  j.GradeCounts.Hard,
			Good:  j.GradeCounts.Good,
			Easy:  j.GradeCounts.Easy,
		},
		DurationMs:   j.DurationMs,
		AccuracyRate: j.AccuracyRate,
	}, nil
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
