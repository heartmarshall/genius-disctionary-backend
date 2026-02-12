package card

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/model"
	"github.com/heartmarshall/my-english/internal/service/types"
)

// createAuditLog создает запись аудита для операции над карточкой.
func (s *Service) createAuditLog(ctx context.Context, cardID uuid.UUID, action model.AuditAction, changes model.JSON) error {
	audit := &model.AuditRecord{
		EntityType: model.EntityCard,
		EntityID:   &cardID,
		Action:     action,
		Changes:    changes,
	}
	_, err := s.repos.Audit.Create(ctx, audit)
	if err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}
	return nil
}

// diffCard сравнивает две карточки и возвращает изменения полей.
func diffCard(old, new *model.Card) model.JSON {
	changes := make(model.JSON)

	if old.Status != new.Status {
		changes[types.AuditFieldStatus] = map[string]any{
			types.AuditFieldOld: old.Status,
			types.AuditFieldNew: new.Status,
		}
	}

	if !equalTimePtr(old.NextReviewAt, new.NextReviewAt) {
		changes[types.AuditFieldNextReviewAt] = map[string]any{
			types.AuditFieldOld: formatTimePtr(old.NextReviewAt),
			types.AuditFieldNew: formatTimePtr(new.NextReviewAt),
		}
	}

	if old.IntervalDays != new.IntervalDays {
		changes[types.AuditFieldIntervalDays] = map[string]any{
			types.AuditFieldOld: old.IntervalDays,
			types.AuditFieldNew: new.IntervalDays,
		}
	}

	if old.EaseFactor != new.EaseFactor {
		changes[types.AuditFieldEaseFactor] = map[string]any{
			types.AuditFieldOld: old.EaseFactor,
			types.AuditFieldNew: new.EaseFactor,
		}
	}

	return changes
}

// buildCreateChanges создает структуру изменений для операции CREATE карточки.
func buildCreateChanges(card *model.Card) model.JSON {
	changes := make(model.JSON)
	changes[types.AuditFieldEntryID] = card.EntryID.String()
	changes[types.AuditFieldStatus] = card.Status
	if card.NextReviewAt != nil {
		changes[types.AuditFieldNextReviewAt] = card.NextReviewAt.Format(time.RFC3339)
	}
	changes[types.AuditFieldIntervalDays] = card.IntervalDays
	changes[types.AuditFieldEaseFactor] = card.EaseFactor
	return changes
}

// Helper functions

func equalTimePtr(a, b *time.Time) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Equal(*b)
}

func formatTimePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}
