package topic

import (
	"context"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

type topicRepo interface {
	Create(ctx context.Context, userID uuid.UUID, topic *domain.Topic) (*domain.Topic, error)
	GetByID(ctx context.Context, userID, topicID uuid.UUID) (*domain.Topic, error)
	Update(ctx context.Context, userID, topicID uuid.UUID, params domain.TopicUpdateParams) (*domain.Topic, error)
	Delete(ctx context.Context, userID, topicID uuid.UUID) error
	List(ctx context.Context, userID uuid.UUID) ([]*domain.Topic, error)
	Count(ctx context.Context, userID uuid.UUID) (int, error)

	// M2M: entry <-> topic
	LinkEntry(ctx context.Context, entryID, topicID uuid.UUID) error
	UnlinkEntry(ctx context.Context, entryID, topicID uuid.UUID) error
	BatchLinkEntries(ctx context.Context, entryIDs []uuid.UUID, topicID uuid.UUID) (int, error)

	// M2M read
	GetTopicsByEntryID(ctx context.Context, entryID uuid.UUID) ([]*domain.Topic, error)
	GetEntryIDsByTopicID(ctx context.Context, topicID uuid.UUID) ([]uuid.UUID, error)
	CountEntriesByTopicID(ctx context.Context, topicID uuid.UUID) (int, error)
}

type entryRepo interface {
	GetByID(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error)
	ExistByIDs(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]bool, error)
}

type auditLogger interface {
	Log(ctx context.Context, record domain.AuditRecord) error
}

type txManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

const (
	MaxTopicsPerUser = 100
)

// Service provides topic management operations.
type Service struct {
	topics  topicRepo
	entries entryRepo
	audit   auditLogger
	tx      txManager
	log     *slog.Logger
}

// NewService creates a new Topic service.
func NewService(
	log *slog.Logger,
	topics topicRepo,
	entries entryRepo,
	audit auditLogger,
	tx txManager,
) *Service {
	return &Service{
		topics:  topics,
		entries: entries,
		audit:   audit,
		tx:      tx,
		log:     log.With("service", "topic"),
	}
}

// BatchLinkResult holds the outcome of a batch link operation.
type BatchLinkResult struct {
	Linked  int
	Skipped int
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

// ptr returns a pointer to the given string.
func ptr(s string) *string {
	return &s
}
