# FSRS-5 Migration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace SM-2 spaced repetition with own FSRS-5 implementation — new algorithm package, updated domain models, DB migration, and service layer changes.

**Architecture:** Separate `internal/service/study/fsrs/` package with pure FSRS-5 algorithm functions. Domain models get new fields (stability, difficulty, reps, lapses). Study service calls the new package instead of `CalculateSRS()`. Complete replacement, no backward compatibility.

**Tech Stack:** Go, PostgreSQL (goose migrations), sqlc, gqlgen, testcontainers (E2E)

---

### Task 1: FSRS-5 Algorithm — Core Formulas

**Files:**
- Create: `backend_v4/internal/service/study/fsrs/algorithm.go`
- Create: `backend_v4/internal/service/study/fsrs/algorithm_test.go`

**Step 1: Create the `fsrs` package with types and constants**

Create `backend_v4/internal/service/study/fsrs/algorithm.go`:

```go
package fsrs

import (
	"math"
	"time"
)

// Rating represents the user's recall quality (matches domain.ReviewGrade values).
type Rating int

const (
	Again Rating = 1
	Hard  Rating = 2
	Good  Rating = 3
	Easy  Rating = 4
)

// State represents the card's learning state.
type State int

const (
	New        State = 0
	Learning   State = 1
	Review     State = 2
	Relearning State = 3
)

// Card holds the FSRS memory state of a flashcard.
type Card struct {
	State         State
	Step          int       // current learning/relearning step index
	Stability     float64   // S: interval in days for R to drop to 90%
	Difficulty    float64   // D: inherent difficulty [1, 10]
	Due           time.Time // when the card should next be reviewed
	LastReview    time.Time // timestamp of last review
	Reps          int       // total successful reviews (not Again)
	Lapses        int       // total lapses (Again while in Review)
	ScheduledDays int       // planned interval in days
	ElapsedDays   int       // actual days since last review
}

// FSRS-5 default weights (19 parameters).
var DefaultWeights = [19]float64{
	0.40255, 1.18385, 3.173, 15.69105, // w0-w3: initial stability per rating
	7.1949,                              // w4: initial difficulty base
	0.5345,                              // w5: initial difficulty scaling
	1.4604,                              // w6: difficulty update factor
	0.0046,                              // w7: mean reversion weight
	1.54575,                             // w8: recall stability factor
	0.1192,                              // w9: recall stability power (S)
	1.01925,                             // w10: recall stability power (R)
	1.9395,                              // w11: forget stability factor
	0.11,                                // w12: forget stability power (D)
	0.29605,                             // w13: forget stability power (S)
	2.2698,                              // w14: forget stability power (R)
	0.2315,                              // w15: hard penalty
	2.9898,                              // w16: easy bonus
	0.51655,                             // w17: short-term stability factor
	0.6621,                              // w18: short-term stability offset
}

const (
	Decay  = -0.5
	Factor = 19.0 / 81.0 // ensures R(S,S) = 0.9

	MinDifficulty = 1.0
	MaxDifficulty = 10.0
	MinStability  = 0.001
)

// --- Core FSRS-5 formulas (pure functions) ---

// Retrievability returns the probability of recall after t days with stability S.
// R(t, S) = (1 + Factor * t/S) ^ Decay
func Retrievability(elapsedDays int, stability float64) float64 {
	if stability < MinStability {
		return 0
	}
	return math.Pow(1+Factor*float64(elapsedDays)/stability, Decay)
}

// NextInterval calculates the optimal interval for a desired retention rate.
// I(r, S) = (S / Factor) * (r^(1/Decay) - 1)
func NextInterval(desiredRetention, stability float64) int {
	if stability < MinStability || desiredRetention <= 0 || desiredRetention >= 1 {
		return 1
	}
	interval := stability / Factor * (math.Pow(desiredRetention, 1.0/Decay) - 1)
	return max(1, int(math.Round(interval)))
}

// InitialStability returns S0(G) = w[G-1] for the first review.
func InitialStability(w [19]float64, rating Rating) float64 {
	idx := int(rating) - 1
	if idx < 0 || idx > 3 {
		idx = 2 // default to Good
	}
	return math.Max(MinStability, w[idx])
}

// InitialDifficulty returns D0(G) = w4 - e^(w5*(G-1)) + 1, clamped to [1, 10].
func InitialDifficulty(w [19]float64, rating Rating) float64 {
	d := w[4] - math.Exp(w[5]*float64(rating-1)) + 1
	return clampDifficulty(d)
}

// NextDifficulty updates difficulty after a review.
// ΔD = -w6*(G-3), D' = D + ΔD*(10-D)/9, D'' = w7*D0(4) + (1-w7)*D' (mean reversion)
func NextDifficulty(w [19]float64, d float64, rating Rating) float64 {
	delta := -w[6] * float64(rating-3)
	dPrime := d + delta*(MaxDifficulty-d)/9.0
	dTarget := InitialDifficulty(w, Easy)
	dFinal := w[7]*dTarget + (1-w[7])*dPrime
	return clampDifficulty(dFinal)
}

// StabilityAfterRecall returns S'r for successful recall (rating >= Hard).
// S'r = S * (e^w8 * (11-D) * S^(-w9) * (e^(w10*(1-R)) - 1) * hardPenalty * easyBonus + 1)
func StabilityAfterRecall(w [19]float64, s, d, r float64, rating Rating) float64 {
	hardPenalty := 1.0
	easyBonus := 1.0
	if rating == Hard {
		hardPenalty = w[15]
	} else if rating == Easy {
		easyBonus = w[16]
	}
	newS := s * (math.Exp(w[8]) * (11 - d) * math.Pow(s, -w[9]) *
		(math.Exp(w[10]*(1-r)) - 1) * hardPenalty * easyBonus + 1)
	return math.Max(MinStability, newS)
}

// StabilityAfterForgetting returns S'f for a lapse (Again while in Review).
// S'f = w11 * D^(-w12) * ((S+1)^w13 - 1) * e^(w14*(1-R))
func StabilityAfterForgetting(w [19]float64, s, d, r float64) float64 {
	newS := w[11] * math.Pow(d, -w[12]) * (math.Pow(s+1, w[13]) - 1) *
		math.Exp(w[14]*(1-r))
	return math.Max(MinStability, newS)
}

// ShortTermStability updates stability for same-day/learning reviews.
// S' = S * e^(w17 * (G - 3 + w18))
func ShortTermStability(w [19]float64, s float64, rating Rating) float64 {
	newS := s * math.Exp(w[17]*(float64(rating)-3+w[18]))
	return math.Max(MinStability, newS)
}

func clampDifficulty(d float64) float64 {
	if d < MinDifficulty {
		return MinDifficulty
	}
	if d > MaxDifficulty {
		return MaxDifficulty
	}
	return d
}
```

**Step 2: Write unit tests for all formulas**

Create `backend_v4/internal/service/study/fsrs/algorithm_test.go`:

