// Package card implements the Card repository using PostgreSQL.
// Simple CRUD queries use sqlc; complex queries with JOINs use raw SQL.
package card

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	postgres "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/card/sqlc"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// Repo provides card persistence backed by PostgreSQL.
type Repo struct {
	pool *pgxpool.Pool
}

// New creates a new card repository.
func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// ---------------------------------------------------------------------------
// FSRS column list used in raw SQL queries
// ---------------------------------------------------------------------------

const cardColumns = `c.id, c.user_id, c.entry_id, c.state, c.step, c.stability, c.difficulty,
       c.due, c.last_review, c.reps, c.lapses, c.scheduled_days, c.elapsed_days,
       c.created_at, c.updated_at`

// ---------------------------------------------------------------------------
// Raw SQL for complex queries requiring JOINs
// ---------------------------------------------------------------------------

var getDueCardsSQL = `
SELECT ` + cardColumns + `
FROM cards c
JOIN entries e ON c.entry_id = e.id
WHERE c.user_id = $1
  AND e.deleted_at IS NULL
  AND c.state IN ('LEARNING', 'RELEARNING', 'REVIEW')
  AND c.due <= $2
ORDER BY c.due ASC
LIMIT $3`

var getNewCardsSQL = `
SELECT ` + cardColumns + `
FROM cards c
JOIN entries e ON c.entry_id = e.id
WHERE c.user_id = $1 AND e.deleted_at IS NULL AND c.state = 'NEW'
ORDER BY c.created_at
LIMIT $2`

var countDueSQL = `
SELECT count(*) FROM cards c
JOIN entries e ON c.entry_id = e.id
WHERE c.user_id = $1 AND e.deleted_at IS NULL
  AND c.state IN ('LEARNING', 'RELEARNING', 'REVIEW')
  AND c.due <= $2`

var countNewSQL = `
SELECT count(*) FROM cards c
JOIN entries e ON c.entry_id = e.id
WHERE c.user_id = $1 AND e.deleted_at IS NULL AND c.state = 'NEW'`

var countByStatusSQL = `
SELECT c.state, count(*) as count
FROM cards c
JOIN entries e ON c.entry_id = e.id
WHERE c.user_id = $1 AND e.deleted_at IS NULL
GROUP BY c.state`

var countOverdueSQL = `
SELECT count(*) FROM cards c
JOIN entries e ON c.entry_id = e.id
WHERE c.user_id = $1 AND e.deleted_at IS NULL
  AND c.state IN ('LEARNING', 'RELEARNING', 'REVIEW')
  AND c.due < $2`

var getByEntryIDsSQL = `
SELECT ` + cardColumns + `
FROM cards c
WHERE c.entry_id = ANY($1::uuid[])`

const existsByEntryIDsSQL = `
SELECT entry_id FROM cards WHERE user_id = $1 AND entry_id = ANY($2::uuid[])`

// ---------------------------------------------------------------------------
// Read operations
// ---------------------------------------------------------------------------

// GetByID returns a card by primary key filtered by user_id.
func (r *Repo) GetByID(ctx context.Context, userID, cardID uuid.UUID) (*domain.Card, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.GetCardByID(ctx, sqlc.GetCardByIDParams{
		ID:     cardID,
		UserID: userID,
	})
	if err != nil {
		return nil, mapError(err, "card", cardID)
	}

	c := toDomainCard(fromGetByIDRow(row))
	return &c, nil
}

// GetByEntryID returns a card by entry_id filtered by user_id.
func (r *Repo) GetByEntryID(ctx context.Context, userID, entryID uuid.UUID) (*domain.Card, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.GetCardByEntryID(ctx, sqlc.GetCardByEntryIDParams{
		EntryID: entryID,
		UserID:  userID,
	})
	if err != nil {
		return nil, mapError(err, "card", uuid.Nil)
	}

	c := toDomainCard(fromGetByEntryIDRow(row))
	return &c, nil
}

// GetByEntryIDs returns cards for multiple entries (batch for DataLoader).
func (r *Repo) GetByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]domain.Card, error) {
	if len(entryIDs) == 0 {
		return []domain.Card{}, nil
	}

	querier := postgres.QuerierFromCtx(ctx, r.pool)

	rows, err := querier.Query(ctx, getByEntryIDsSQL, entryIDs)
	if err != nil {
		return nil, fmt.Errorf("get cards by entry_ids: %w", err)
	}
	defer rows.Close()

	cards, err := scanCards(rows)
	if err != nil {
		return nil, fmt.Errorf("get cards by entry_ids: %w", err)
	}

	return cards, nil
}

// GetDueCards returns cards that are due for review.
func (r *Repo) GetDueCards(ctx context.Context, userID uuid.UUID, now time.Time, limit int) ([]*domain.Card, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	rows, err := querier.Query(ctx, getDueCardsSQL, userID, now, limit)
	if err != nil {
		return nil, fmt.Errorf("get due cards: %w", err)
	}
	defer rows.Close()

	cards, err := scanCardPointers(rows)
	if err != nil {
		return nil, fmt.Errorf("get due cards: %w", err)
	}

	return cards, nil
}

