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

// aggregateSessionResult computes session statistics from review logs.
func aggregateSessionResult(logs []*domain.ReviewLog, startedAt, now time.Time) domain.SessionResult {
	totalReviewed := len(logs)
	newReviewed := 0
	gradeCounts := domain.GradeCounts{}

	for _, log := range logs {
		if log.PrevState != nil && log.PrevState.State == domain.CardStateNew {
			newReviewed++
		}
		switch log.Grade {
		case domain.ReviewGradeAgain:
			gradeCounts.Again++
		case domain.ReviewGradeHard:
			gradeCounts.Hard++
		case domain.ReviewGradeGood:
			gradeCounts.Good++
		case domain.ReviewGradeEasy:
			gradeCounts.Easy++
		}
	}

	accuracyRate := 0.0
	if totalReviewed > 0 {
		accuracyRate = float64(gradeCounts.Good+gradeCounts.Easy) / float64(totalReviewed) * 100
	}

	return domain.SessionResult{
		TotalReviewed: totalReviewed,
		NewReviewed:   newReviewed,
		DueReviewed:   totalReviewed - newReviewed,
		GradeCounts:   gradeCounts,
		DurationMs:    now.Sub(startedAt).Milliseconds(),
		AccuracyRate:  accuracyRate,
	}
}

// snapshotToUpdateParams converts a CardSnapshot back to SRSUpdateParams for restoration.
func snapshotToUpdateParams(ps *domain.CardSnapshot) domain.SRSUpdateParams {
	return domain.SRSUpdateParams{
		State:         ps.State,
		Step:          ps.Step,
		Stability:     ps.Stability,
		Difficulty:    ps.Difficulty,
		Due:           ps.Due,
		LastReview:    ps.LastReview,
		Reps:          ps.Reps,
		Lapses:        ps.Lapses,
		ScheduledDays: ps.ScheduledDays,
		ElapsedDays:   ps.ElapsedDays,
	}
}

// filterBatchEntries categorizes entry IDs for batch card creation.
// Returns entries ready for creation, skip counts, and errors for not-found entries.
func filterBatchEntries(
	entryIDs []uuid.UUID,
	existMap map[uuid.UUID]bool,
	cardExistsMap map[uuid.UUID]bool,
	senseCounts map[uuid.UUID]int,
) (toCreate []uuid.UUID, skippedExisting int, skippedNoSenses int, errors []BatchCreateError) {
	// Phase 1: filter to existing entries
	var existing []uuid.UUID
	for _, id := range entryIDs {
		if exists, ok := existMap[id]; !ok || !exists {
			errors = append(errors, BatchCreateError{EntryID: id, Reason: "entry not found"})
		} else {
			existing = append(existing, id)
		}
	}

	// Phase 2: filter out entries that already have cards
	var withoutCards []uuid.UUID
	for _, id := range existing {
		if has, ok := cardExistsMap[id]; ok && has {
			skippedExisting++
		} else {
			withoutCards = append(withoutCards, id)
		}
	}

	// Phase 3: filter out entries without senses
	for _, id := range withoutCards {
		if cnt, ok := senseCounts[id]; !ok || cnt == 0 {
			skippedNoSenses++
		} else {
			toCreate = append(toCreate, id)
		}
	}

	return toCreate, skippedExisting, skippedNoSenses, errors
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
