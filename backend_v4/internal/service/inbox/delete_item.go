package inbox

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// DeleteItem deletes an inbox item by ID.
func (s *Service) DeleteItem(ctx context.Context, input DeleteItemInput) error {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return err
	}

	if err := s.inbox.Delete(ctx, userID, input.ItemID); err != nil {
		return fmt.Errorf("delete inbox item: %w", err)
	}

	s.log.InfoContext(ctx, "inbox item deleted",
		slog.String("user_id", userID.String()),
		slog.String("item_id", input.ItemID.String()),
	)

	return nil
}

// DeleteAll deletes all inbox items for the current user.
func (s *Service) DeleteAll(ctx context.Context) (int, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return 0, domain.ErrUnauthorized
	}

	deletedCount, err := s.inbox.DeleteAll(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("delete all inbox items: %w", err)
	}

	s.log.InfoContext(ctx, "all inbox items deleted",
		slog.String("user_id", userID.String()),
		slog.Int("deleted_count", deletedCount),
	)

	return deletedCount, nil
}