```go
package fsrs

import (
	"math"
	"testing"
)

func TestRetrievability(t *testing.T) {
	tests := []struct {
		name     string
		elapsed  int
		stab     float64
		wantMin  float64
		wantMax  float64
	}{
		{"at stability, R~0.9", 1, 1.0, 0.89, 0.91},
		{"zero elapsed, R=1", 0, 10.0, 0.99, 1.01},
		{"high elapsed, low R", 100, 1.0, 0.0, 0.15},
		{"zero stability", 1, 0.0, -0.01, 0.01},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Retrievability(tt.elapsed, tt.stab)
			if r < tt.wantMin || r > tt.wantMax {
				t.Errorf("Retrievability(%d, %f) = %f, want [%f, %f]",
					tt.elapsed, tt.stab, r, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestNextInterval(t *testing.T) {
	tests := []struct {
		name      string
		retention float64
		stability float64
		wantMin   int
		wantMax   int
	}{
		{"R=0.9 S=1 → ~1d", 0.9, 1.0, 1, 1},
		{"R=0.9 S=10 → ~10d", 0.9, 10.0, 9, 11},
		{"R=0.9 S=100 → ~100d", 0.9, 100.0, 95, 105},
		{"lower retention → longer", 0.8, 10.0, 11, 20},
		{"higher retention → shorter", 0.95, 10.0, 3, 8},
		{"edge: R=0", 0.0, 10.0, 1, 1},
		{"edge: R=1", 1.0, 10.0, 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interval := NextInterval(tt.retention, tt.stability)
			if interval < tt.wantMin || interval > tt.wantMax {
				t.Errorf("NextInterval(%f, %f) = %d, want [%d, %d]",
					tt.retention, tt.stability, interval, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestInitialStability(t *testing.T) {
	w := DefaultWeights
	for _, r := range []Rating{Again, Hard, Good, Easy} {
		s := InitialStability(w, r)
		if s < MinStability {
			t.Errorf("InitialStability(rating=%d) = %f, want >= %f", r, s, MinStability)
		}
	}
	// S increases with better rating
	sAgain := InitialStability(w, Again)
	sEasy := InitialStability(w, Easy)
	if sAgain >= sEasy {
		t.Errorf("S(Again)=%f should be < S(Easy)=%f", sAgain, sEasy)
	}
}

func TestInitialDifficulty(t *testing.T) {
	w := DefaultWeights
	dAgain := InitialDifficulty(w, Again)
	dEasy := InitialDifficulty(w, Easy)
	if dAgain <= dEasy {
		t.Errorf("D(Again)=%f should be > D(Easy)=%f", dAgain, dEasy)
	}
	for _, r := range []Rating{Again, Hard, Good, Easy} {
		d := InitialDifficulty(w, r)
		if d < MinDifficulty || d > MaxDifficulty {
			t.Errorf("InitialDifficulty(rating=%d) = %f, want [%f, %f]", r, d, MinDifficulty, MaxDifficulty)
		}
	}
}

func TestNextDifficulty(t *testing.T) {
	w := DefaultWeights
	d := 5.0
	// Again increases difficulty
	dAfterAgain := NextDifficulty(w, d, Again)
	if dAfterAgain <= d {
		t.Errorf("NextDifficulty(Again) = %f, want > %f", dAfterAgain, d)
	}
	// Easy decreases difficulty
	dAfterEasy := NextDifficulty(w, d, Easy)
	if dAfterEasy >= d {
		t.Errorf("NextDifficulty(Easy) = %f, want < %f", dAfterEasy, d)
	}
	// Stays clamped
	dMin := NextDifficulty(w, MinDifficulty, Easy)
	if dMin < MinDifficulty {
		t.Errorf("NextDifficulty at min = %f, want >= %f", dMin, MinDifficulty)
	}
	dMax := NextDifficulty(w, MaxDifficulty, Again)
	if dMax > MaxDifficulty {
		t.Errorf("NextDifficulty at max = %f, want <= %f", dMax, MaxDifficulty)
	}
}

func TestStabilityAfterRecall(t *testing.T) {
	w := DefaultWeights
	s, d, r := 10.0, 5.0, 0.9
	sGood := StabilityAfterRecall(w, s, d, r, Good)
	if sGood <= s {
		t.Errorf("StabilityAfterRecall(Good) = %f, want > %f", sGood, s)
	}
	sHard := StabilityAfterRecall(w, s, d, r, Hard)
	sEasy := StabilityAfterRecall(w, s, d, r, Easy)
	if sHard >= sGood {
		t.Errorf("S(Hard)=%f should be < S(Good)=%f", sHard, sGood)
	}
	if sEasy <= sGood {
		t.Errorf("S(Easy)=%f should be > S(Good)=%f", sEasy, sGood)
	}
}

func TestStabilityAfterForgetting(t *testing.T) {
	w := DefaultWeights
	s, d, r := 10.0, 5.0, 0.3
	sf := StabilityAfterForgetting(w, s, d, r)
	if sf >= s {
		t.Errorf("StabilityAfterForgetting = %f, want < %f (original stability)", sf, s)
	}
	if sf < MinStability {
		t.Errorf("StabilityAfterForgetting = %f, want >= %f", sf, MinStability)
	}
}

func TestShortTermStability(t *testing.T) {
	w := DefaultWeights
	s := 1.0
	sGood := ShortTermStability(w, s, Good)
	sAgain := ShortTermStability(w, s, Again)
	if sAgain >= s {
		t.Errorf("ShortTerm(Again) = %f, want < %f", sAgain, s)
	}
	_ = sGood // Good may increase or decrease depending on w18
}

func almostEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}
```

**Step 3: Run tests to verify they pass**

Run: `cd backend_v4 && go test ./internal/service/study/fsrs/ -v -race -count=1`
Expected: All PASS

**Step 4: Commit**

```bash
git add backend_v4/internal/service/study/fsrs/
git commit -m "feat(fsrs): add FSRS-5 core algorithm formulas with tests"
```

---

### Task 2: FSRS-5 Algorithm — Scheduler

**Files:**
- Create: `backend_v4/internal/service/study/fsrs/scheduler.go`
- Create: `backend_v4/internal/service/study/fsrs/scheduler_test.go`

**Step 1: Create the scheduler**

Create `backend_v4/internal/service/study/fsrs/scheduler.go`:

