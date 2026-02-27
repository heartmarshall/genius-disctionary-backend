package enrichment

import (
	"context"
	"testing"

	"log/slog"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

type mockQueueRepo struct {
	enqueueFn         func(ctx context.Context, refEntryID uuid.UUID, priority int) error
	claimBatchFn      func(ctx context.Context, limit int) ([]domain.EnrichmentQueueItem, error)
	markDoneFn        func(ctx context.Context, refEntryID uuid.UUID) error
	markFailedFn      func(ctx context.Context, refEntryID uuid.UUID, errMsg string) error
	getStatsFn        func(ctx context.Context) (domain.EnrichmentQueueStats, error)
	listFn            func(ctx context.Context, status string, limit, offset int) ([]domain.EnrichmentQueueItem, error)
	retryAllFailedFn  func(ctx context.Context) (int, error)
	resetProcessingFn func(ctx context.Context) (int, error)
}

func (m *mockQueueRepo) Enqueue(ctx context.Context, refEntryID uuid.UUID, priority int) error {
	return m.enqueueFn(ctx, refEntryID, priority)
}
func (m *mockQueueRepo) ClaimBatch(ctx context.Context, limit int) ([]domain.EnrichmentQueueItem, error) {
	return m.claimBatchFn(ctx, limit)
}
func (m *mockQueueRepo) MarkDone(ctx context.Context, refEntryID uuid.UUID) error {
	return m.markDoneFn(ctx, refEntryID)
}
func (m *mockQueueRepo) MarkFailed(ctx context.Context, refEntryID uuid.UUID, errMsg string) error {
	return m.markFailedFn(ctx, refEntryID, errMsg)
}
func (m *mockQueueRepo) GetStats(ctx context.Context) (domain.EnrichmentQueueStats, error) {
	return m.getStatsFn(ctx)
}
func (m *mockQueueRepo) List(ctx context.Context, status string, limit, offset int) ([]domain.EnrichmentQueueItem, error) {
	return m.listFn(ctx, status, limit, offset)
}
func (m *mockQueueRepo) RetryAllFailed(ctx context.Context) (int, error) {
	return m.retryAllFailedFn(ctx)
}
func (m *mockQueueRepo) ResetProcessing(ctx context.Context) (int, error) {
	return m.resetProcessingFn(ctx)
}

func TestService_Enqueue(t *testing.T) {
	t.Parallel()

	refID := uuid.New()
	var calledWith uuid.UUID
	var calledPriority int

	repo := &mockQueueRepo{
		enqueueFn: func(_ context.Context, id uuid.UUID, p int) error {
			calledWith = id
			calledPriority = p
			return nil
		},
	}

	svc := NewService(slog.Default(), repo)
	err := svc.Enqueue(context.Background(), refID)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if calledWith != refID {
		t.Errorf("Enqueue called with %s, want %s", calledWith, refID)
	}
	if calledPriority != 0 {
		t.Errorf("Enqueue priority = %d, want 0", calledPriority)
	}
}

func TestService_ClaimBatch_DefaultLimit(t *testing.T) {
	t.Parallel()

	var calledLimit int
	repo := &mockQueueRepo{
		claimBatchFn: func(_ context.Context, limit int) ([]domain.EnrichmentQueueItem, error) {
			calledLimit = limit
			return nil, nil
		},
	}

	svc := NewService(slog.Default(), repo)
	_, _ = svc.ClaimBatch(context.Background(), 0)
	if calledLimit != 50 {
		t.Errorf("ClaimBatch default limit = %d, want 50", calledLimit)
	}
}

func TestService_GetStats(t *testing.T) {
	t.Parallel()

	expected := domain.EnrichmentQueueStats{Pending: 5, Done: 10, Total: 15}
	repo := &mockQueueRepo{
		getStatsFn: func(_ context.Context) (domain.EnrichmentQueueStats, error) {
			return expected, nil
		},
	}

	svc := NewService(slog.Default(), repo)
	stats, err := svc.GetStats(context.Background())
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.Pending != 5 || stats.Done != 10 || stats.Total != 15 {
		t.Errorf("stats = %+v, want %+v", stats, expected)
	}
}
