// Package topic implements the Topic repository using PostgreSQL.
// It provides CRUD operations for user-defined topics and M2M entry linking
// via the entry_topics join table.
package topic

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
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/topic/sqlc"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// TopicWithEntryID is the batch result type for GetByEntryIDs.
// It embeds domain.Topic and adds EntryID for grouping by the caller.
type TopicWithEntryID struct {
	EntryID uuid.UUID
	domain.Topic
}

// Repo provides topic persistence backed by PostgreSQL.
type Repo struct {
	pool *pgxpool.Pool
}

// New creates a new topic repository.
func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// ---------------------------------------------------------------------------
// Raw SQL for JOIN read queries
// ---------------------------------------------------------------------------

const getByEntryIDSQL = `
SELECT
    t.id, t.user_id, t.name, t.description, t.created_at, t.updated_at
FROM entry_topics et
JOIN topics t ON et.topic_id = t.id
WHERE et.entry_id = $1
ORDER BY t.name`

const getByEntryIDsSQL = `
SELECT
    et.entry_id,
    t.id, t.user_id, t.name, t.description, t.created_at, t.updated_at
FROM entry_topics et
JOIN topics t ON et.topic_id = t.id
WHERE et.entry_id = ANY($1::uuid[])
ORDER BY et.entry_id, t.name`

// ---------------------------------------------------------------------------
// Read operations
// ---------------------------------------------------------------------------

// GetByID returns a topic by primary key with user_id filter.
// Returns domain.ErrNotFound if the topic does not exist or belongs to another user.
func (r *Repo) GetByID(ctx context.Context, userID, topicID uuid.UUID) (*domain.Topic, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.GetTopicByID(ctx, sqlc.GetTopicByIDParams{
		ID:     topicID,
		UserID: userID,
	})
	if err != nil {
		return nil, mapError(err, "topic", topicID)
	}

	t := toDomainTopic(row)
	return &t, nil
}

// List returns all topics for a user ordered by name.
// Returns an empty slice (not nil) when the user has no topics.
func (r *Repo) List(ctx context.Context, userID uuid.UUID) ([]*domain.Topic, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	rows, err := q.ListTopicsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list topics: %w", err)
	}

	topics := make([]*domain.Topic, len(rows))
	for i, row := range rows {
		t := toDomainTopic(row)
		topics[i] = &t
	}

	return topics, nil
}

// GetTopicsByEntryID returns all topics linked to an entry via the M2M table,
// ordered by name. Returns an empty slice (not nil) when no topics are linked.
func (r *Repo) GetTopicsByEntryID(ctx context.Context, entryID uuid.UUID) ([]*domain.Topic, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	rows, err := querier.Query(ctx, getByEntryIDSQL, entryID)
	if err != nil {
		return nil, fmt.Errorf("get topics by entry_id: %w", err)
	}
	defer rows.Close()

	result, err := scanTopicPointers(rows)
	if err != nil {
		return nil, fmt.Errorf("get topics by entry_id: %w", err)
	}

	return result, nil
}

// GetByEntryIDs returns topics for multiple entries (batch for DataLoader).
// Results include EntryID for grouping by the caller.
func (r *Repo) GetByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]TopicWithEntryID, error) {
	if len(entryIDs) == 0 {
		return []TopicWithEntryID{}, nil
	}

	querier := postgres.QuerierFromCtx(ctx, r.pool)

	rows, err := querier.Query(ctx, getByEntryIDsSQL, entryIDs)
	if err != nil {
		return nil, fmt.Errorf("get topics by entry_ids: %w", err)
	}
	defer rows.Close()

	result, err := scanTopicsWithEntryID(rows)
	if err != nil {
		return nil, fmt.Errorf("get topics by entry_ids: %w", err)
	}

	return result, nil
}

// GetEntryIDsByTopicID returns entry IDs linked to a topic.
// Returns an empty slice (not nil) when no entries are linked.
func (r *Repo) GetEntryIDsByTopicID(ctx context.Context, topicID uuid.UUID) ([]uuid.UUID, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	ids, err := q.GetEntryIDsByTopicID(ctx, topicID)
	if err != nil {
		return nil, fmt.Errorf("get entry_ids by topic_id: %w", err)
	}

	return ids, nil
}

// ---------------------------------------------------------------------------
// Write operations
// ---------------------------------------------------------------------------