```go
package fsrs

import (
	"math"
	"math/rand"
	"time"
)

// Parameters configures the FSRS-5 scheduler.
type Parameters struct {
	W                [19]float64
	RequestRetention float64       // desired retention (0.7–0.97), default 0.9
	MaximumInterval  int           // max interval in days, default 365
	EnableFuzz       bool          // interval fuzzing for ≥3 day intervals
	EnableShortTerm  bool          // use learning steps for New/Relearning
	LearningSteps    []time.Duration // e.g. [1m, 10m]
	RelearningSteps  []time.Duration // e.g. [10m]
}

// DefaultParameters returns sensible defaults.
func DefaultParameters() Parameters {
	return Parameters{
		W:                DefaultWeights,
		RequestRetention: 0.9,
		MaximumInterval:  365,
		EnableFuzz:       true,
		EnableShortTerm:  true,
		LearningSteps:    []time.Duration{1 * time.Minute, 10 * time.Minute},
		RelearningSteps:  []time.Duration{10 * time.Minute},
	}
}

// SchedulingResult holds the output of a review.
type SchedulingResult struct {
	Card Card
}

// ReviewCard processes a review and returns the updated card.
// This is the main entry point for the FSRS scheduler.
func ReviewCard(params Parameters, card Card, rating Rating, now time.Time) Card {
	result := card // copy

	elapsedDays := 0
	if !card.LastReview.IsZero() {
		elapsedDays = int(math.Round(now.Sub(card.LastReview).Hours() / 24))
		if elapsedDays < 0 {
			elapsedDays = 0
		}
	}
	result.ElapsedDays = elapsedDays

	switch card.State {
	case New:
		result = reviewNew(params, result, rating, now)
	case Learning, Relearning:
		result = reviewLearning(params, result, rating, now)
	case Review:
		result = reviewReview(params, result, rating, now, elapsedDays)
	}

	result.LastReview = now
	return result
}

func reviewNew(params Parameters, card Card, rating Rating, now time.Time) Card {
	card.Stability = InitialStability(params.W, rating)
	card.Difficulty = InitialDifficulty(params.W, rating)
	card.Reps = 0
	card.Lapses = 0

	if params.EnableShortTerm && len(params.LearningSteps) > 0 {
		// Enter learning phase with steps
		switch rating {
		case Again:
			card.State = Learning
			card.Step = 0
			card.Due = now.Add(params.LearningSteps[0])
			card.ScheduledDays = 0
		case Hard:
			card.State = Learning
			card.Step = 0
			if len(params.LearningSteps) > 1 {
				card.Due = now.Add((params.LearningSteps[0] + params.LearningSteps[1]) / 2)
			} else {
				card.Due = now.Add(params.LearningSteps[0])
			}
			card.ScheduledDays = 0
		case Good:
			if len(params.LearningSteps) > 1 {
				card.State = Learning
				card.Step = 1
				card.Due = now.Add(params.LearningSteps[1])
				card.ScheduledDays = 0
			} else {
				// Graduate immediately
				card = graduateToReview(params, card, now)
			}
		case Easy:
			card = graduateToReview(params, card, now)
		}
	} else {
		// Long-term scheduler: go directly to Review
		card = graduateToReview(params, card, now)
	}

	return card
}

func reviewLearning(params Parameters, card Card, rating Rating, now time.Time) Card {
	isRelearning := card.State == Relearning
	var steps []time.Duration
	if isRelearning {
		steps = params.RelearningSteps
	} else {
		steps = params.LearningSteps
	}
	if len(steps) == 0 {
		steps = []time.Duration{1 * time.Minute}
	}

	// Update stability for short-term review
	card.Stability = ShortTermStability(params.W, card.Stability, rating)
	card.Difficulty = NextDifficulty(params.W, card.Difficulty, rating)

	switch rating {
	case Again:
		card.Step = 0
		card.Due = now.Add(steps[0])
		card.ScheduledDays = 0
	case Hard:
		step := card.Step
		if step >= len(steps) {
			step = len(steps) - 1
		}
		card.Due = now.Add(steps[step])
		card.ScheduledDays = 0
	case Good:
		nextStep := card.Step + 1
		if nextStep >= len(steps) {
			card = graduateToReview(params, card, now)
		} else {
			card.Step = nextStep
			card.Due = now.Add(steps[nextStep])
			card.ScheduledDays = 0
		}
	case Easy:
		card = graduateToReview(params, card, now)
	}

	return card
}

func reviewReview(params Parameters, card Card, rating Rating, now time.Time, elapsedDays int) Card {
	r := Retrievability(elapsedDays, card.Stability)

	switch rating {
	case Again:
		card.Stability = StabilityAfterForgetting(params.W, card.Stability, card.Difficulty, r)
		card.Difficulty = NextDifficulty(params.W, card.Difficulty, rating)
		card.Lapses++

		if params.EnableShortTerm && len(params.RelearningSteps) > 0 {
			card.State = Relearning
			card.Step = 0
			card.Due = now.Add(params.RelearningSteps[0])
			card.ScheduledDays = 0
		} else {
			// No relearning steps: stay in Review with new stability
			interval := NextInterval(params.RequestRetention, card.Stability)
			interval = min(interval, params.MaximumInterval)
			if params.EnableFuzz {
				interval = applyFuzz(interval, elapsedDays)
			}
			card.ScheduledDays = interval
			card.Due = now.AddDate(0, 0, interval)
		}

	case Hard, Good, Easy:
		card.Stability = StabilityAfterRecall(params.W, card.Stability, card.Difficulty, r, rating)
		card.Difficulty = NextDifficulty(params.W, card.Difficulty, rating)
		card.Reps++
		card.State = Review

		interval := NextInterval(params.RequestRetention, card.Stability)
		interval = max(interval, card.ElapsedDays+1) // at least 1 day more than elapsed
		interval = min(interval, params.MaximumInterval)
		if params.EnableFuzz {
			interval = applyFuzz(interval, elapsedDays)
		}
		card.ScheduledDays = interval
		card.Due = now.AddDate(0, 0, interval)
	}

	return card
}

func graduateToReview(params Parameters, card Card, now time.Time) Card {
	card.State = Review
	card.Step = 0

	interval := NextInterval(params.RequestRetention, card.Stability)
	interval = min(interval, params.MaximumInterval)
	if params.EnableFuzz {
		interval = applyFuzz(interval, card.ElapsedDays)
	}
	interval = max(1, interval)

	card.ScheduledDays = interval
	card.Due = now.AddDate(0, 0, interval)
	return card
}

// applyFuzz adds deterministic jitter to prevent card clustering.
// Only applied to intervals >= 3 days. Range: ±5%.
func applyFuzz(interval, elapsedDays int) int {
	if interval < 3 {
		return interval
	}
	fuzzRange := max(1, interval*5/100)
	// Use a seeded RNG for deterministic but varied results
	rng := rand.New(rand.NewSource(int64(interval*1000 + elapsedDays)))
	fuzzDays := rng.Intn(fuzzRange*2+1) - fuzzRange
	result := interval + fuzzDays
	if result < 1 {
		return 1
	}
	return result
}
```

**Step 2: Write scheduler tests**

Create `backend_v4/internal/service/study/fsrs/scheduler_test.go`:

