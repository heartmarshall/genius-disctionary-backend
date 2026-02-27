// Package enrichment provides business logic for the enrichment queue.
package enrichment

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

type queueRepo interface {
	Enqueue(ctx context.Context, refEntryID uuid.UUID, priority int) error
	ClaimBatch(ctx context.Context, limit int) ([]domain.EnrichmentQueueItem, error)
	MarkDone(ctx context.Context, refEntryID uuid.UUID) error
	MarkFailed(ctx context.Context, refEntryID uuid.UUID, errMsg string) error
	GetStats(ctx context.Context) (domain.EnrichmentQueueStats, error)
	List(ctx context.Context, status string, limit, offset int) ([]domain.EnrichmentQueueItem, error)
	RetryAllFailed(ctx context.Context) (int, error)
	ResetProcessing(ctx context.Context) (int, error)
}

// Service wraps the enrichment queue repository with business logic.
type Service struct {
	log   *slog.Logger
	queue queueRepo
}

// NewService creates a new enrichment service.
func NewService(log *slog.Logger, queue queueRepo) *Service {
	return &Service{
		log:   log.With("service", "enrichment"),
		queue: queue,
	}
}

// Enqueue adds a ref entry to the enrichment queue (idempotent, bumps priority if re-queued).
func (s *Service) Enqueue(ctx context.Context, refEntryID uuid.UUID) error {
	return s.queue.Enqueue(ctx, refEntryID, 0)
}

// ClaimBatch claims up to limit pending items for processing.
func (s *Service) ClaimBatch(ctx context.Context, limit int) ([]domain.EnrichmentQueueItem, error) {
	if limit <= 0 {
		limit = 50
	}
	items, err := s.queue.ClaimBatch(ctx, limit)
	if err != nil {
		return nil, err
	}
	s.log.InfoContext(ctx, "claimed batch", slog.Int("count", len(items)))
	return items, nil
}

// MarkDone marks an item as successfully enriched.
func (s *Service) MarkDone(ctx context.Context, refEntryID uuid.UUID) error {
	return s.queue.MarkDone(ctx, refEntryID)
}

// MarkFailed marks an item as failed with error message.
func (s *Service) MarkFailed(ctx context.Context, refEntryID uuid.UUID, errMsg string) error {
	return s.queue.MarkFailed(ctx, refEntryID, errMsg)
}

// GetStats returns aggregate counts by status.
func (s *Service) GetStats(ctx context.Context) (domain.EnrichmentQueueStats, error) {
	return s.queue.GetStats(ctx)
}

// List returns queue items filtered by status with pagination.
func (s *Service) List(ctx context.Context, status string, limit, offset int) ([]domain.EnrichmentQueueItem, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.queue.List(ctx, status, limit, offset)
}

// RetryAllFailed resets all failed items to pending.
func (s *Service) RetryAllFailed(ctx context.Context) (int, error) {
	n, err := s.queue.RetryAllFailed(ctx)
	if err != nil {
		return 0, err
	}
	s.log.InfoContext(ctx, "retried all failed items", slog.Int("count", n))
	return n, nil
}

// ResetProcessing resets stuck processing items back to pending.
func (s *Service) ResetProcessing(ctx context.Context) (int, error) {
	n, err := s.queue.ResetProcessing(ctx)
	if err != nil {
		return 0, err
	}
	s.log.InfoContext(ctx, "reset processing items", slog.Int("count", n))
	return n, nil
}
