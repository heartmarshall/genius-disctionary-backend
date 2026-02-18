package topic

import (
	"context"
	"fmt"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

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