```go
package fsrs

import (
	"testing"
	"time"
)

func TestReviewCard_NewToLearning(t *testing.T) {
	params := DefaultParameters()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	card := Card{State: New}

	// Again → Learning step 0
	c := ReviewCard(params, card, Again, now)
	if c.State != Learning {
		t.Errorf("State = %d, want Learning(%d)", c.State, Learning)
	}
	if c.Step != 0 {
		t.Errorf("Step = %d, want 0", c.Step)
	}
	if c.Due != now.Add(1*time.Minute) {
		t.Errorf("Due = %v, want %v", c.Due, now.Add(1*time.Minute))
	}

	// Good with 2 steps → Learning step 1
	c = ReviewCard(params, card, Good, now)
	if c.State != Learning {
		t.Errorf("State = %d, want Learning(%d)", c.State, Learning)
	}
	if c.Step != 1 {
		t.Errorf("Step = %d, want 1", c.Step)
	}
	if c.Due != now.Add(10*time.Minute) {
		t.Errorf("Due = %v, want %v", c.Due, now.Add(10*time.Minute))
	}

	// Easy → directly to Review
	c = ReviewCard(params, card, Easy, now)
	if c.State != Review {
		t.Errorf("State = %d, want Review(%d)", c.State, Review)
	}
	if c.ScheduledDays < 1 {
		t.Errorf("ScheduledDays = %d, want >= 1", c.ScheduledDays)
	}
}

func TestReviewCard_LearningGraduation(t *testing.T) {
	params := DefaultParameters()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	// Start as Learning, step 1 (last step with 2 steps)
	card := Card{
		State:      Learning,
		Step:       1,
		Stability:  1.0,
		Difficulty: 5.0,
		LastReview: now.Add(-10 * time.Minute),
	}

	// Good at last step → graduate to Review
	c := ReviewCard(params, card, Good, now)
	if c.State != Review {
		t.Errorf("State = %d, want Review(%d)", c.State, Review)
	}
	if c.ScheduledDays < 1 {
		t.Errorf("ScheduledDays = %d, want >= 1", c.ScheduledDays)
	}
}

func TestReviewCard_ReviewSuccess(t *testing.T) {
	params := DefaultParameters()
	params.EnableFuzz = false
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	card := Card{
		State:      Review,
		Stability:  10.0,
		Difficulty: 5.0,
		LastReview: now.AddDate(0, 0, -10),
		Reps:       3,
	}

	c := ReviewCard(params, card, Good, now)
	if c.State != Review {
		t.Errorf("State = %d, want Review", c.State)
	}
	if c.Stability <= card.Stability {
		t.Errorf("Stability = %f, want > %f (should increase after Good)", c.Stability, card.Stability)
	}
	if c.Reps != 4 {
		t.Errorf("Reps = %d, want 4", c.Reps)
	}
	if c.ScheduledDays <= 10 {
		t.Errorf("ScheduledDays = %d, want > 10 (stability grew)", c.ScheduledDays)
	}
}

func TestReviewCard_ReviewLapse(t *testing.T) {
	params := DefaultParameters()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	card := Card{
		State:      Review,
		Stability:  20.0,
		Difficulty: 5.0,
		LastReview: now.AddDate(0, 0, -20),
		Reps:       5,
		Lapses:     1,
	}

	c := ReviewCard(params, card, Again, now)
	if c.State != Relearning {
		t.Errorf("State = %d, want Relearning(%d)", c.State, Relearning)
	}
	if c.Lapses != 2 {
		t.Errorf("Lapses = %d, want 2", c.Lapses)
	}
	if c.Stability >= card.Stability {
		t.Errorf("Stability = %f, want < %f (should decrease after lapse)", c.Stability, card.Stability)
	}
	if c.Step != 0 {
		t.Errorf("Step = %d, want 0", c.Step)
	}
}

func TestReviewCard_RelearningGraduation(t *testing.T) {
	params := DefaultParameters()
	params.RelearningSteps = []time.Duration{10 * time.Minute}
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	card := Card{
		State:      Relearning,
		Step:       0,
		Stability:  3.0,
		Difficulty: 6.0,
		LastReview: now.Add(-10 * time.Minute),
		Lapses:     2,
	}

	// Good at last step → graduate back to Review
	c := ReviewCard(params, card, Good, now)
	if c.State != Review {
		t.Errorf("State = %d, want Review(%d)", c.State, Review)
	}
}

func TestReviewCard_HardVsGoodVsEasy(t *testing.T) {
	params := DefaultParameters()
	params.EnableFuzz = false
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	card := Card{
		State:      Review,
		Stability:  15.0,
		Difficulty: 5.0,
		LastReview: now.AddDate(0, 0, -15),
		Reps:       5,
	}

	cHard := ReviewCard(params, card, Hard, now)
	cGood := ReviewCard(params, card, Good, now)
	cEasy := ReviewCard(params, card, Easy, now)

	if cHard.ScheduledDays >= cGood.ScheduledDays {
		t.Errorf("Hard interval(%d) should be < Good interval(%d)", cHard.ScheduledDays, cGood.ScheduledDays)
	}
	if cGood.ScheduledDays >= cEasy.ScheduledDays {
		t.Errorf("Good interval(%d) should be < Easy interval(%d)", cGood.ScheduledDays, cEasy.ScheduledDays)
	}
}

func TestReviewCard_MaxInterval(t *testing.T) {
	params := DefaultParameters()
	params.MaximumInterval = 30
	params.EnableFuzz = false
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	card := Card{
		State:      Review,
		Stability:  1000.0, // very high stability
		Difficulty: 1.0,
		LastReview: now.AddDate(0, 0, -30),
		Reps:       20,
	}

	c := ReviewCard(params, card, Good, now)
	if c.ScheduledDays > 30 {
		t.Errorf("ScheduledDays = %d, want <= 30 (max interval cap)", c.ScheduledDays)
	}
}

func TestReviewCard_LongTermScheduler(t *testing.T) {
	params := DefaultParameters()
	params.EnableShortTerm = false
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	card := Card{State: New}

	// Without short-term, New goes directly to Review
	c := ReviewCard(params, card, Good, now)
	if c.State != Review {
		t.Errorf("State = %d, want Review (long-term scheduler)", c.State)
	}
	if c.ScheduledDays < 1 {
		t.Errorf("ScheduledDays = %d, want >= 1", c.ScheduledDays)
	}
}
```

**Step 3: Run tests**

Run: `cd backend_v4 && go test ./internal/service/study/fsrs/ -v -race -count=1`
Expected: All PASS

**Step 4: Commit**

```bash
git add backend_v4/internal/service/study/fsrs/
git commit -m "feat(fsrs): add FSRS-5 scheduler with learning steps and tests"
```

---

### Task 3: Domain Model Changes

**Files:**
- Modify: `backend_v4/internal/domain/enums.go:3-21` (replace LearningStatus)
- Modify: `backend_v4/internal/domain/card.go:9-60` (replace Card, CardSnapshot, SRSResult)
- Modify: `backend_v4/internal/domain/study.go:9-73` (replace SRSConfig, SRSUpdateParams, CardStatusCounts, CardStats)
- Modify: `backend_v4/internal/domain/user.go:21-40` (add DesiredRetention to UserSettings)

**Step 1: Replace LearningStatus with CardState in `enums.go`**

Replace lines 3-21 of `enums.go` with:

```go
// CardState represents the FSRS learning state of a card.
type CardState string

const (
	CardStateNew        CardState = "NEW"
	CardStateLearning   CardState = "LEARNING"
	CardStateReview     CardState = "REVIEW"
	CardStateRelearning CardState = "RELEARNING"
)

func (s CardState) String() string { return string(s) }

func (s CardState) IsValid() bool {
	switch s {
	case CardStateNew, CardStateLearning, CardStateReview, CardStateRelearning:
		return true
	}
	return false
}
```

