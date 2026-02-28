package fsrs

import (
	"fmt"
	"time"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// Card holds the FSRS state of a flashcard.
type Card struct {
	State         domain.CardState
	Step          int
	Stability     float64
	Difficulty    float64
	Due           time.Time
	LastReview    *time.Time
	Reps          int
	Lapses        int
	ScheduledDays int
	ElapsedDays   int
}

// Parameters holds all FSRS configuration.
type Parameters struct {
	W                [19]float64
	DesiredRetention float64
	MaxIntervalDays  int
	EnableFuzz       bool
	LearningSteps    []time.Duration
	RelearningSteps  []time.Duration
}

// DefaultParameters returns sensible defaults.
func DefaultParameters() Parameters {
	return Parameters{
		W:                DefaultWeights,
		DesiredRetention: 0.9,
		MaxIntervalDays:  365,
		EnableFuzz:       true,
		LearningSteps:    []time.Duration{time.Minute, 10 * time.Minute},
		RelearningSteps:  []time.Duration{10 * time.Minute},
	}
}

// ReviewCard is the main entry point: given current card state, rating, and time,
// return the updated card.
func ReviewCard(params Parameters, card Card, rating Rating, now time.Time) (Card, error) {
	switch card.State {
	case domain.CardStateNew:
		return reviewNew(params, card, rating, now), nil
	case domain.CardStateLearning:
		return reviewLearning(params, card, rating, now, false), nil
	case domain.CardStateRelearning:
		return reviewLearning(params, card, rating, now, true), nil
	case domain.CardStateReview:
		return reviewReview(params, card, rating, now), nil
	default:
		return Card{}, fmt.Errorf("unknown card state: %q", card.State)
	}
}

// reviewNew handles a NEW card's first review.
func reviewNew(params Parameters, card Card, rating Rating, now time.Time) Card {
	card.Reps++
	card.LastReview = &now

	s := InitialStability(params.W, rating)
	d := InitialDifficulty(params.W, rating)

	card.Stability = s
	card.Difficulty = d

	steps := params.LearningSteps
	if len(steps) == 0 {
		steps = []time.Duration{time.Minute}
	}

	switch rating {
	case Again:
		card.State = domain.CardStateLearning
		card.Step = 0
		card.ScheduledDays = 0
		card.ElapsedDays = 0
		card.Due = now.Add(steps[0])

	case Hard:
		card.State = domain.CardStateLearning
		card.Step = 0
		card.ScheduledDays = 0
		card.ElapsedDays = 0
		// Hard: avg of step 0 and step 1
		var delay time.Duration
		if len(steps) > 1 {
			delay = (steps[0] + steps[1]) / 2
		} else {
			delay = steps[0]
		}
		card.Due = now.Add(delay)

	case Good:
		if len(steps) > 1 {
			card.State = domain.CardStateLearning
			card.Step = 1
			card.ScheduledDays = 0
			card.ElapsedDays = 0
			card.Due = now.Add(steps[1])
		} else {
			// Graduate immediately
			card = graduateToReview(params, card, s, d, now)
		}

	case Easy:
		// Graduate with easy bonus
		card = graduateToReview(params, card, s, d, now)
		// Use Good stability (not Easy) as the baseline for the minimum interval
		goodS := InitialStability(params.W, Good)
		goodInterval := NextInterval(goodS, params.DesiredRetention)
		goodInterval = clampInterval(goodInterval, params.MaxIntervalDays)
		if card.ScheduledDays <= goodInterval {
			card.ScheduledDays = goodInterval + 1
			card.ScheduledDays = clampInterval(card.ScheduledDays, params.MaxIntervalDays)
			card.Due = now.Add(time.Duration(card.ScheduledDays) * 24 * time.Hour)
		}
	}

	return card
}

// reviewLearning handles LEARNING or RELEARNING cards.
func reviewLearning(params Parameters, card Card, rating Rating, now time.Time, isRelearning bool) Card {
	card.Reps++
	card.LastReview = &now

	steps := params.LearningSteps
	if isRelearning {
		steps = params.RelearningSteps
	}
	if len(steps) == 0 {
		steps = []time.Duration{time.Minute}
	}

	// Snapshot pre-update stability for interval ordering (Easy vs Good comparison).
	preS := card.Stability

	// Update S/D for every rating (FSRS-5 spec: short-term stability applies to all ratings).
	card.Stability = ShortTermStability(params.W, card.Stability, rating)
	card.Difficulty = NextDifficulty(params.W, card.Difficulty, rating)

	switch rating {
	case Again:
		// Reset to step 0. Lapses are only incremented on REVIEW → RELEARNING transition.
		card.Step = 0
		card.ElapsedDays = 0
		card.ScheduledDays = 0
		card.Due = now.Add(steps[0])

	case Hard:
		// Repeat current step
		step := card.Step
		if step >= len(steps) {
			step = len(steps) - 1
		}
		card.ElapsedDays = 0
		card.ScheduledDays = 0
		card.Due = now.Add(steps[step])

	case Good:
		nextStep := card.Step + 1
		if nextStep >= len(steps) {
			// Graduate — S/D already updated above.
			card = graduateToReview(params, card, card.Stability, card.Difficulty, now)
		} else {
			card.Step = nextStep
			card.ElapsedDays = 0
			card.ScheduledDays = 0
			card.Due = now.Add(steps[nextStep])
		}

	case Easy:
		// Graduate immediately — S/D already updated above.
		card = graduateToReview(params, card, card.Stability, card.Difficulty, now)

		// For Easy from learning, ensure easyInterval >= goodInterval + 1.
		// Use pre-update stability to compute what Good would have produced.
		goodS := ShortTermStability(params.W, preS, Good)
		goodInterval := NextInterval(goodS, params.DesiredRetention)
		goodInterval = clampInterval(goodInterval, params.MaxIntervalDays)
		if card.ScheduledDays <= goodInterval {
			card.ScheduledDays = goodInterval + 1
			card.ScheduledDays = clampInterval(card.ScheduledDays, params.MaxIntervalDays)
			card.Due = now.Add(time.Duration(card.ScheduledDays) * 24 * time.Hour)
		}
	}

	return card
}

// reviewReview handles REVIEW cards. Computes all 4 outcomes and enforces interval ordering.
func reviewReview(params Parameters, card Card, rating Rating, now time.Time) Card {
	card.Reps++
	card.LastReview = &now

	elapsedDays := card.ElapsedDays
	if elapsedDays < 1 {
		elapsedDays = 1
	}

	r := Retrievability(elapsedDays, card.Stability)

	// Use PRE-UPDATE difficulty for all stability calculations (FSRS-5 spec).
	preD := card.Difficulty

	// Update difficulty with chosen rating.
	d := NextDifficulty(params.W, card.Difficulty, rating)

	if rating == Again {
		// Lapse: capped forget stability
		card.Lapses++
		card.State = domain.CardStateRelearning
		card.Step = 0
		card.Difficulty = d

		newS := StabilityAfterForgettingCapped(params.W, card.Stability, preD, r)
		card.Stability = newS

		steps := params.RelearningSteps
		if len(steps) == 0 {
			steps = []time.Duration{10 * time.Minute}
		}

		card.ElapsedDays = 0
		card.ScheduledDays = 0
		card.Due = now.Add(steps[0])
		return card
	}

	// Compute all recall stabilities using PRE-UPDATE difficulty.
	hardS := StabilityAfterRecall(params.W, card.Stability, preD, r, Hard)
	goodS := StabilityAfterRecall(params.W, card.Stability, preD, r, Good)
	easyS := StabilityAfterRecall(params.W, card.Stability, preD, r, Easy)

	// Raw intervals
	hardIvl := NextInterval(hardS, params.DesiredRetention)
	goodIvl := NextInterval(goodS, params.DesiredRetention)
	easyIvl := NextInterval(easyS, params.DesiredRetention)

	// Clamp to max interval
	hardIvl = clampInterval(hardIvl, params.MaxIntervalDays)
	goodIvl = clampInterval(goodIvl, params.MaxIntervalDays)
	easyIvl = clampInterval(easyIvl, params.MaxIntervalDays)

	// Enforce interval ordering: Hard <= Good < Easy
	if hardIvl > goodIvl {
		hardIvl = goodIvl
	}
	if goodIvl <= hardIvl {
		goodIvl = hardIvl + 1
	}
	if easyIvl <= goodIvl {
		easyIvl = goodIvl + 1
	}

	// Re-clamp after ordering adjustments
	hardIvl = clampInterval(hardIvl, params.MaxIntervalDays)
	goodIvl = clampInterval(goodIvl, params.MaxIntervalDays)
	easyIvl = clampInterval(easyIvl, params.MaxIntervalDays)

	// Apply fuzz if enabled
	if params.EnableFuzz {
		maxIvl := float64(params.MaxIntervalDays)
		ed := float64(elapsedDays)
		seed := FuzzSeed(now, card.Reps, card.Difficulty, card.Stability)

		hardIvl = int(applyFuzz(float64(hardIvl), ed, maxIvl, seed))
		goodIvl = int(applyFuzz(float64(goodIvl), ed, maxIvl, seed+1))
		easyIvl = int(applyFuzz(float64(easyIvl), ed, maxIvl, seed+2))

		// Re-enforce ordering after fuzz
		if hardIvl > goodIvl {
			hardIvl = goodIvl
		}
		if goodIvl <= hardIvl {
			goodIvl = hardIvl + 1
		}
		if easyIvl <= goodIvl {
			easyIvl = goodIvl + 1
		}
	}

	// Set difficulty after all stability calculations are done.
	card.Difficulty = d

	// Select the interval for the chosen rating
	var chosenIvl int
	var chosenS float64
	switch rating {
	case Hard:
		chosenIvl = hardIvl
		chosenS = hardS
	case Good:
		chosenIvl = goodIvl
		chosenS = goodS
	case Easy:
		chosenIvl = easyIvl
		chosenS = easyS
	}

	chosenIvl = clampInterval(chosenIvl, params.MaxIntervalDays)

	card.Stability = chosenS
	card.State = domain.CardStateReview
	card.ScheduledDays = chosenIvl
	card.ElapsedDays = 0
	card.Due = now.Add(time.Duration(chosenIvl) * 24 * time.Hour)

	return card
}

// graduateToReview transitions a card from Learning/New to Review.
func graduateToReview(params Parameters, card Card, stability, difficulty float64, now time.Time) Card {
	card.State = domain.CardStateReview
	card.Step = 0
	card.Stability = stability
	card.Difficulty = difficulty

	interval := NextInterval(stability, params.DesiredRetention)
	interval = clampInterval(interval, params.MaxIntervalDays)

	card.ScheduledDays = interval
	card.ElapsedDays = 0
	card.Due = now.Add(time.Duration(interval) * 24 * time.Hour)

	return card
}

// clampInterval constrains an interval to [1, maxDays].
func clampInterval(interval, maxDays int) int {
	if interval < 1 {
		return 1
	}
	if interval > maxDays {
		return maxDays
	}
	return interval
}
