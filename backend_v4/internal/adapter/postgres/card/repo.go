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
// Raw SQL for complex queries requiring JOINs
// ---------------------------------------------------------------------------

const getDueCardsSQL = `
SELECT c.id, c.user_id, c.entry_id, c.status, c.learning_step,
       c.next_review_at, c.interval_days, c.ease_factor, c.created_at, c.updated_at
FROM cards c
JOIN entries e ON c.entry_id = e.id
WHERE c.user_id = $1
  AND e.deleted_at IS NULL
  AND c.status != 'MASTERED'
  AND (c.status = 'NEW' OR c.next_review_at <= $2)
ORDER BY
  CASE WHEN c.status = 'NEW' THEN 1 ELSE 0 END,
  c.next_review_at ASC NULLS LAST
LIMIT $3`

const getNewCardsSQL = `
SELECT c.id, c.user_id, c.entry_id, c.status, c.learning_step,
       c.next_review_at, c.interval_days, c.ease_factor, c.created_at, c.updated_at
FROM cards c
JOIN entries e ON c.entry_id = e.id
WHERE c.user_id = $1 AND e.deleted_at IS NULL AND c.status = 'NEW'
ORDER BY c.created_at
LIMIT $2`

const countDueSQL = `
SELECT count(*) FROM cards c
JOIN entries e ON c.entry_id = e.id
WHERE c.user_id = $1 AND e.deleted_at IS NULL
  AND c.status != 'MASTERED'
  AND (c.status = 'NEW' OR c.next_review_at <= $2)`

const countNewSQL = `
SELECT count(*) FROM cards c
JOIN entries e ON c.entry_id = e.id
WHERE c.user_id = $1 AND e.deleted_at IS NULL AND c.status = 'NEW'`

const countByStatusSQL = `
SELECT c.status, count(*) as count
FROM cards c
JOIN entries e ON c.entry_id = e.id
WHERE c.user_id = $1 AND e.deleted_at IS NULL
GROUP BY c.status`