Delete the old `LearningStatus` type entirely. Keep `ReviewGrade` and everything else unchanged.

**Step 2: Replace Card, CardSnapshot, SRSResult in `card.go`**

Replace lines 9-60 of `card.go` with:

```go
// Card represents an FSRS flashcard linked 1:1 with an Entry.
type Card struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	EntryID       uuid.UUID
	State         CardState
	Step          int        // current learning/relearning step index
	Stability     float64    // S: time for R to drop to 90%
	Difficulty    float64    // D: inherent difficulty [1, 10]
	Due           time.Time  // when card should be reviewed
	LastReview    *time.Time // last review timestamp
	Reps          int        // total successful reviews
	Lapses        int        // total lapses (Again in Review)
	ScheduledDays int        // planned interval in days
	ElapsedDays   int        // actual days since last review
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// IsDue returns true if the card needs review at the given time.
func (c *Card) IsDue(now time.Time) bool {
	if c.State == CardStateNew {
		return true
	}
	return !c.Due.After(now)
}

// CardSnapshot captures the FSRS state of a card before a review (for undo).
type CardSnapshot struct {
	State         CardState
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
```

Remove `SRSResult` struct entirely — it's replaced by `fsrs.Card` return from scheduler.

**Step 3: Replace SRSConfig, SRSUpdateParams, CardStatusCounts, CardStats in `study.go`**

Replace entire content of `study.go` with:

```go
package domain

import (
	"time"

	"github.com/google/uuid"
)

// SRSConfig holds FSRS algorithm parameters (pure domain type).
type SRSConfig struct {
	Weights           [19]float64
	DefaultRetention  float64
	MaxIntervalDays   int
	EnableFuzz        bool
	LearningSteps     []time.Duration
	RelearningSteps   []time.Duration
	NewCardsPerDay    int
	ReviewsPerDay     int
	UndoWindowMinutes int
}

// SRSUpdateParams holds the fields to update on a card after FSRS calculation.
type SRSUpdateParams struct {
	State         CardState
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

// CardStatusCounts holds the count of cards per FSRS state.
type CardStatusCounts struct {
	New        int
	Learning   int
	Review     int
	Relearning int
	Total      int
}

// Dashboard holds aggregated study statistics for the user.
type Dashboard struct {
	DueCount      int
	NewCount      int
	ReviewedToday int
	NewToday      int
	Streak        int
	StatusCounts  CardStatusCounts
	OverdueCount  int
	ActiveSession *uuid.UUID
}

// DayReviewCount holds the review count for a specific date.
type DayReviewCount struct {
	Date  time.Time
	Count int
}

// CardStats holds statistics for a single card.
type CardStats struct {
	TotalReviews      int
	AccuracyRate      float64
	AverageTimeMs     *int
	CurrentState      CardState
	Stability         float64
	Difficulty        float64
	GradeDistribution *GradeCounts
}
```

**Step 4: Add DesiredRetention to UserSettings in `user.go`**

Add `DesiredRetention` field to `UserSettings` struct (after `Timezone`):

```go
type UserSettings struct {
	UserID           uuid.UUID
	NewCardsPerDay   int
	ReviewsPerDay    int
	MaxIntervalDays  int
	Timezone         string
	DesiredRetention float64
	UpdatedAt        time.Time
}
```

Update `DefaultUserSettings`:

```go
func DefaultUserSettings(userID uuid.UUID) UserSettings {
	return UserSettings{
		UserID:           userID,
		NewCardsPerDay:   20,
		ReviewsPerDay:    200,
		MaxIntervalDays:  365,
		Timezone:         "UTC",
		DesiredRetention: 0.9,
	}
}
```

**Step 5: Verify compilation**

Run: `cd backend_v4 && go build ./internal/domain/...`
Expected: Compilation errors in consumers (expected — they still reference old types). Domain package itself should compile.

**Step 6: Commit**

```bash
git add backend_v4/internal/domain/
git commit -m "feat(domain): replace SM-2 types with FSRS-5 (CardState, stability, difficulty)"
```

---

### Task 4: Database Migration

**Files:**
- Create: `backend_v4/migrations/00018_fsrs_migration.sql`

**Step 1: Create migration file**

```sql
-- +goose Up

-- 1. Add new card_state enum (with RELEARNING instead of MASTERED)
CREATE TYPE card_state AS ENUM ('NEW', 'LEARNING', 'REVIEW', 'RELEARNING');

-- 2. Add FSRS columns to cards
ALTER TABLE cards
  ADD COLUMN state         card_state NOT NULL DEFAULT 'NEW',
  ADD COLUMN step          INT NOT NULL DEFAULT 0,
  ADD COLUMN stability     FLOAT NOT NULL DEFAULT 0,
  ADD COLUMN difficulty    FLOAT NOT NULL DEFAULT 0,
  ADD COLUMN due           TIMESTAMPTZ NOT NULL DEFAULT now(),
  ADD COLUMN last_review   TIMESTAMPTZ,
  ADD COLUMN reps          INT NOT NULL DEFAULT 0,
  ADD COLUMN lapses        INT NOT NULL DEFAULT 0,
  ADD COLUMN scheduled_days INT NOT NULL DEFAULT 0,
  ADD COLUMN elapsed_days  INT NOT NULL DEFAULT 0;

-- 3. Migrate existing data: map old status → new state
UPDATE cards SET state = 'NEW'       WHERE status = 'NEW';
UPDATE cards SET state = 'LEARNING'  WHERE status = 'LEARNING';
UPDATE cards SET state = 'REVIEW'    WHERE status IN ('REVIEW', 'MASTERED');

-- 4. Copy existing scheduling data where applicable
UPDATE cards SET due = next_review_at WHERE next_review_at IS NOT NULL;
UPDATE cards SET stability = ease_factor WHERE ease_factor > 0;

-- 5. Drop old SM-2 columns
ALTER TABLE cards
  DROP COLUMN status,
  DROP COLUMN learning_step,
  DROP COLUMN next_review_at,
  DROP COLUMN interval_days,
  DROP COLUMN ease_factor;

-- 6. Update indexes
DROP INDEX IF EXISTS ix_cards_user_due;
CREATE INDEX ix_cards_user_due ON cards(user_id, state, due);

-- 7. Add desired_retention to user_settings
ALTER TABLE user_settings
  ADD COLUMN desired_retention FLOAT NOT NULL DEFAULT 0.9;

-- +goose Down

-- Reverse: add old columns, drop new ones
ALTER TABLE user_settings DROP COLUMN IF EXISTS desired_retention;

DROP INDEX IF EXISTS ix_cards_user_due;

ALTER TABLE cards
  ADD COLUMN status learning_status NOT NULL DEFAULT 'NEW',
  ADD COLUMN learning_step INT NOT NULL DEFAULT 0,
  ADD COLUMN next_review_at TIMESTAMPTZ,
  ADD COLUMN interval_days INT NOT NULL DEFAULT 0,
  ADD COLUMN ease_factor FLOAT NOT NULL DEFAULT 2.5;

UPDATE cards SET status = 'NEW'      WHERE state = 'NEW';
UPDATE cards SET status = 'LEARNING' WHERE state IN ('LEARNING', 'RELEARNING');
UPDATE cards SET status = 'REVIEW'   WHERE state = 'REVIEW';
UPDATE cards SET next_review_at = due;
UPDATE cards SET ease_factor = stability WHERE stability > 0;

ALTER TABLE cards
  DROP COLUMN state,
  DROP COLUMN step,
  DROP COLUMN stability,
  DROP COLUMN difficulty,
  DROP COLUMN due,
  DROP COLUMN last_review,
  DROP COLUMN reps,
  DROP COLUMN lapses,
  DROP COLUMN scheduled_days,
  DROP COLUMN elapsed_days;

CREATE INDEX ix_cards_user_due ON cards(user_id, status, next_review_at) WHERE status != 'MASTERED';
DROP TYPE IF EXISTS card_state;
```