// GetNewCards returns NEW cards ordered by creation time.
func (r *Repo) GetNewCards(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.Card, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	rows, err := querier.Query(ctx, getNewCardsSQL, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("get new cards: %w", err)
	}
	defer rows.Close()

	cards, err := scanCardPointers(rows)
	if err != nil {
		return nil, fmt.Errorf("get new cards: %w", err)
	}

	return cards, nil
}

// CountDue returns the count of cards due for review.
func (r *Repo) CountDue(ctx context.Context, userID uuid.UUID, now time.Time) (int, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	var count int
	if err := querier.QueryRow(ctx, countDueSQL, userID, now).Scan(&count); err != nil {
		return 0, fmt.Errorf("count due cards: %w", err)
	}

	return count, nil
}

// CountNew returns the count of NEW cards.
func (r *Repo) CountNew(ctx context.Context, userID uuid.UUID) (int, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	var count int
	if err := querier.QueryRow(ctx, countNewSQL, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count new cards: %w", err)
	}

	return count, nil
}

// CountByStatus returns card counts grouped by state.
func (r *Repo) CountByStatus(ctx context.Context, userID uuid.UUID) (domain.CardStatusCounts, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	rows, err := querier.Query(ctx, countByStatusSQL, userID)
	if err != nil {
		return domain.CardStatusCounts{}, fmt.Errorf("count cards by status: %w", err)
	}
	defer rows.Close()

	var counts domain.CardStatusCounts
	for rows.Next() {
		var state string
		var count int
		if err := rows.Scan(&state, &count); err != nil {
			return domain.CardStatusCounts{}, fmt.Errorf("scan status count: %w", err)
		}
		switch domain.CardState(state) {
		case domain.CardStateNew:
			counts.New = count
		case domain.CardStateLearning:
			counts.Learning = count
		case domain.CardStateReview:
			counts.Review = count
		case domain.CardStateRelearning:
			counts.Relearning = count
		}
		counts.Total += count
	}
	if err := rows.Err(); err != nil {
		return domain.CardStatusCounts{}, fmt.Errorf("iterate status counts: %w", err)
	}

	return counts, nil
}

// CountOverdue returns the count of cards that were due before dayStart (overdue by at least one full day).
func (r *Repo) CountOverdue(ctx context.Context, userID uuid.UUID, dayStart time.Time) (int, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	var count int
	if err := querier.QueryRow(ctx, countOverdueSQL, userID, dayStart).Scan(&count); err != nil {
		return 0, fmt.Errorf("count overdue cards: %w", err)
	}

	return count, nil
}

