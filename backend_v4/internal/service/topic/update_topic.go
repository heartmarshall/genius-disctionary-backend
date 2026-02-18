package topic

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// UpdateTopic updates an existing topic for the authenticated user.
func (s *Service) UpdateTopic(ctx context.Context, input UpdateTopicInput) (*domain.Topic, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
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
	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Fetch old state inside transaction for accurate audit diff.
		old, getErr := s.topics.GetByID(txCtx, userID, input.TopicID)
		if getErr != nil {
			return fmt.Errorf("get topic: %w", getErr)
		}

		var updateErr error
		updated, updateErr = s.topics.Update(txCtx, userID, input.TopicID, params)
		if updateErr != nil {
			return fmt.Errorf("update topic: %w", updateErr)
		}

		// Skip audit if nothing actually changed.
		changes := buildTopicChanges(old, updated)
		if len(changes) > 0 {
			if auditErr := s.audit.Log(txCtx, domain.AuditRecord{
				UserID:     userID,
				EntityType: domain.EntityTypeTopic,
				EntityID:   &input.TopicID,
				Action:     domain.AuditActionUpdate,
				Changes:    changes,
			}); auditErr != nil {
				return fmt.Errorf("audit log: %w", auditErr)
			}
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