**Step 2: Commit**

```bash
git add backend_v4/migrations/00018_fsrs_migration.sql
git commit -m "feat(db): add FSRS migration — card_state enum, stability/difficulty columns"
```

---

### Task 5: Config Layer Changes

**Files:**
- Modify: `backend_v4/internal/config/config.go:96-117` (replace SRSConfig)

**Step 1: Replace SRSConfig in config.go**

Replace lines 96-117 with:

```go
// SRSConfig holds FSRS spaced-repetition parameters.
type SRSConfig struct {
	DefaultRetention   float64 `yaml:"default_retention"    env:"SRS_DEFAULT_RETENTION"     env-default:"0.9"`
	MaxIntervalDays    int     `yaml:"max_interval_days"    env:"SRS_MAX_INTERVAL"          env-default:"365"`
	EnableFuzz         bool    `yaml:"enable_fuzz"          env:"SRS_ENABLE_FUZZ"           env-default:"true"`
	LearningStepsRaw   string  `yaml:"learning_steps"       env:"SRS_LEARNING_STEPS"        env-default:"1m,10m"`
	RelearningStepsRaw string  `yaml:"relearning_steps"     env:"SRS_RELEARNING_STEPS"      env-default:"10m"`
	NewCardsPerDay     int     `yaml:"new_cards_per_day"    env:"SRS_NEW_CARDS_DAY"         env-default:"20"`
	ReviewsPerDay      int     `yaml:"reviews_per_day"      env:"SRS_REVIEWS_DAY"           env-default:"200"`
	UndoWindowMinutes  int     `yaml:"undo_window_minutes"  env:"SRS_UNDO_WINDOW_MINUTES"   env-default:"10"`

	// Parsed from raw strings during validation.
	LearningSteps   []time.Duration `yaml:"-" env:"-"`
	RelearningSteps []time.Duration `yaml:"-" env:"-"`
}
```

**Step 2: Verify build**

Run: `cd backend_v4 && go build ./internal/config/...`
Expected: PASS (config package is standalone)

**Step 3: Commit**

```bash
git add backend_v4/internal/config/config.go
git commit -m "feat(config): replace SM-2 config with FSRS parameters"
```

---

### Task 6: Card Repository — SQL and Repo Changes

**Files:**
- Modify: `backend_v4/internal/adapter/postgres/card/query/cards.sql` (all queries)
- Modify: `backend_v4/internal/adapter/postgres/card/repo.go` (raw SQL, Create, UpdateSRS, scanning)
- Run: `make generate` to regenerate sqlc code

**Step 1: Rewrite cards.sql with FSRS columns**

Replace entire `backend_v4/internal/adapter/postgres/card/query/cards.sql`:

```sql
-- ---------------------------------------------------------------------------
-- cards (FSRS)
-- ---------------------------------------------------------------------------

-- name: GetCardByID :one
SELECT id, user_id, entry_id, state, step,
       stability, difficulty, due, last_review,
       reps, lapses, scheduled_days, elapsed_days,
       created_at, updated_at
FROM cards
WHERE id = @id AND user_id = @user_id;

-- name: GetCardByEntryID :one
SELECT id, user_id, entry_id, state, step,
       stability, difficulty, due, last_review,
       reps, lapses, scheduled_days, elapsed_days,
       created_at, updated_at
FROM cards
WHERE entry_id = @entry_id AND user_id = @user_id;

-- name: CreateCard :one
INSERT INTO cards (id, user_id, entry_id, state, due, created_at, updated_at)
VALUES (@id, @user_id, @entry_id, @state, @due, @created_at, @updated_at)
RETURNING id, user_id, entry_id, state, step,
          stability, difficulty, due, last_review,
          reps, lapses, scheduled_days, elapsed_days,
          created_at, updated_at;

-- name: UpdateCardSRS :execrows
UPDATE cards
SET state = @state,
    step = @step,
    stability = @stability,
    difficulty = @difficulty,
    due = @due,
    last_review = @last_review,
    reps = @reps,
    lapses = @lapses,
    scheduled_days = @scheduled_days,
    elapsed_days = @elapsed_days,
    updated_at = now()
WHERE id = @id AND user_id = @user_id;

-- name: DeleteCard :execrows
DELETE FROM cards
WHERE id = @id AND user_id = @user_id;
```

**Step 2: Run sqlc generate**

Run: `cd backend_v4 && make generate`
Expected: sqlc regenerates `internal/adapter/postgres/card/sqlc/` with new types

**Step 3: Update raw SQL constants in repo.go**

Replace the raw SQL constants (lines 35-84) in `repo.go` with FSRS columns:

```go
const getDueCardsSQL = `
SELECT c.id, c.user_id, c.entry_id, c.state, c.step,
       c.stability, c.difficulty, c.due, c.last_review,
       c.reps, c.lapses, c.scheduled_days, c.elapsed_days,
       c.created_at, c.updated_at
FROM cards c
JOIN entries e ON c.entry_id = e.id
WHERE c.user_id = $1
  AND e.deleted_at IS NULL
  AND c.state IN ('LEARNING', 'RELEARNING', 'REVIEW')
  AND c.due <= $2
ORDER BY c.due ASC
LIMIT $3`

const getNewCardsSQL = `
SELECT c.id, c.user_id, c.entry_id, c.state, c.step,
       c.stability, c.difficulty, c.due, c.last_review,
       c.reps, c.lapses, c.scheduled_days, c.elapsed_days,
       c.created_at, c.updated_at