// ExistsByEntryIDs returns a map of entry IDs to whether a card exists for that entry.
func (r *Repo) ExistsByEntryIDs(ctx context.Context, userID uuid.UUID, entryIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	if len(entryIDs) == 0 {
		return map[uuid.UUID]bool{}, nil
	}

	querier := postgres.QuerierFromCtx(ctx, r.pool)

	rows, err := querier.Query(ctx, existsByEntryIDsSQL, userID, entryIDs)
	if err != nil {
		return nil, fmt.Errorf("exists by entry_ids: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID]bool, len(entryIDs))
	for rows.Next() {
		var entryID uuid.UUID
		if err := rows.Scan(&entryID); err != nil {
			return nil, fmt.Errorf("scan entry_id: %w", err)
		}
		result[entryID] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate entry_ids: %w", err)
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// Write operations
// ---------------------------------------------------------------------------

// Create inserts a new card with default FSRS state (NEW) and returns it.
func (r *Repo) Create(ctx context.Context, userID, entryID uuid.UUID) (*domain.Card, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	now := time.Now().UTC().Truncate(time.Microsecond)
	id := uuid.New()

	row, err := q.CreateCard(ctx, sqlc.CreateCardParams{
		ID:        id,
		UserID:    userID,
		EntryID:   entryID,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return nil, mapError(err, "card", id)
	}

	c := toDomainCard(fromCreateRow(row))
	return &c, nil
}

// UpdateSRS updates all FSRS fields on a card.
func (r *Repo) UpdateSRS(ctx context.Context, userID, cardID uuid.UUID, params domain.SRSUpdateParams) (*domain.Card, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	rowsAffected, err := q.UpdateCardSRS(ctx, sqlc.UpdateCardSRSParams{
		ID:            cardID,
		UserID:        userID,
		State:         sqlc.CardState(params.State),
		Step:          int32(params.Step),
		Stability:     params.Stability,
		Difficulty:    params.Difficulty,
		Due:           params.Due,
		LastReview:    params.LastReview,
		Reps:          int32(params.Reps),
		Lapses:        int32(params.Lapses),
		ScheduledDays: int32(params.ScheduledDays),
		ElapsedDays:   int32(params.ElapsedDays),
	})
	if err != nil {
		return nil, mapError(err, "card", cardID)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("card %s: %w", cardID, domain.ErrNotFound)
	}

	return r.GetByID(ctx, userID, cardID)
}

// Delete removes a card by ID.
func (r *Repo) Delete(ctx context.Context, userID, cardID uuid.UUID) error {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	rowsAffected, err := q.DeleteCard(ctx, sqlc.DeleteCardParams{
		ID:     cardID,
		UserID: userID,
	})
	if err != nil {
		return mapError(err, "card", cardID)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("card %s: %w", cardID, domain.ErrNotFound)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Row scanning helpers
// ---------------------------------------------------------------------------

func scanCards(rows pgx.Rows) ([]domain.Card, error) {
	var cards []domain.Card
	for rows.Next() {
		c, err := scanCardFromRows(rows)
		if err != nil {
			return nil, err
		}
		cards = append(cards, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if cards == nil {
		cards = []domain.Card{}
	}

	return cards, nil
}

func scanCardPointers(rows pgx.Rows) ([]*domain.Card, error) {
	var cards []*domain.Card
	for rows.Next() {
		c, err := scanCardFromRows(rows)
		if err != nil {
			return nil, err
		}
		cards = append(cards, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if cards == nil {
		cards = []*domain.Card{}
	}

	return cards, nil
}

func scanCardFromRows(rows pgx.Rows) (domain.Card, error) {
	var (
		id            uuid.UUID
		userID        uuid.UUID
		entryID       uuid.UUID
		state         string
		step          int32
		stability     float64
		difficulty    float64
		due           time.Time
		lastReview    *time.Time
		reps          int32
		lapses        int32
		scheduledDays int32
		elapsedDays   int32
		createdAt     time.Time
		updatedAt     time.Time
	)

	if err := rows.Scan(&id, &userID, &entryID, &state, &step, &stability, &difficulty,
		&due, &lastReview, &reps, &lapses, &scheduledDays, &elapsedDays,
		&createdAt, &updatedAt); err != nil {
		return domain.Card{}, err
	}

	return domain.Card{
		ID:            id,
		UserID:        userID,
		EntryID:       entryID,
		State:         domain.CardState(state),
		Step:          int(step),
		Stability:     stability,
		Difficulty:    difficulty,
		Due:           due,
		LastReview:    lastReview,
		Reps:          int(reps),
		Lapses:        int(lapses),
		ScheduledDays: int(scheduledDays),
		ElapsedDays:   int(elapsedDays),
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}, nil
}

// ---------------------------------------------------------------------------
// Mapping helpers: sqlc -> domain
// ---------------------------------------------------------------------------

func fromGetByIDRow(r sqlc.GetCardByIDRow) sqlc.Card {
	return sqlc.Card{
		ID: r.ID, UserID: r.UserID, EntryID: r.EntryID,
		State: r.State, Step: r.Step, Stability: r.Stability, Difficulty: r.Difficulty,
		Due: r.Due, LastReview: r.LastReview, Reps: r.Reps, Lapses: r.Lapses,
		ScheduledDays: r.ScheduledDays, ElapsedDays: r.ElapsedDays,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func fromGetByEntryIDRow(r sqlc.GetCardByEntryIDRow) sqlc.Card {
	return sqlc.Card{
		ID: r.ID, UserID: r.UserID, EntryID: r.EntryID,
		State: r.State, Step: r.Step, Stability: r.Stability, Difficulty: r.Difficulty,
		Due: r.Due, LastReview: r.LastReview, Reps: r.Reps, Lapses: r.Lapses,
		ScheduledDays: r.ScheduledDays, ElapsedDays: r.ElapsedDays,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func fromCreateRow(r sqlc.CreateCardRow) sqlc.Card {
	return sqlc.Card{
		ID: r.ID, UserID: r.UserID, EntryID: r.EntryID,
		State: r.State, Step: r.Step, Stability: r.Stability, Difficulty: r.Difficulty,
		Due: r.Due, LastReview: r.LastReview, Reps: r.Reps, Lapses: r.Lapses,
		ScheduledDays: r.ScheduledDays, ElapsedDays: r.ElapsedDays,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func toDomainCard(row sqlc.Card) domain.Card {
	return domain.Card{
		ID:            row.ID,
		UserID:        row.UserID,
		EntryID:       row.EntryID,
		State:         domain.CardState(row.State),
		Step:          int(row.Step),
		Stability:     row.Stability,
		Difficulty:    row.Difficulty,
		Due:           row.Due,
		LastReview:    row.LastReview,
		Reps:          int(row.Reps),
		Lapses:        int(row.Lapses),
		ScheduledDays: int(row.ScheduledDays),
		ElapsedDays:   int(row.ElapsedDays),
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}

// ---------------------------------------------------------------------------
// Error mapping
// ---------------------------------------------------------------------------

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
