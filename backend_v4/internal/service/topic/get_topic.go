package topic

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// GetTopic returns a single topic by ID for the authenticated user.
func (s *Service) GetTopic(ctx context.Context, topicID uuid.UUID) (*domain.Topic, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if topicID == uuid.Nil {
		return nil, domain.NewValidationError("topic_id", "required")
	}

	topic, err := s.topics.GetByID(ctx, userID, topicID)
	if err != nil {
		return nil, fmt.Errorf("get topic: %w", err)
	}

	return topic, nil
}