FROM cards c
JOIN entries e ON c.entry_id = e.id
WHERE c.user_id = $1 AND e.deleted_at IS NULL AND c.state = 'NEW'
ORDER BY c.created_at
LIMIT $2`

const countDueSQL = `
SELECT count(*) FROM cards c
JOIN entries e ON c.entry_id = e.id
WHERE c.user_id = $1 AND e.deleted_at IS NULL
  AND c.state IN ('LEARNING', 'RELEARNING', 'REVIEW')
  AND c.due <= $2`

const countNewSQL = `
SELECT count(*) FROM cards c
JOIN entries e ON c.entry_id = e.id
WHERE c.user_id = $1 AND e.deleted_at IS NULL AND c.state = 'NEW'`

const countByStatusSQL = `
SELECT c.state, count(*) as count
FROM cards c
JOIN entries e ON c.entry_id = e.id
WHERE c.user_id = $1 AND e.deleted_at IS NULL
GROUP BY c.state`

const getByEntryIDsSQL = `
SELECT c.id, c.user_id, c.entry_id, c.state, c.step,
       c.stability, c.difficulty, c.due, c.last_review,
       c.reps, c.lapses, c.scheduled_days, c.elapsed_days,
       c.created_at, c.updated_at
FROM cards c
WHERE c.entry_id = ANY($1::uuid[])`
```

**Step 4: Update Create method signature and implementation**

Change `Create` method signature from:
```go
func (r *Repo) Create(ctx context.Context, userID, entryID uuid.UUID, status domain.LearningStatus, easeFactor float64) (*domain.Card, error)
```
to:
```go
func (r *Repo) Create(ctx context.Context, userID, entryID uuid.UUID, state domain.CardState) (*domain.Card, error)
```

Update the body to use new sqlc params (state, due instead of status, easeFactor). Also update `CreateFromCard`.

**Step 5: Update UpdateSRS method**

Change `UpdateSRS` to pass all FSRS fields from `domain.SRSUpdateParams` (now has state, step, stability, difficulty, due, last_review, reps, lapses, scheduled_days, elapsed_days).

**Step 6: Update scan functions**

Replace `scanCardFromRows` (lines 401-432) and `toDomainCard` (lines 439-452) to scan/map new FSRS columns.

**Step 7: Update CountByStatus**

Replace the switch on `domain.LearningStatus` (lines 226-235) with `domain.CardState` values including `CardStateRelearning` instead of `CardStateMastered`.

**Step 8: Run build**

Run: `cd backend_v4 && go build ./internal/adapter/postgres/card/...`
Expected: PASS

**Step 9: Commit**

```bash
git add backend_v4/internal/adapter/postgres/card/
git commit -m "feat(repo): update card repository for FSRS columns"
```

---

### Task 7: Review Log Repository — CardSnapshot Serialization

**Files:**
- Modify: `backend_v4/internal/adapter/postgres/reviewlog/repo.go:382-442` (JSONB serialization)

**Step 1: Update cardSnapshotJSON struct**

Replace lines 382-442 with FSRS fields:

```go
type cardSnapshotJSON struct {
	State         string  `json:"state"`
	Step          int     `json:"step"`
	Stability     float64 `json:"stability"`
	Difficulty    float64 `json:"difficulty"`
	Due           string  `json:"due"`
	LastReview    *string `json:"last_review,omitempty"`
	Reps          int     `json:"reps"`
	Lapses        int     `json:"lapses"`
	ScheduledDays int     `json:"scheduled_days"`
	ElapsedDays   int     `json:"elapsed_days"`
}
```

Update `marshalPrevState` and `unmarshalPrevState` to serialize/deserialize the new `domain.CardSnapshot` fields.

**Step 2: Update CountNewToday query**

The current `CountNewToday` query filters on `prev_state->>'status' = 'NEW'`. Update to `prev_state->>'state' = 'NEW'`.

**Step 3: Run build**

Run: `cd backend_v4 && go build ./internal/adapter/postgres/reviewlog/...`
Expected: PASS

**Step 4: Commit**

```bash
git add backend_v4/internal/adapter/postgres/reviewlog/
git commit -m "feat(repo): update review log JSONB serialization for FSRS snapshot"
```

---

### Task 8: Study Service — Wire FSRS Scheduler

**Files:**
- Modify: `backend_v4/internal/service/study/service.go:16-28` (cardRepo interface)
- Modify: `backend_v4/internal/service/study/review_card.go` (use fsrs.ReviewCard)
- Modify: `backend_v4/internal/service/study/card_crud.go:44,217` (CreateCard initial params)
- Modify: `backend_v4/internal/service/study/undo_review.go:42-71` (restore FSRS fields)
- Modify: `backend_v4/internal/service/study/study_queue.go` (update status references)
- Modify: `backend_v4/internal/service/study/dashboard.go:167-174` (CardStats fields)
- Delete: `backend_v4/internal/service/study/srs.go` (old SM-2 algorithm)
- Delete: `backend_v4/internal/service/study/srs_test.go` (old SM-2 tests)

**Step 1: Update cardRepo interface**

Change the `Create` signature in `cardRepo` interface (line 19):
```go
Create(ctx context.Context, userID, entryID uuid.UUID, state domain.CardState) (*domain.Card, error)
```

Change all `domain.LearningStatus` references to `domain.CardState`.

**Step 2: Rewrite ReviewCard**

Replace `review_card.go` to use `fsrs.ReviewCard()`:

```go
func (s *Service) ReviewCard(ctx context.Context, input ReviewCardInput) (*domain.Card, error) {
	// ... auth, validation, load card, load settings (same as before) ...

	// Snapshot state before review
	snapshot := &domain.CardSnapshot{
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

	// Build FSRS params
	retention := s.srsConfig.DefaultRetention
	if settings.DesiredRetention > 0 {
		retention = settings.DesiredRetention
	}
	maxInterval := min(s.srsConfig.MaxIntervalDays, settings.MaxIntervalDays)

	params := fsrs.Parameters{
		W:                s.srsConfig.Weights,
		RequestRetention: retention,
		MaximumInterval:  maxInterval,
		EnableFuzz:       s.srsConfig.EnableFuzz,
		EnableShortTerm:  true,
		LearningSteps:    s.srsConfig.LearningSteps,
		RelearningSteps:  s.srsConfig.RelearningSteps,
	}

	// Convert domain card → fsrs card
	fsrsCard := toFSRSCard(card)
	rating := fsrs.Rating(gradeToRating(input.Grade))

	// Calculate new state
	result := fsrs.ReviewCard(params, fsrsCard, rating, now)

	// Transaction: update card + create log + audit
	// ... use result fields for SRSUpdateParams ...
}
```

**Step 3: Update CreateCard**

Change line 44 in `card_crud.go` from:
```go
card, createErr = s.cards.Create(txCtx, userID, input.EntryID, domain.LearningStatusNew, s.srsConfig.DefaultEaseFactor)
```
to:
```go
card, createErr = s.cards.Create(txCtx, userID, input.EntryID, domain.CardStateNew)
```

Similarly update `BatchCreateCards` (line 217).

**Step 4: Update UndoReview**

Replace all `PrevState` field references (lines 65-71) to use new `CardSnapshot` FSRS fields.

**Step 5: Update dashboard and study_queue**

Replace `domain.LearningStatusNew` → `domain.CardStateNew`, etc. throughout.
Update `CardStats` fields in dashboard.go (line 167-174): `CurrentStatus` → `CurrentState`, `IntervalDays` → removed, `EaseFactor` → removed, add `Stability`, `Difficulty`.

