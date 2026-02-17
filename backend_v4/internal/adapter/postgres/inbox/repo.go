// Package inbox implements the Inbox repository using PostgreSQL.
// It provides CRUD operations for quick-capture inbox items.
package inbox

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
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/inbox/sqlc"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// Repo provides inbox item persistence backed by PostgreSQL.
type Repo struct {
	pool *pgxpool.Pool
}

// New creates a new inbox repository.
func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// ---------------------------------------------------------------------------
// Read operations
// ---------------------------------------------------------------------------

// GetByID returns an inbox item by primary key.
// Returns domain.ErrNotFound if the item does not exist or belongs to another user.
func (r *Repo) GetByID(ctx context.Context, userID, itemID uuid.UUID) (*domain.InboxItem, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.GetInboxItemByID(ctx, sqlc.GetInboxItemByIDParams{
		ID:     itemID,
		UserID: userID,
	})
	if err != nil {
		return nil, mapError(err, "inbox_item", itemID)
	}

	item := toDomainInboxItem(row)
	return &item, nil
}

// List returns inbox items ordered by created_at DESC with pagination.
// Returns items, total count, and error. Returns an empty slice and totalCount 0 if the inbox is empty.
func (r *Repo) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.InboxItem, int, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	// Count total items for this user.
	totalCount, err := q.CountInboxItemsByUser(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("count inbox_items: %w", err)
	}

	// Fetch the page.
	rows, err := q.ListInboxItemsByUser(ctx, sqlc.ListInboxItemsByUserParams{
		UserID: userID,
		Lim:    int32(limit),
		Off:    int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list inbox_items: %w", err)
	}

	items := make([]*domain.InboxItem, len(rows))
	for i, row := range rows {
		item := toDomainInboxItem(row)
		items[i] = &item
	}

	return items, int(totalCount), nil
}

// Count returns the number of inbox items for a user.
func (r *Repo) Count(ctx context.Context, userID uuid.UUID) (int, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	count, err := q.CountInboxItemsByUser(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("count inbox_items: %w", err)
	}

	return int(count), nil
}

// ---------------------------------------------------------------------------
// Write operations
// ---------------------------------------------------------------------------

// Create inserts a new inbox item and returns the persisted domain.InboxItem.
func (r *Repo) Create(ctx context.Context, userID uuid.UUID, item *domain.InboxItem) (*domain.InboxItem, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.CreateInboxItem(ctx, sqlc.CreateInboxItemParams{
		ID:        item.ID,
		UserID:    userID,
		Text:      item.Text,
		Context:   ptrStringToPgText(item.Context),
		CreatedAt: item.CreatedAt,
	})
	if err != nil {
		return nil, mapError(err, "inbox_item", item.ID)
	}

	result := toDomainInboxItem(row)
	return &result, nil
}

// Delete removes an inbox item. Returns domain.ErrNotFound if the item
// does not exist or belongs to another user.
func (r *Repo) Delete(ctx context.Context, userID, itemID uuid.UUID) error {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	rowsAffected, err := q.DeleteInboxItem(ctx, sqlc.DeleteInboxItemParams{
		ID:     itemID,
		UserID: userID,
	})
	if err != nil {
		return mapError(err, "inbox_item", itemID)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("inbox_item %s: %w", itemID, domain.ErrNotFound)
	}

	return nil
}

// deleteAllInboxItemsSQL is used instead of sqlc to get rows affected count.
const deleteAllInboxItemsSQL = `DELETE FROM inbox_items WHERE user_id = $1`

// DeleteAll removes all inbox items for a user. Idempotent: calling on an
// empty inbox is not an error. Returns the number of deleted items.
func (r *Repo) DeleteAll(ctx context.Context, userID uuid.UUID) (int, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	tag, err := querier.Exec(ctx, deleteAllInboxItemsSQL, userID)
	if err != nil {
		return 0, fmt.Errorf("delete all inbox_items: %w", err)
	}

	return int(tag.RowsAffected()), nil
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

// toDomainInboxItem converts a sqlc.InboxItem row into a domain.InboxItem.
func toDomainInboxItem(row sqlc.InboxItem) domain.InboxItem {
	item := domain.InboxItem{
		ID:        row.ID,
		UserID:    row.UserID,
		Text:      row.Text,
		CreatedAt: row.CreatedAt,
	}

	if row.Context.Valid {
		item.Context = &row.Context.String
	}

	return item
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
