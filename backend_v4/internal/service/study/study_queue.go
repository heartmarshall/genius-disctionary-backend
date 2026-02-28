package study

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// GetStudyQueue returns cards ready for review (due cards + new cards respecting daily limit).
func (s *Service) GetStudyQueue(ctx context.Context, input GetQueueInput) ([]*domain.Card, error) {
	userID, err := s.userID(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	limit := input.Limit
	if limit == 0 {
		limit = 50
	}

	now := s.clock.Now()

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

	// Due cards are always returned regardless of ReviewsPerDay setting.
	// Design decision: hiding due cards degrades long-term retention (Anki behaviour).
	// ReviewsPerDay is an informational goal shown in dashboard UI, not a hard limit.
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