**Step 6: Delete old SM-2 files**

```bash
rm backend_v4/internal/service/study/srs.go
rm backend_v4/internal/service/study/srs_test.go
```

**Step 7: Add helper functions**

Add to `review_card.go`:
```go
func toFSRSCard(c *domain.Card) fsrs.Card {
	fc := fsrs.Card{
		State:         fsrs.State(cardStateToFSRS(c.State)),
		Step:          c.Step,
		Stability:     c.Stability,
		Difficulty:    c.Difficulty,
		Due:           c.Due,
		Reps:          c.Reps,
		Lapses:        c.Lapses,
		ScheduledDays: c.ScheduledDays,
		ElapsedDays:   c.ElapsedDays,
	}
	if c.LastReview != nil {
		fc.LastReview = *c.LastReview
	}
	return fc
}

func gradeToRating(g domain.ReviewGrade) fsrs.Rating {
	switch g {
	case domain.ReviewGradeAgain: return fsrs.Again
	case domain.ReviewGradeHard:  return fsrs.Hard
	case domain.ReviewGradeGood:  return fsrs.Good
	case domain.ReviewGradeEasy:  return fsrs.Easy
	default: return fsrs.Good
	}
}

func cardStateToFSRS(s domain.CardState) int {
	switch s {
	case domain.CardStateNew:        return 0
	case domain.CardStateLearning:   return 1
	case domain.CardStateReview:     return 2
	case domain.CardStateRelearning: return 3
	default: return 0
	}
}
```

**Step 8: Run build**

Run: `cd backend_v4 && go build ./internal/service/study/...`
Expected: PASS

**Step 9: Commit**

```bash
git add backend_v4/internal/service/study/
git commit -m "feat(study): wire FSRS-5 scheduler, remove SM-2 algorithm"
```

---

### Task 9: GraphQL Schema and Resolver Updates

**Files:**
- Modify: `backend_v4/internal/transport/graphql/schema/enums.graphql:1-6` (LearningStatus → CardState)
- Modify: `backend_v4/internal/transport/graphql/schema/study.graphql:5-14,55-60,62-67` (Card, CardStatusCounts, CardStats types)
- Modify: `backend_v4/internal/transport/graphql/schema/user.graphql` (add desiredRetention)
- Run: `make generate` to regenerate resolvers

**Step 1: Update enums.graphql**

Replace `LearningStatus` (lines 1-6):
```graphql
enum CardState {
  NEW
  LEARNING
  REVIEW
  RELEARNING
}
```

**Step 2: Update study.graphql**

Replace `Card` type (lines 5-14):
```graphql
type Card {
  id: UUID!
  entryId: UUID!
  state: CardState!
  stability: Float!
  difficulty: Float!
  due: DateTime!
  reps: Int!
  lapses: Int!
  scheduledDays: Int!
  createdAt: DateTime!
  updatedAt: DateTime!
}
```

Replace `CardStatusCounts` (lines 55-60):
```graphql
type CardStatusCounts {
  new: Int!
  learning: Int!
  review: Int!
  relearning: Int!
}
```

Replace `CardStats` (lines 62-67):
```graphql
type CardStats {
  totalReviews: Int!
  averageDurationMs: Int!
  accuracy: Float!
  stability: Float!
  difficulty: Float!
  gradeDistribution: GradeCounts!
}
```

**Step 3: Update user.graphql**

Add `desiredRetention` to `UserSettings` type.

**Step 4: Regenerate**

Run: `cd backend_v4 && make generate`

Fix any resolver compilation errors resulting from type changes.

**Step 5: Commit**

```bash
git add backend_v4/internal/transport/graphql/
git commit -m "feat(graphql): update schema for FSRS — CardState, stability, difficulty"
```

---

### Task 10: App Wiring

**Files:**
- Modify: `backend_v4/internal/app/app.go:151-166` (srsConfig initialization)

**Step 1: Update srsConfig in app.go**

Replace lines 151-166:
```go
srsConfig := domain.SRSConfig{
	Weights:           fsrs.DefaultWeights,
	DefaultRetention:  cfg.SRS.DefaultRetention,
	MaxIntervalDays:   cfg.SRS.MaxIntervalDays,
	EnableFuzz:        cfg.SRS.EnableFuzz,
	LearningSteps:     cfg.SRS.LearningSteps,
	RelearningSteps:   cfg.SRS.RelearningSteps,
	NewCardsPerDay:    cfg.SRS.NewCardsPerDay,
	ReviewsPerDay:     cfg.SRS.ReviewsPerDay,
	UndoWindowMinutes: cfg.SRS.UndoWindowMinutes,
}
```

Add import for the fsrs package.

**Step 2: Run build**

Run: `cd backend_v4 && make build`
Expected: PASS

**Step 3: Commit**

```bash
git add backend_v4/internal/app/app.go
git commit -m "feat(app): wire FSRS config into service layer"
```

---

### Task 11: User Settings — DB and Service Updates

**Files:**
- Modify: `backend_v4/internal/adapter/postgres/user/` — settings queries to include `desired_retention`
- Modify: user service if settings update mutation exists

**Step 1: Update settings SQL queries**

Add `desired_retention` to all SELECT, INSERT, UPDATE queries in user/settings SQL files.

**Step 2: Run sqlc generate**

Run: `cd backend_v4 && make generate`

**Step 3: Update repo scan functions**

Add `DesiredRetention` to settings scan/mapping in user repo.

**Step 4: Run build**

Run: `cd backend_v4 && go build ./...`
Expected: PASS — full project compiles

**Step 5: Commit**

```bash
git add backend_v4/internal/adapter/postgres/user/
git commit -m "feat(repo): add desired_retention to user settings queries"
```

---

### Task 12: Integration — Full Build + Lint

**Files:** None (verification only)

**Step 1: Full build**

Run: `cd backend_v4 && make build`
Expected: Binary builds successfully

**Step 2: Lint**

Run: `cd backend_v4 && make lint`
Expected: No new lint errors

**Step 3: Run unit tests**

Run: `cd backend_v4 && make test`
Expected: PASS (some tests may need updates — fix as needed)

**Step 4: Commit any fixes**

```bash
git add -A && git commit -m "fix: resolve compilation and lint issues from FSRS migration"
```

---

### Task 13: E2E Tests Update

**Files:**
- Modify: `backend_v4/tests/e2e/` — update study-related E2E tests

**Step 1: Update E2E test assertions**

Change all assertions from:
- `status` → `state`
- `easeFactor` → `stability` / `difficulty`
- `nextReviewAt` → `due`
- `intervalDays` → `scheduledDays`
- `MASTERED` → remove or replace with `REVIEW`

Verify `reps` and `lapses` are correctly tracked.

**Step 2: Run E2E tests**

Run: `cd backend_v4 && make test-e2e`
Expected: PASS

**Step 3: Commit**

```bash
git add backend_v4/tests/e2e/
git commit -m "test(e2e): update study E2E tests for FSRS migration"
```
