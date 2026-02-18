package inbox

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// ListItems returns a paginated list of inbox items.
func (s *Service) ListItems(ctx context.Context, input ListItemsInput) ([]*domain.InboxItem, int, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, 0, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, 0, err
	}

	limit := input.Limit
	if limit == 0 {
		limit = DefaultLimit
	}

	items, totalCount, err := s.inbox.List(ctx, userID, limit, input.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list inbox items: %w", err)
	}

	return items, totalCount, nil
}

// GetItem returns a single inbox item by ID.
func (s *Service) GetItem(ctx context.Context, itemID uuid.UUID) (*domain.InboxItem, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	item, err := s.inbox.GetByID(ctx, userID, itemID)
	if err != nil {
		return nil, fmt.Errorf("get inbox item: %w", err)
	}

	return item, nil
}
