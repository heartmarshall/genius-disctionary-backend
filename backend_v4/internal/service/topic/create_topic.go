package topic

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

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