// Create inserts a new topic and returns the persisted domain.Topic.
// Returns domain.ErrAlreadyExists if the user already has a topic with the same name.
func (r *Repo) Create(ctx context.Context, userID uuid.UUID, topic *domain.Topic) (*domain.Topic, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.CreateTopic(ctx, sqlc.CreateTopicParams{
		UserID:      userID,
		Name:        topic.Name,
		Description: ptrStringToPgText(topic.Description),
	})
	if err != nil {
		return nil, mapError(err, "topic", uuid.Nil)
	}

	result := toDomainTopic(row)
	return &result, nil
}

// Update modifies a topic's name and/or description using partial update params.
// Returns domain.ErrNotFound if the topic does not exist or belongs to another user.
func (r *Repo) Update(ctx context.Context, userID, topicID uuid.UUID, params domain.TopicUpdateParams) (*domain.Topic, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	// First get current topic to apply partial updates.
	current, err := q.GetTopicByID(ctx, sqlc.GetTopicByIDParams{
		ID:     topicID,
		UserID: userID,
	})
	if err != nil {
		return nil, mapError(err, "topic", topicID)
	}

	// Apply partial updates.
	name := current.Name
	if params.Name != nil {
		name = *params.Name
	}

	description := current.Description
	if params.Description != nil {
		if *params.Description == "" {
			// ptr("") means clear (set NULL in DB).
			description = pgtype.Text{}
		} else {
			description = pgtype.Text{String: *params.Description, Valid: true}
		}
	}

	rowsAffected, err := q.UpdateTopic(ctx, sqlc.UpdateTopicParams{
		ID:          topicID,
		UserID:      userID,
		Name:        name,
		Description: description,
	})
	if err != nil {
		return nil, mapError(err, "topic", topicID)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("topic %s: %w", topicID, domain.ErrNotFound)
	}

	// Re-fetch to get updated_at.
	updated, err := q.GetTopicByID(ctx, sqlc.GetTopicByIDParams{
		ID:     topicID,
		UserID: userID,
	})
	if err != nil {
		return nil, mapError(err, "topic", topicID)
	}

	t := toDomainTopic(updated)
	return &t, nil
}

// Delete removes a topic. CASCADE deletes entry_topics; entries are NOT affected.
// Returns domain.ErrNotFound if the topic does not exist or belongs to another user.
func (r *Repo) Delete(ctx context.Context, userID, topicID uuid.UUID) error {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	rowsAffected, err := q.DeleteTopic(ctx, sqlc.DeleteTopicParams{
		ID:     topicID,
		UserID: userID,
	})
	if err != nil {
		return mapError(err, "topic", topicID)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("topic %s: %w", topicID, domain.ErrNotFound)
	}

	return nil
}

// LinkEntry creates an M2M link between an entry and a topic.
// Idempotent: linking the same pair twice is NOT an error (ON CONFLICT DO NOTHING).
func (r *Repo) LinkEntry(ctx context.Context, entryID, topicID uuid.UUID) error {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	err := q.LinkEntry(ctx, sqlc.LinkEntryParams{
		EntryID: entryID,
		TopicID: topicID,
	})
	if err != nil {
		return mapError(err, "entry_topic", entryID)
	}

	return nil
}

