package inbox

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

const (
	MaxInboxItems = 500
	DefaultLimit  = 50
)

type inboxRepo interface {
	Create(ctx context.Context, userID uuid.UUID, item *domain.InboxItem) (*domain.InboxItem, error)
	GetByID(ctx context.Context, userID, itemID uuid.UUID) (*domain.InboxItem, error)
	List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.InboxItem, int, error)
	Delete(ctx context.Context, userID, itemID uuid.UUID) error
	DeleteAll(ctx context.Context, userID uuid.UUID) (int, error)
	Count(ctx context.Context, userID uuid.UUID) (int, error)
}

// Service provides inbox management operations.
type Service struct {
	inbox inboxRepo
	log   *slog.Logger
}

// NewService creates a new Inbox service.
func NewService(
	log *slog.Logger,
	inbox inboxRepo,
) *Service {
	return &Service{
		inbox: inbox,
		log:   log.With("service", "inbox"),
	}
}

// trimOrNil trims whitespace. Returns nil if result is empty.
func trimOrNil(s *string) *string {
	if s == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// CreateItem creates a new inbox item.
func (s *Service) CreateItem(ctx context.Context, input CreateItemInput) (*domain.InboxItem, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	text := strings.TrimSpace(input.Text)
	ctxField := trimOrNil(input.Context)

	count, err := s.inbox.Count(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("count inbox items: %w", err)
	}
	if count >= MaxInboxItems {
		return nil, domain.NewValidationError("inbox", "inbox is full (max 500 items)")
	}

	item, err := s.inbox.Create(ctx, userID, &domain.InboxItem{
		Text:    text,
		Context: ctxField,
	})
	if err != nil {
		return nil, fmt.Errorf("create inbox item: %w", err)
	}

	textPreview := text
	if len(textPreview) > 50 {
		textPreview = textPreview[:50]
	}

	s.log.InfoContext(ctx, "inbox item created",
		slog.String("user_id", userID.String()),
		slog.String("item_id", item.ID.String()),
		slog.String("text", textPreview),
	)

	return item, nil
}

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
