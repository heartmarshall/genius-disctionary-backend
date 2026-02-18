package study

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// GetStudyQueue returns cards ready for review (due cards + new cards respecting daily limit).
func (s *Service) GetStudyQueue(ctx context.Context, input GetQueueInput) ([]*domain.Card, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	limit := input.Limit
	if limit == 0 {
		limit = 50
	}

	now := time.Now()

	// Load user settings for limits and timezone
	settings, err := s.settings.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load settings: %w", err)
	}

	tz := ParseTimezone(settings.Timezone)
	dayStart := DayStart(now, tz)

	// Count new cards reviewed today
	newToday, err := s.reviews.CountNewToday(ctx, userID, dayStart)
	if err != nil {
		return nil, fmt.Errorf("count new today: %w", err)
	}

	newRemaining := max(0, settings.NewCardsPerDay-newToday)

	// Get due cards (overdue not limited by reviews_per_day)
	dueCards, err := s.cards.GetDueCards(ctx, userID, now, limit)
	if err != nil {
		return nil, fmt.Errorf("get due cards: %w", err)
	}

	// Fill remaining slots with new cards
	queue := dueCards
	if len(dueCards) < limit && newRemaining > 0 {
		newLimit := min(limit-len(dueCards), newRemaining)
		newCards, err := s.cards.GetNewCards(ctx, userID, newLimit)
		if err != nil {
			return nil, fmt.Errorf("get new cards: %w", err)
		}
		queue = append(queue, newCards...)
	}

	s.log.InfoContext(ctx, "study queue generated",
		slog.String("user_id", userID.String()),
		slog.Int("due_count", len(dueCards)),
		slog.Int("new_count", len(queue)-len(dueCards)),
		slog.Int("total", len(queue)),
	)

	return queue, nil
}
