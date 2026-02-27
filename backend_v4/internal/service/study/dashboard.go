package study

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// GetDashboard returns aggregated study statistics for the user.
func (s *Service) GetDashboard(ctx context.Context) (domain.Dashboard, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.Dashboard{}, domain.ErrUnauthorized
	}

	now := time.Now()

	// Load settings for timezone
	settings, err := s.settings.GetByUserID(ctx, userID)
	if err != nil {
		return domain.Dashboard{}, fmt.Errorf("load settings: %w", err)
	}

	tz := ParseTimezone(settings.Timezone)
	dayStart := DayStart(now, tz)

	// Make 7 repo calls to gather all data
	dueCount, err := s.cards.CountDue(ctx, userID, now)
	if err != nil {
		return domain.Dashboard{}, fmt.Errorf("count due cards: %w", err)
	}

	newCount, err := s.cards.CountNew(ctx, userID)
	if err != nil {
		return domain.Dashboard{}, fmt.Errorf("count new cards: %w", err)
	}

	reviewedToday, err := s.reviews.CountToday(ctx, userID, dayStart)
	if err != nil {
		return domain.Dashboard{}, fmt.Errorf("count reviewed today: %w", err)
	}

	newToday, err := s.reviews.CountNewToday(ctx, userID, dayStart)
	if err != nil {
		return domain.Dashboard{}, fmt.Errorf("count new today: %w", err)
	}

	statusCounts, err := s.cards.CountByStatus(ctx, userID)
	if err != nil {
		return domain.Dashboard{}, fmt.Errorf("count by status: %w", err)
	}

	streakDays, err := s.reviews.GetStreakDays(ctx, userID, dayStart, 365)
	if err != nil {
		return domain.Dashboard{}, fmt.Errorf("get streak days: %w", err)
	}

	// Active session (may be nil)
	var activeSessionID *uuid.UUID
	activeSession, err := s.sessions.GetActive(ctx, userID)
	if err == nil && activeSession != nil {
		activeSessionID = &activeSession.ID
	} else if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return domain.Dashboard{}, fmt.Errorf("get active session: %w", err)
	}

	// Calculate streak using helper function
	// Convert now to user's timezone and get date at midnight
	nowInTz := now.In(tz)
	today := time.Date(nowInTz.Year(), nowInTz.Month(), nowInTz.Day(), 0, 0, 0, 0, tz)
	streak := calculateStreak(streakDays, today)

	// Cards that were due before today's start (overdue by at least one full day)
	overdueCount, err := s.cards.CountOverdue(ctx, userID, dayStart)
	if err != nil {
		return domain.Dashboard{}, fmt.Errorf("count overdue cards: %w", err)
	}

	dashboard := domain.Dashboard{
		DueCount:      dueCount,
		NewCount:      newCount,
		ReviewedToday: reviewedToday,
		NewToday:      newToday,
		Streak:        streak,
		StatusCounts:  statusCounts,
		OverdueCount:  overdueCount,
		ActiveSession: activeSessionID,
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
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, 0, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, 0, err
	}

	// Check ownership
	_, err := s.cards.GetByID(ctx, userID, input.CardID)
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
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.CardStats{}, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return domain.CardStats{}, err
	}

	// Load card
	card, err := s.cards.GetByID(ctx, userID, input.CardID)
	if err != nil {
		return domain.CardStats{}, fmt.Errorf("get card: %w", err)
	}

	// Load ALL review logs (limit=0 means no limit)
	logs, total, err := s.reviews.GetByCardID(ctx, input.CardID, 0, 0)
	if err != nil {
		return domain.CardStats{}, fmt.Errorf("get review logs: %w", err)
	}

	// Calculate stats
	stats := domain.CardStats{
		TotalReviews:  total,
		AccuracyRate:  0.0,
		AverageTimeMs: nil,
		CurrentState:  card.State,
		Stability:     card.Stability,
		Difficulty:    card.Difficulty,
		ScheduledDays: card.ScheduledDays,
	}

	if total == 0 {
		return stats, nil
	}

	// Calculate accuracy rate and grade distribution
	dist := &domain.GradeCounts{}
	for _, log := range logs {
		switch log.Grade {
		case domain.ReviewGradeAgain:
			dist.Again++
		case domain.ReviewGradeHard:
			dist.Hard++
		case domain.ReviewGradeGood:
			dist.Good++
		case domain.ReviewGradeEasy:
			dist.Easy++
		}
	}
	stats.GradeDistribution = dist
	stats.AccuracyRate = float64(dist.Good+dist.Easy) / float64(total) * 100

	// Calculate average time
	totalDuration := 0
	durationCount := 0
	for _, log := range logs {
		if log.DurationMs != nil {
			totalDuration += *log.DurationMs
			durationCount++
		}
	}
	if durationCount > 0 {
		avgTime := totalDuration / durationCount
		stats.AverageTimeMs = &avgTime
	}

	s.log.InfoContext(ctx, "card stats calculated",
		slog.String("user_id", userID.String()),
		slog.String("card_id", input.CardID.String()),
		slog.Int("total_reviews", total),
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