const getByEntryIDsSQL = `
SELECT c.id, c.user_id, c.entry_id, c.status, c.learning_step,
       c.next_review_at, c.interval_days, c.ease_factor, c.created_at, c.updated_at
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

	c := toDomainCard(row)
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

	c := toDomainCard(row)
	return &c, nil
}

// GetByEntryIDs returns cards for multiple entries (batch for DataLoader).
// Results include EntryID in domain.Card for grouping by the caller.
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
// Overdue cards come first (ordered by next_review_at ASC), then NEW cards.
// Soft-deleted entries and MASTERED cards are excluded.
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
// Soft-deleted entries are excluded.
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

// CountDue returns the count of cards due for review (excluding mastered and soft-deleted).
func (r *Repo) CountDue(ctx context.Context, userID uuid.UUID, now time.Time) (int, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	var count int
	if err := querier.QueryRow(ctx, countDueSQL, userID, now).Scan(&count); err != nil {
		return 0, fmt.Errorf("count due cards: %w", err)
	}

	return count, nil
}

// CountNew returns the count of NEW cards (excluding soft-deleted entries).
func (r *Repo) CountNew(ctx context.Context, userID uuid.UUID) (int, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	var count int
	if err := querier.QueryRow(ctx, countNewSQL, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count new cards: %w", err)
	}

	return count, nil
}

// CountByStatus returns card counts grouped by learning status.
// Soft-deleted entries are excluded.
func (r *Repo) CountByStatus(ctx context.Context, userID uuid.UUID) (domain.CardStatusCounts, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	rows, err := querier.Query(ctx, countByStatusSQL, userID)
	if err != nil {
		return domain.CardStatusCounts{}, fmt.Errorf("count cards by status: %w", err)
	}
	defer rows.Close()

	var counts domain.CardStatusCounts
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return domain.CardStatusCounts{}, fmt.Errorf("scan status count: %w", err)
		}
		switch domain.LearningStatus(status) {
		case domain.LearningStatusNew:
			counts.New = count
		case domain.LearningStatusLearning:
			counts.Learning = count
		case domain.LearningStatusReview:
			counts.Review = count
		case domain.LearningStatusMastered:
			counts.Mastered = count
		}
		counts.Total += count
	}
	if err := rows.Err(); err != nil {
		return domain.CardStatusCounts{}, fmt.Errorf("iterate status counts: %w", err)
	}

	return counts, nil
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

// Create inserts a new card and returns the persisted domain.Card.
// Duplicate entry_id results in domain.ErrAlreadyExists.
func (r *Repo) Create(ctx context.Context, userID, entryID uuid.UUID, status domain.LearningStatus, easeFactor float64) (*domain.Card, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	now := time.Now().UTC().Truncate(time.Microsecond)
	id := uuid.New()

	row, err := q.CreateCard(ctx, sqlc.CreateCardParams{
		ID:         id,
		UserID:     userID,
		EntryID:    entryID,
		Status:     sqlc.LearningStatus(status),
		EaseFactor: easeFactor,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	if err != nil {
		return nil, mapError(err, "card", id)
	}

	c := toDomainCard(row)
	return &c, nil
}

// CreateFromCard inserts a new card from a domain.Card struct and returns the persisted card.
// This is used by the study service which passes a pre-built Card.
func (r *Repo) CreateFromCard(ctx context.Context, userID uuid.UUID, card *domain.Card) (*domain.Card, error) {
	return r.Create(ctx, userID, card.EntryID, card.Status, card.EaseFactor)
}

// UpdateSRS updates all SRS fields on a card.
// Returns the updated card or domain.ErrNotFound if the card does not exist or belongs to another user.
func (r *Repo) UpdateSRS(ctx context.Context, userID, cardID uuid.UUID, params domain.SRSUpdateParams) (*domain.Card, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	rowsAffected, err := q.UpdateCardSRS(ctx, sqlc.UpdateCardSRSParams{
		ID:           cardID,
		UserID:       userID,
		Status:       sqlc.LearningStatus(params.Status),
		NextReviewAt: params.NextReviewAt,
		IntervalDays: int32(params.IntervalDays),
		EaseFactor:   params.EaseFactor,
		LearningStep: int32(params.LearningStep),
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
// Returns domain.ErrNotFound if the card does not exist or belongs to another user.
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

// scanCards scans multiple rows into a domain.Card slice.
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

// scanCardPointers scans multiple rows into a []*domain.Card slice.
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

// scanCardFromRows scans a single card row from pgx.Rows.
func scanCardFromRows(rows pgx.Rows) (domain.Card, error) {
	var (
		id           uuid.UUID
		userID       uuid.UUID
		entryID      uuid.UUID
		status       string
		learningStep int32
		nextReviewAt *time.Time
		intervalDays int32
		easeFactor   float64
		createdAt    time.Time
		updatedAt    time.Time
	)

	if err := rows.Scan(&id, &userID, &entryID, &status, &learningStep,
		&nextReviewAt, &intervalDays, &easeFactor, &createdAt, &updatedAt); err != nil {
		return domain.Card{}, err
	}

	return domain.Card{
		ID:           id,
		UserID:       userID,
		EntryID:      entryID,
		Status:       domain.LearningStatus(status),
		LearningStep: int(learningStep),
		NextReviewAt: nextReviewAt,
		IntervalDays: int(intervalDays),
		EaseFactor:   easeFactor,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}, nil
}

// ---------------------------------------------------------------------------
// Mapping helpers: sqlc -> domain
// ---------------------------------------------------------------------------

// toDomainCard converts a sqlc.Card to a domain.Card.
func toDomainCard(row sqlc.Card) domain.Card {
	return domain.Card{
		ID:           row.ID,
		UserID:       row.UserID,
		EntryID:      row.EntryID,
		Status:       domain.LearningStatus(row.Status),
		LearningStep: int(row.LearningStep),
		NextReviewAt: row.NextReviewAt,
		IntervalDays: int(row.IntervalDays),
		EaseFactor:   row.EaseFactor,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
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