// UnlinkEntry removes the M2M link between an entry and a topic.
// Not an error if the link does not exist (0 rows affected is OK).
func (r *Repo) UnlinkEntry(ctx context.Context, entryID, topicID uuid.UUID) error {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	err := q.UnlinkEntry(ctx, sqlc.UnlinkEntryParams{
		EntryID: entryID,
		TopicID: topicID,
	})
	if err != nil {
		return mapError(err, "entry_topic", entryID)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Raw SQL for new queries
// ---------------------------------------------------------------------------

const countTopicsByUserSQL = `SELECT count(*) FROM topics WHERE user_id = $1`

const batchLinkEntriesSQL = `INSERT INTO entry_topics (entry_id, topic_id) SELECT unnest($1::uuid[]), $2 ON CONFLICT DO NOTHING`

const countEntriesByTopicIDSQL = `SELECT count(*) FROM entry_topics WHERE topic_id = $1`

// Count returns the number of topics for a user.
func (r *Repo) Count(ctx context.Context, userID uuid.UUID) (int, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	var count int
	if err := querier.QueryRow(ctx, countTopicsByUserSQL, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count topics: %w", err)
	}

	return count, nil
}

// BatchLinkEntries creates M2M links between multiple entries and a topic.
// Returns the number of new links created (existing links are skipped via ON CONFLICT DO NOTHING).
func (r *Repo) BatchLinkEntries(ctx context.Context, entryIDs []uuid.UUID, topicID uuid.UUID) (int, error) {
	if len(entryIDs) == 0 {
		return 0, nil
	}

	querier := postgres.QuerierFromCtx(ctx, r.pool)

	tag, err := querier.Exec(ctx, batchLinkEntriesSQL, entryIDs, topicID)
	if err != nil {
		return 0, mapError(err, "entry_topic", topicID)
	}

	return int(tag.RowsAffected()), nil
}

// CountEntriesByTopicID returns the number of entries linked to a topic.
func (r *Repo) CountEntriesByTopicID(ctx context.Context, topicID uuid.UUID) (int, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	var count int
	if err := querier.QueryRow(ctx, countEntriesByTopicIDSQL, topicID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count entries by topic: %w", err)
	}

	return count, nil
}

// ---------------------------------------------------------------------------
// Row scanning helpers
// ---------------------------------------------------------------------------

// scanTopics scans multiple rows from a JOIN query into domain.Topic slices.
func scanTopics(rows pgx.Rows) ([]domain.Topic, error) {
	var result []domain.Topic
	for rows.Next() {
		t, err := scanTopicFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if result == nil {
		result = []domain.Topic{}
	}

	return result, nil
}

// scanTopicPointers scans multiple rows from a JOIN query into []*domain.Topic slices.
func scanTopicPointers(rows pgx.Rows) ([]*domain.Topic, error) {
	var result []*domain.Topic
	for rows.Next() {
		t, err := scanTopicFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if result == nil {
		result = []*domain.Topic{}
	}

	return result, nil
}

// scanTopicFromRows scans a single row from pgx.Rows into a domain.Topic.
func scanTopicFromRows(rows pgx.Rows) (domain.Topic, error) {
	var (
		id          uuid.UUID
		userID      uuid.UUID
		name        string
		description pgtype.Text
		createdAt   time.Time
		updatedAt   time.Time
	)

	if err := rows.Scan(&id, &userID, &name, &description, &createdAt, &updatedAt); err != nil {
		return domain.Topic{}, err
	}

	return buildDomainTopic(id, userID, name, description, createdAt, updatedAt), nil
}

// scanTopicsWithEntryID scans multiple rows from GetByEntryIDs into TopicWithEntryID slices.
func scanTopicsWithEntryID(rows pgx.Rows) ([]TopicWithEntryID, error) {
	var result []TopicWithEntryID
	for rows.Next() {
		var (
			entryID     uuid.UUID
			id          uuid.UUID
			userID      uuid.UUID
			name        string
			description pgtype.Text
			createdAt   time.Time
			updatedAt   time.Time
		)

		if err := rows.Scan(&entryID, &id, &userID, &name, &description, &createdAt, &updatedAt); err != nil {
			return nil, err
		}

		result = append(result, TopicWithEntryID{
			EntryID: entryID,
			Topic:   buildDomainTopic(id, userID, name, description, createdAt, updatedAt),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if result == nil {
		result = []TopicWithEntryID{}
	}

	return result, nil
}

// buildDomainTopic constructs a domain.Topic from scanned values.
func buildDomainTopic(id, userID uuid.UUID, name string, description pgtype.Text, createdAt, updatedAt time.Time) domain.Topic {
	t := domain.Topic{
		ID:        id,
		UserID:    userID,
		Name:      name,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	if description.Valid {
		t.Description = &description.String
	}

	return t
}

// ---------------------------------------------------------------------------
// Mapping helpers: sqlc -> domain
// ---------------------------------------------------------------------------

// toDomainTopic converts a sqlc.Topic row into a domain.Topic.
func toDomainTopic(row sqlc.Topic) domain.Topic {
	t := domain.Topic{
		ID:        row.ID,
		UserID:    row.UserID,
		Name:      row.Name,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}

	if row.Description.Valid {
		t.Description = &row.Description.String
	}

	return t
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
// pgtype helpers
// ---------------------------------------------------------------------------

// ptrStringToPgText converts a *string to pgtype.Text (nil -> NULL).
func ptrStringToPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}
