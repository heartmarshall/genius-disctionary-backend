package study

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/service/study/fsrs"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// cardToFSRS converts a domain Card to an fsrs.Card for scheduling calculations.
func cardToFSRS(card *domain.Card) fsrs.Card {
	return fsrs.Card{
		State:         card.State,
		Step:          card.Step,
		Stability:     card.Stability,
		Difficulty:    card.Difficulty,
		Due:           card.Due,
		LastReview:    card.LastReview,
		Reps:          card.Reps,
		Lapses:        card.Lapses,
		ScheduledDays: card.ScheduledDays,
		ElapsedDays:   card.ElapsedDays,
	}
}

// fsrsResultToUpdateParams converts an FSRS scheduling result to domain update params.
func fsrsResultToUpdateParams(result fsrs.Card) domain.SRSUpdateParams {
	var lastReview *time.Time
	if result.LastReview != nil {
		t := *result.LastReview
		lastReview = &t
	}

	return domain.SRSUpdateParams{
		State:         result.State,
		Step:          result.Step,
		Stability:     result.Stability,
		Difficulty:    result.Difficulty,
		Due:           result.Due,
		LastReview:    lastReview,
		Reps:          result.Reps,
		Lapses:        result.Lapses,
		ScheduledDays: result.ScheduledDays,
		ElapsedDays:   result.ElapsedDays,
	}
}

// snapshotFromCard captures the current SRS state of a card before mutation.
func snapshotFromCard(card *domain.Card) *domain.CardSnapshot {
	return &domain.CardSnapshot{
		State:         card.State,
		Step:          card.Step,
		Stability:     card.Stability,
		Difficulty:    card.Difficulty,
		Due:           card.Due,
		LastReview:    card.LastReview,
		Reps:          card.Reps,
		Lapses:        card.Lapses,
		ScheduledDays: card.ScheduledDays,
		ElapsedDays:   card.ElapsedDays,
	}
}

// computeElapsedDays calculates whole days elapsed since the last review.
func computeElapsedDays(lastReview *time.Time, now time.Time) int {
	if lastReview == nil {
		return 0
	}
	return max(0, int(now.Sub(*lastReview).Hours()/24))
}

// userID extracts the authenticated user's ID from context.
func (s *Service) userID(ctx context.Context) (uuid.UUID, error) {
	uid, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return uuid.Nil, domain.ErrUnauthorized
	}
	return uid, nil
}

// buildFSRSParams merges global SRS config with per-user settings into FSRS parameters.
func (s *Service) buildFSRSParams(settings *domain.UserSettings) fsrs.Parameters {
	return fsrs.Parameters{
		W:                s.fsrsWeights,
		DesiredRetention: settings.DesiredRetention,
		MaxIntervalDays:  min(s.srsConfig.MaxIntervalDays, settings.MaxIntervalDays),
		EnableFuzz:       s.srsConfig.EnableFuzz,
		LearningSteps:    s.srsConfig.LearningSteps,
		RelearningSteps:  s.srsConfig.RelearningSteps,
	}
}
