package topic

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

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
