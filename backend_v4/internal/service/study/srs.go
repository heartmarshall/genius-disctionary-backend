package study

import (
	"time"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// SRSInput holds all data needed for SRS calculation. Pure value — no side effects.
type SRSInput struct {
	CurrentStatus   domain.LearningStatus
	CurrentInterval int
	CurrentEase     float64
	LearningStep    int
	Grade           domain.ReviewGrade
	Now             time.Time
	Config          domain.SRSConfig
	MaxIntervalDays int // min(config.MaxIntervalDays, user_settings.MaxIntervalDays)
}

// SRSOutput is the result of SRS calculation.
type SRSOutput struct {
	NewStatus       domain.LearningStatus
	NewInterval     int
	NewEase         float64
	NewLearningStep int
	NextReviewAt    time.Time
}

// CalculateSRS is a pure function. No DB, no context, no logger.
// All decisions are deterministic based on input parameters.
func CalculateSRS(input SRSInput) SRSOutput {
	switch input.CurrentStatus {
	case domain.LearningStatusNew:
		return calculateNew(input)
	case domain.LearningStatusLearning:
		return calculateLearning(input)
	case domain.LearningStatusReview, domain.LearningStatusMastered:
		return calculateReview(input)
	default:
		return calculateNew(input)
	}
}

func calculateNew(input SRSInput) SRSOutput {
	steps := input.Config.LearningSteps

	switch input.Grade {
	case domain.ReviewGradeAgain:
		// AGAIN → LEARNING, step 0
		delay := steps[0]
		if len(steps) == 0 {
			delay = 1 * time.Minute
		}
		return SRSOutput{
			NewStatus:       domain.LearningStatusLearning,
			NewInterval:     0,
			NewEase:         input.Config.DefaultEaseFactor,
			NewLearningStep: 0,
			NextReviewAt:    input.Now.Add(delay),
		}

	case domain.ReviewGradeHard:
		// HARD → LEARNING, step 0, delay = avg(steps[0], steps[1])
		var delay time.Duration
		if len(steps) > 1 {
			delay = (steps[0] + steps[1]) / 2
		} else if len(steps) == 1 {
			delay = steps[0]
		} else {
			delay = 1 * time.Minute
		}
		return SRSOutput{
			NewStatus:       domain.LearningStatusLearning,
			NewInterval:     0,
			NewEase:         input.Config.DefaultEaseFactor,
			NewLearningStep: 0,
			NextReviewAt:    input.Now.Add(delay),
		}

	case domain.ReviewGradeGood:
		// GOOD → step 1 or graduate
		if len(steps) > 1 {
			return SRSOutput{
				NewStatus:       domain.LearningStatusLearning,
				NewInterval:     0,
				NewEase:         input.Config.DefaultEaseFactor,
				NewLearningStep: 1,
				NextReviewAt:    input.Now.Add(steps[1]),
			}
		}
		// Graduate immediately
		return graduate(input, input.Config.GraduatingInterval, input.Config.DefaultEaseFactor)

	case domain.ReviewGradeEasy:
		// EASY → graduate with easy_interval
		return graduate(input, input.Config.EasyInterval, input.Config.DefaultEaseFactor)

	default:
		return calculateNew(input) // fallback
	}
}

func calculateLearning(input SRSInput) SRSOutput {
	// Determine if this is relearning or initial learning
	var steps []time.Duration
	isRelearning := input.CurrentInterval > 0

	if isRelearning {
		steps = input.Config.RelearningSteps
	} else {
		steps = input.Config.LearningSteps
	}

	if len(steps) == 0 {
		steps = []time.Duration{1 * time.Minute}
	}

	switch input.Grade {
	case domain.ReviewGradeAgain:
		// Reset to step 0
		return SRSOutput{
			NewStatus:       domain.LearningStatusLearning,
			NewInterval:     input.CurrentInterval,
			NewEase:         input.CurrentEase,
			NewLearningStep: 0,
			NextReviewAt:    input.Now.Add(steps[0]),
		}

	case domain.ReviewGradeHard:
		// Repeat current step
		step := input.LearningStep
		if step >= len(steps) {
			step = len(steps) - 1
		}
		return SRSOutput{
			NewStatus:       domain.LearningStatusLearning,
			NewInterval:     input.CurrentInterval,
			NewEase:         input.CurrentEase,
			NewLearningStep: input.LearningStep,
			NextReviewAt:    input.Now.Add(steps[step]),
		}

	case domain.ReviewGradeGood:
		// Next step or graduate
		nextStep := input.LearningStep + 1
		if nextStep >= len(steps) {
			// Graduate
			var ease float64
			if isRelearning {
				ease = input.CurrentEase // Preserve ease during relearning
			} else {
				ease = input.Config.DefaultEaseFactor
			}
			return graduate(input, input.Config.GraduatingInterval, ease)
		}
		return SRSOutput{
			NewStatus:       domain.LearningStatusLearning,
			NewInterval:     input.CurrentInterval,
			NewEase:         input.CurrentEase,
			NewLearningStep: nextStep,
			NextReviewAt:    input.Now.Add(steps[nextStep]),
		}

	case domain.ReviewGradeEasy:
		// Graduate with easy_interval
		var ease float64
		if isRelearning {
			ease = input.CurrentEase
		} else {
			ease = input.Config.DefaultEaseFactor
		}
		return graduate(input, input.Config.EasyInterval, ease)

	default:
		return calculateLearning(input)
	}
}

func calculateReview(input SRSInput) SRSOutput {
	newEase := input.CurrentEase
	var newInterval int

	switch input.Grade {
	case domain.ReviewGradeAgain:
		// Lapse → relearning
		newEase = maxFloat(input.Config.MinEaseFactor, input.CurrentEase-0.20)
		lapseInterval := int(float64(input.CurrentInterval) * input.Config.LapseNewInterval)
		newInterval = max(1, lapseInterval)

		steps := input.Config.RelearningSteps
		if len(steps) == 0 {
			steps = []time.Duration{10 * time.Minute}
		}

		return SRSOutput{
			NewStatus:       domain.LearningStatusLearning,
			NewInterval:     newInterval,
			NewEase:         newEase,
			NewLearningStep: 0,
			NextReviewAt:    input.Now.Add(steps[0]),
		}

	case domain.ReviewGradeHard:
		// HARD: ease -0.15, interval × hard_modifier
		newEase = maxFloat(input.Config.MinEaseFactor, input.CurrentEase-0.15)
		hardInterval := float64(input.CurrentInterval) * input.Config.HardIntervalModifier
		newInterval = max(input.CurrentInterval+1, int(hardInterval))

	case domain.ReviewGradeGood:
		// GOOD: no ease change, interval × ease × modifier
		goodInterval := float64(input.CurrentInterval) * input.CurrentEase * input.Config.IntervalModifier
		newInterval = max(input.CurrentInterval+1, int(goodInterval))

	case domain.ReviewGradeEasy:
		// EASY: ease +0.15, interval × ease × easy_bonus × modifier
		newEase = input.CurrentEase + 0.15
		easyInterval := float64(input.CurrentInterval) * input.CurrentEase * input.Config.EasyBonus * input.Config.IntervalModifier
		newInterval = max(input.CurrentInterval+1, int(easyInterval))

	default:
		newInterval = input.CurrentInterval
	}

	// Cap at max interval
	newInterval = min(newInterval, input.MaxIntervalDays)

	// Apply fuzz
	newInterval = applyFuzz(newInterval, input.Now)

	// Determine if MASTERED
	newStatus := domain.LearningStatusReview
	if newInterval >= 21 && newEase >= 2.5 {
		newStatus = domain.LearningStatusMastered
	}

	return SRSOutput{
		NewStatus:       newStatus,
		NewInterval:     newInterval,
		NewEase:         newEase,
		NewLearningStep: 0,
		NextReviewAt:    input.Now.Add(time.Duration(newInterval) * 24 * time.Hour),
	}
}

func graduate(input SRSInput, intervalDays int, ease float64) SRSOutput {
	interval := min(intervalDays, input.MaxIntervalDays)
	return SRSOutput{
		NewStatus:       domain.LearningStatusReview,
		NewInterval:     interval,
		NewEase:         ease,
		NewLearningStep: 0,
		NextReviewAt:    input.Now.Add(time.Duration(interval) * 24 * time.Hour),
	}
}

// applyFuzz adds deterministic jitter to prevent card clustering.
// Only applied to intervals >= 3 days. Range: ±5%.
func applyFuzz(interval int, now time.Time) int {
	if interval < 3 {
		return interval
	}
	fuzzRange := max(1, interval*5/100)
	// Deterministic: based on interval only, not timestamp
	seed := interval
	fuzzDays := seed%(fuzzRange*2+1) - fuzzRange
	result := interval + fuzzDays
	if result < 1 {
		return 1
	}
	return result
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
