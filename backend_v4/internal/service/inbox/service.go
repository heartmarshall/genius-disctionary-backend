package inbox

import (
	"context"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
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
