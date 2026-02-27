// Package enrichment implements the enrichment queue repository using PostgreSQL.
package enrichment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	postgres "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/enrichment/sqlc"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// Repo provides enrichment queue persistence backed by PostgreSQL.
type Repo struct {
	pool *pgxpool.Pool
}

// New creates a new enrichment queue repository.
func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// Enqueue adds a ref entry to the enrichment queue (idempotent, bumps priority on re-queue).
func (r *Repo) Enqueue(ctx context.Context, refEntryID uuid.UUID, priority int) error {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))
	err := q.Enqueue(ctx, sqlc.EnqueueParams{
		RefEntryID: refEntryID,
		Priority:   int32(priority),
	})
	if err != nil {
		return fmt.Errorf("enrichment.Enqueue: %w", err)
	}
	return nil
}

// ClaimBatch claims up to limit pending items for processing.
func (r *Repo) ClaimBatch(ctx context.Context, limit int) ([]domain.EnrichmentQueueItem, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))
	rows, err := q.ClaimBatch(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("enrichment.ClaimBatch: %w", err)
	}
	return toDomainItems(rows), nil
}

// MarkDone marks an item as successfully enriched.
func (r *Repo) MarkDone(ctx context.Context, refEntryID uuid.UUID) error {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))
	if err := q.MarkDone(ctx, refEntryID); err != nil {
		return fmt.Errorf("enrichment.MarkDone: %w", err)
	}
	return nil
}

// MarkFailed marks an item as failed with error message.
func (r *Repo) MarkFailed(ctx context.Context, refEntryID uuid.UUID, errMsg string) error {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))
	if err := q.MarkFailed(ctx, sqlc.MarkFailedParams{
		RefEntryID:   refEntryID,
		ErrorMessage: pgtype.Text{String: errMsg, Valid: true},
	}); err != nil {
		return fmt.Errorf("enrichment.MarkFailed: %w", err)
	}
	return nil
}

// GetStats returns aggregate counts by status.
func (r *Repo) GetStats(ctx context.Context) (domain.EnrichmentQueueStats, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))
	row, err := q.GetStats(ctx)
	if err != nil {
		return domain.EnrichmentQueueStats{}, fmt.Errorf("enrichment.GetStats: %w", err)
	}
	return domain.EnrichmentQueueStats{
		Pending:    int(row.Pending),
		Processing: int(row.Processing),
		Done:       int(row.Done),
		Failed:     int(row.Failed),
		Total:      int(row.Total),
	}, nil
}

// List returns queue items filtered by status with pagination.
func (r *Repo) List(ctx context.Context, status string, limit, offset int) ([]domain.EnrichmentQueueItem, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))
	rows, err := q.List(ctx, sqlc.ListParams{
		Column1: status,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("enrichment.List: %w", err)
	}
	return toDomainItems(rows), nil
}

// RetryAllFailed resets all failed items to pending.
func (r *Repo) RetryAllFailed(ctx context.Context) (int, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))
	n, err := q.RetryAllFailed(ctx)
	if err != nil {
		return 0, fmt.Errorf("enrichment.RetryAllFailed: %w", err)
	}
	return int(n), nil
}

// ResetProcessing resets all processing items back to pending (stuck items).
func (r *Repo) ResetProcessing(ctx context.Context) (int, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))
	n, err := q.ResetProcessing(ctx)
	if err != nil {
		return 0, fmt.Errorf("enrichment.ResetProcessing: %w", err)
	}
	return int(n), nil
}

// toDomainItems converts sqlc rows to domain items.
func toDomainItems(rows []sqlc.EnrichmentQueue) []domain.EnrichmentQueueItem {
	items := make([]domain.EnrichmentQueueItem, len(rows))
	for i, row := range rows {
		items[i] = domain.EnrichmentQueueItem{
			ID:          row.ID,
			RefEntryID:  row.RefEntryID,
			Status:      domain.EnrichmentStatus(row.Status),
			Priority:    int(row.Priority),
			ErrorMessage: pgTextToPtr(row.ErrorMessage),
			RequestedAt: row.RequestedAt,
			ProcessedAt: row.ProcessedAt,
			CreatedAt:   row.CreatedAt,
		}
	}
	return items
}

func pgTextToPtr(t pgtype.Text) *string {
	if t.Valid {
		return &t.String
	}
	return nil
}
