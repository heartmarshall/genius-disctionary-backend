package study

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"golang.org/x/sync/errgroup"
)

// GetDashboard returns aggregated study statistics for the user.
func (s *Service) GetDashboard(ctx context.Context) (domain.Dashboard, error) {
	userID, err := s.userID(ctx)
	if err != nil {
		return domain.Dashboard{}, err
	}

	now := s.clock.Now()

	// Load settings for timezone
	settings, err := s.settings.GetByUserID(ctx, userID)
	if err != nil {
		return domain.Dashboard{}, fmt.Errorf("load settings: %w", err)
	}

	tz := ParseTimezone(settings.Timezone)
	dayStart := DayStart(now, tz)

	var (
		dueCount      int
		newCount      int
		reviewedToday int
		newToday      int
		overdueCount  int
		statusCounts  domain.CardStatusCounts
		streakDays    []domain.DayReviewCount
		activeSession *domain.StudySession
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var gErr error
		dueCount, gErr = s.cards.CountDue(gctx, userID, now)
		return gErr
	})
	g.Go(func() error {
		var gErr error
		newCount, gErr = s.cards.CountNew(gctx, userID)
		return gErr
	})
	g.Go(func() error {
		var gErr error
		reviewedToday, gErr = s.reviews.CountToday(gctx, userID, dayStart)
		return gErr
	})
	g.Go(func() error {
		var gErr error
		newToday, gErr = s.reviews.CountNewToday(gctx, userID, dayStart)
		return gErr
	})
	g.Go(func() error {
		var gErr error
		overdueCount, gErr = s.cards.CountOverdue(gctx, userID, dayStart)
		return gErr
	})
	g.Go(func() error {
		var gErr error
		statusCounts, gErr = s.cards.CountByStatus(gctx, userID)
		return gErr
	})
	g.Go(func() error {
		var gErr error
		streakDays, gErr = s.reviews.GetStreakDays(gctx, userID, dayStart, 365, settings.Timezone)
		return gErr
	})
	g.Go(func() error {
		var gErr error
		activeSession, gErr = s.sessions.GetActive(gctx, userID)
		if gErr != nil && !errors.Is(gErr, domain.ErrNotFound) {
			return gErr
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return domain.Dashboard{}, fmt.Errorf("dashboard queries: %w", err)
	}

	// Calculate streak
	nowInTz := now.In(tz)
	today := time.Date(nowInTz.Year(), nowInTz.Month(), nowInTz.Day(), 0, 0, 0, 0, tz)
	streak := calculateStreak(streakDays, today)

	dashboard := domain.Dashboard{
		DueCount:      dueCount,
		NewCount:      newCount,
		ReviewedToday: reviewedToday,
		NewToday:      newToday,
		Streak:        streak,
		StatusCounts:  statusCounts,
		OverdueCount:  overdueCount,
		ActiveSession: activeSession,
	}

	s.log.InfoContext(ctx, "dashboard loaded",
		slog.String("user_id", userID.String()),
		slog.Int("due_count", dueCount),
		slog.Int("new_count", newCount),
		slog.Int("streak", streak),
	)

	return dashboard, nil
}

// GetCardHistory returns the review history of a card with pagination.
func (s *Service) GetCardHistory(ctx context.Context, input GetCardHistoryInput) ([]*domain.ReviewLog, int, error) {
	userID, err := s.userID(ctx)
	if err != nil {
		return nil, 0, err
	}

	if err := input.Validate(); err != nil {
		return nil, 0, err
	}

	// Check ownership
	_, err = s.cards.GetByID(ctx, userID, input.CardID)
	if err != nil {
		return nil, 0, fmt.Errorf("get card: %w", err)
	}

	limit := input.Limit
	if limit == 0 {
		limit = 50
	}

	// Get history
	logs, total, err := s.reviews.GetByCardID(ctx, input.CardID, limit, input.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("get review logs: %w", err)
	}

	s.log.InfoContext(ctx, "card history retrieved",
		slog.String("user_id", userID.String()),
		slog.String("card_id", input.CardID.String()),
		slog.Int("count", len(logs)),
		slog.Int("total", total),
	)

	return logs, total, nil
}

// GetCardStats returns aggregated statistics for a card.
func (s *Service) GetCardStats(ctx context.Context, input GetCardHistoryInput) (domain.CardStats, error) {
	userID, err := s.userID(ctx)
	if err != nil {
		return domain.CardStats{}, err
	}

	if err := input.Validate(); err != nil {
		return domain.CardStats{}, err
	}

	card, err := s.cards.GetByID(ctx, userID, input.CardID)
	if err != nil {
		return domain.CardStats{}, fmt.Errorf("get card: %w", err)
	}

	agg, err := s.reviews.GetStatsByCardID(ctx, input.CardID)
	if err != nil {
		return domain.CardStats{}, fmt.Errorf("get review stats: %w", err)
	}

	stats := domain.CardStats{
		TotalReviews:  agg.TotalReviews,
		CurrentState:  card.State,
		Stability:     card.Stability,
		Difficulty:    card.Difficulty,
		ScheduledDays: card.ScheduledDays,
		AverageTimeMs: agg.AvgDurationMs,
	}

	if agg.TotalReviews > 0 {
		stats.AccuracyRate = float64(agg.GoodCount+agg.EasyCount) / float64(agg.TotalReviews) * 100
		stats.GradeDistribution = &domain.GradeCounts{
			Again: agg.AgainCount,
			Hard:  agg.HardCount,
			Good:  agg.GoodCount,
			Easy:  agg.EasyCount,
		}
	}

	s.log.InfoContext(ctx, "card stats calculated",
		slog.String("user_id", userID.String()),
		slog.String("card_id", input.CardID.String()),
		slog.Int("total_reviews", agg.TotalReviews),
		slog.Float64("accuracy_rate", stats.AccuracyRate),
	)

	return stats, nil
}

// ---------------------------------------------------------------------------
// Helper Functions
// ---------------------------------------------------------------------------

// calculateStreak calculates the current review streak in days.
// days must be sorted DESC by date (most recent first).
// Returns the number of consecutive days with reviews, starting from today or yesterday.
func calculateStreak(days []domain.DayReviewCount, today time.Time) int {
	if len(days) == 0 {
		return 0
	}

	streak := 0
	expectedDate := today

	// Helper to compare only date parts (ignore time)
	sameDay := func(a, b time.Time) bool {
		return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
	}

	// If today has no reviews, start from yesterday
	if len(days) > 0 && !sameDay(days[0].Date, today) {
		expectedDate = today.AddDate(0, 0, -1)
	}

	for _, d := range days {
		if sameDay(d.Date, expectedDate) {
			streak++
			expectedDate = expectedDate.AddDate(0, 0, -1)
		} else {
			break // Gap in streak or unexpected date order
		}
	}
	return streak
}
