package topic

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
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

// CreateTopic creates a new topic for the authenticated user.
func (s *Service) CreateTopic(ctx context.Context, input CreateTopicInput) (*domain.Topic, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(input.Name)
	description := trimOrNil(input.Description)

	count, err := s.topics.Count(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("count topics: %w", err)
	}
	if count >= MaxTopicsPerUser {
		return nil, domain.NewValidationError("topics", "limit reached (max 100)")
	}

	var topic *domain.Topic
	err = s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		var createErr error
		topic, createErr = s.topics.Create(txCtx, userID, &domain.Topic{
			Name:        name,
			Description: description,
		})
		if createErr != nil {
			return fmt.Errorf("create topic: %w", createErr)
		}

		auditErr := s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeTopic,
			EntityID:   &topic.ID,
			Action:     domain.AuditActionCreate,
			Changes: map[string]any{
				"name": map[string]any{"new": name},
			},
		})
		if auditErr != nil {
			return fmt.Errorf("audit log: %w", auditErr)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "topic created",
		slog.String("user_id", userID.String()),
		slog.String("topic_id", topic.ID.String()),
		slog.String("name", name),
	)

	return topic, nil
}

// UpdateTopic updates an existing topic for the authenticated user.
func (s *Service) UpdateTopic(ctx context.Context, input UpdateTopicInput) (*domain.Topic, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	old, err := s.topics.GetByID(ctx, userID, input.TopicID)
	if err != nil {
		return nil, fmt.Errorf("get topic: %w", err)
	}

	params := domain.TopicUpdateParams{}
	if input.Name != nil {
		trimmed := strings.TrimSpace(*input.Name)
		params.Name = &trimmed
	}
	if input.Description != nil {
		if strings.TrimSpace(*input.Description) == "" {
			params.Description = ptr("") // clear description -> NULL in DB
		} else {
			trimmed := strings.TrimSpace(*input.Description)
			params.Description = &trimmed
		}
	}

	var updated *domain.Topic
	err = s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		var updateErr error
		updated, updateErr = s.topics.Update(txCtx, userID, input.TopicID, params)
		if updateErr != nil {
			return fmt.Errorf("update topic: %w", updateErr)
		}

		auditErr := s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeTopic,
			EntityID:   &input.TopicID,
			Action:     domain.AuditActionUpdate,
			Changes:    buildTopicChanges(old, updated),
		})
		if auditErr != nil {
			return fmt.Errorf("audit log: %w", auditErr)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "topic updated",
		slog.String("user_id", userID.String()),
		slog.String("topic_id", input.TopicID.String()),
	)

	return updated, nil
}

// buildTopicChanges returns only changed fields for audit.
func buildTopicChanges(old, updated *domain.Topic) map[string]any {
	changes := make(map[string]any)
	if old.Name != updated.Name {
		changes["name"] = map[string]any{"old": old.Name, "new": updated.Name}
	}
	oldDesc := ""
	if old.Description != nil {
		oldDesc = *old.Description
	}
	newDesc := ""
	if updated.Description != nil {
		newDesc = *updated.Description
	}
	if oldDesc != newDesc || (old.Description == nil) != (updated.Description == nil) {
		changes["description"] = map[string]any{"old": old.Description, "new": updated.Description}
	}
	return changes
}

// DeleteTopic deletes a topic for the authenticated user.
func (s *Service) DeleteTopic(ctx context.Context, input DeleteTopicInput) error {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return err
	}

	topic, err := s.topics.GetByID(ctx, userID, input.TopicID)
	if err != nil {
		return fmt.Errorf("get topic: %w", err)
	}

	err = s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		if deleteErr := s.topics.Delete(txCtx, userID, input.TopicID); deleteErr != nil {
			return fmt.Errorf("delete topic: %w", deleteErr)
		}

		auditErr := s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeTopic,
			EntityID:   &input.TopicID,
			Action:     domain.AuditActionDelete,
			Changes: map[string]any{
				"name": map[string]any{"old": topic.Name},
			},
		})
		if auditErr != nil {
			return fmt.Errorf("audit log: %w", auditErr)
		}

		return nil
	})
	if err != nil {
		return err
	}

	s.log.InfoContext(ctx, "topic deleted",
		slog.String("user_id", userID.String()),
		slog.String("topic_id", input.TopicID.String()),
		slog.String("name", topic.Name),
	)

	return nil
}

// ListTopics returns all topics for the authenticated user.
func (s *Service) ListTopics(ctx context.Context) ([]*domain.Topic, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	topics, err := s.topics.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list topics: %w", err)
	}

	return topics, nil
}

// LinkEntry links a dictionary entry to a topic. Idempotent — re-linking is not an error.
func (s *Service) LinkEntry(ctx context.Context, input LinkEntryInput) error {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return err
	}

	// Check topic ownership
	if _, err := s.topics.GetByID(ctx, userID, input.TopicID); err != nil {
		return fmt.Errorf("get topic: %w", err)
	}

	// Check entry ownership (also filters soft-deleted)
	if _, err := s.entries.GetByID(ctx, userID, input.EntryID); err != nil {
		return fmt.Errorf("get entry: %w", err)
	}

	// ON CONFLICT DO NOTHING — idempotent
	if err := s.topics.LinkEntry(ctx, input.EntryID, input.TopicID); err != nil {
		return fmt.Errorf("link entry: %w", err)
	}

	return nil
}

// UnlinkEntry removes a link between an entry and a topic. Idempotent — unlinking a non-existent link is not an error.
func (s *Service) UnlinkEntry(ctx context.Context, input UnlinkEntryInput) error {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return err
	}

	// Check topic ownership only — no entry check needed
	if _, err := s.topics.GetByID(ctx, userID, input.TopicID); err != nil {
		return fmt.Errorf("get topic: %w", err)
	}

	// Idempotent — 0 affected rows is not an error
	if err := s.topics.UnlinkEntry(ctx, input.EntryID, input.TopicID); err != nil {
		return fmt.Errorf("unlink entry: %w", err)
	}

	return nil
}

// BatchLinkEntries links multiple entries to a topic. Entries that do not exist are skipped.
func (s *Service) BatchLinkEntries(ctx context.Context, input BatchLinkEntriesInput) (*BatchLinkResult, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Check topic ownership
	if _, err := s.topics.GetByID(ctx, userID, input.TopicID); err != nil {
		return nil, fmt.Errorf("get topic: %w", err)
	}

	// Filter to existing entries only
	existing, err := s.entries.ExistByIDs(ctx, userID, input.EntryIDs)
	if err != nil {
		return nil, fmt.Errorf("check entries: %w", err)
	}

	var validEntryIDs []uuid.UUID
	for _, id := range input.EntryIDs {
		if existing[id] {
			validEntryIDs = append(validEntryIDs, id)
		}
	}

	if len(validEntryIDs) == 0 {
		return &BatchLinkResult{Linked: 0, Skipped: len(input.EntryIDs)}, nil
	}

	linked, err := s.topics.BatchLinkEntries(ctx, validEntryIDs, input.TopicID)
	if err != nil {
		return nil, fmt.Errorf("batch link entries: %w", err)
	}

	skipped := len(input.EntryIDs) - linked

	s.log.InfoContext(ctx, "entries batch linked",
		slog.String("user_id", userID.String()),
		slog.String("topic_id", input.TopicID.String()),
		slog.Int("requested", len(input.EntryIDs)),
		slog.Int("linked", linked),
		slog.Int("skipped", skipped),
	)

	return &BatchLinkResult{Linked: linked, Skipped: skipped}, nil
}
