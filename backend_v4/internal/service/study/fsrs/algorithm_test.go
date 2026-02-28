package fsrs

import (
	"math"
	"testing"
)

const epsilon = 1e-6

func TestRetrievability(t *testing.T) {
	tests := []struct {
		name        string
		elapsedDays int
		stability   float64
		want        float64
	}{
		{"zero elapsed", 0, 10.0, 1.0},
		{"one day, S=9", 1, 9.0, 0.98780}, // (1 + 1/81)^-1
		{"zero stability", 5, 0, 0},
		{"half life", 90, 10.0, 0.5},       // t=9*S → R=0.5
		{"long elapsed", 100, 10.0, 0.4737}, // (1 + 100/90)^-1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Retrievability(tt.elapsedDays, tt.stability)
			if math.Abs(got-tt.want) > 0.001 {
				t.Errorf("Retrievability(%d, %f) = %f, want %f", tt.elapsedDays, tt.stability, got, tt.want)
			}
		})
	}
}

func TestNextInterval(t *testing.T) {
	tests := []struct {
		name      string
		stability float64
		retention float64
		want      int
	}{
		{"basic", 10.0, 0.9, 1},   // 9*10*(1/0.9-1) = 10, round = 10... let me recalc
		{"high retention", 10, 0.9, 10},
		{"low retention", 10, 0.5, 90},
		{"floor at 1", 0.01, 0.9, 1},
		{"invalid retention 0", 10, 0, 1},
		{"invalid retention 1", 10, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NextInterval(tt.stability, tt.retention)
			if tt.name == "basic" || tt.name == "high retention" {
				// 9 * 10 * (1/0.9 - 1) = 9*10*0.111 = 10
				if got != 10 {
					t.Errorf("NextInterval(%f, %f) = %d, want 10", tt.stability, tt.retention, got)
				}
				return
			}
			if tt.name == "low retention" {
				// 9 * 10 * (1/0.5 - 1) = 9*10*1 = 90
				if got != 90 {
					t.Errorf("NextInterval(%f, %f) = %d, want 90", tt.stability, tt.retention, got)
				}
				return
			}
			if got < 1 {
				t.Errorf("NextInterval(%f, %f) = %d, want >= 1", tt.stability, tt.retention, got)
			}
		})
	}
}

func TestInitialStability(t *testing.T) {
	w := DefaultWeights

	tests := []struct {
		rating Rating
		want   float64
	}{
		{Again, w[0]},
		{Hard, w[1]},
		{Good, w[2]},
		{Easy, w[3]},
	}

	for _, tt := range tests {
		got := InitialStability(w, tt.rating)
		if math.Abs(got-tt.want) > epsilon {
			t.Errorf("InitialStability(rating=%d) = %f, want %f", tt.rating, got, tt.want)
		}
	}
}

func TestInitialDifficulty(t *testing.T) {
	w := DefaultWeights

	// D0(G) = w4 - exp(w5 * (G - 1)) + 1
	tests := []struct {
		rating Rating
	}{
		{Again},
		{Hard},
		{Good},
		{Easy},
	}

	var prev float64
	for _, tt := range tests {
		got := InitialDifficulty(w, tt.rating)
		if got < 1 || got > 10 {
			t.Errorf("InitialDifficulty(rating=%d) = %f, out of [1,10]", tt.rating, got)
		}
		if tt.rating > Again && got >= prev {
			t.Errorf("InitialDifficulty should decrease as rating increases: rating=%d, got=%f, prev=%f", tt.rating, got, prev)
		}
		prev = got
	}
}

func TestNextDifficulty(t *testing.T) {
	w := DefaultWeights

	// Easy should decrease difficulty
	d := 5.0
	dAfterEasy := NextDifficulty(w, d, Easy)
	if dAfterEasy >= d {
		t.Errorf("NextDifficulty with Easy should decrease: got %f from %f", dAfterEasy, d)
	}

	// Again should increase difficulty
	dAfterAgain := NextDifficulty(w, d, Again)
	if dAfterAgain <= d {
		t.Errorf("NextDifficulty with Again should increase: got %f from %f", dAfterAgain, d)
	}

	// Result should be clamped
	dLow := NextDifficulty(w, 1.0, Easy)
	if dLow < 1 {
		t.Errorf("NextDifficulty should be >= 1, got %f", dLow)
	}

	dHigh := NextDifficulty(w, 10.0, Again)
	if dHigh > 10 {
		t.Errorf("NextDifficulty should be <= 10, got %f", dHigh)
	}
}

func TestStabilityAfterRecall(t *testing.T) {
	w := DefaultWeights

	s := 10.0
	d := 5.0
	r := 0.9

	// Stability should increase after successful recall
	for _, rating := range []Rating{Hard, Good, Easy} {
		got := StabilityAfterRecall(w, s, d, r, rating)
		if got < s {
			t.Errorf("StabilityAfterRecall(rating=%d) = %f, should be >= %f", rating, got, s)
		}
		if got < MinStability {
			t.Errorf("StabilityAfterRecall(rating=%d) = %f, below MinStability", rating, got)
		}
	}

	// Easy > Good > Hard
	hardS := StabilityAfterRecall(w, s, d, r, Hard)
	goodS := StabilityAfterRecall(w, s, d, r, Good)
	easyS := StabilityAfterRecall(w, s, d, r, Easy)

	if !(easyS > goodS && goodS > hardS) {
		t.Errorf("Expected Easy > Good > Hard stability: easy=%f, good=%f, hard=%f", easyS, goodS, hardS)
	}
}

func TestStabilityAfterForgetting(t *testing.T) {
	w := DefaultWeights

	s := 10.0
	d := 5.0
	r := 0.3 // low retrievability (forgotten)

	got := StabilityAfterForgetting(w, s, d, r)

	if got >= s {
		t.Errorf("StabilityAfterForgetting should be < original S: got %f, original %f", got, s)
	}
	if got < MinStability {
		t.Errorf("StabilityAfterForgetting = %f, below MinStability", got)
	}
}

func TestNextSMin(t *testing.T) {
	w := DefaultWeights

	s := 10.0
	sMin := NextSMin(w, s)

	// nextSMin = S / exp(w17 * w18) should be less than S
	expected := s / math.Exp(w[17]*w[18])
	if math.Abs(sMin-expected) > epsilon {
		t.Errorf("NextSMin(%f) = %f, want %f", s, sMin, expected)
	}

	if sMin >= s {
		t.Errorf("NextSMin should be < original S: got %f", sMin)
	}
}

func TestStabilityAfterForgettingCapped(t *testing.T) {
	w := DefaultWeights

	s := 10.0
	d := 5.0
	r := 0.3

	capped := StabilityAfterForgettingCapped(w, s, d, r)
	uncapped := StabilityAfterForgetting(w, s, d, r)
	sMin := NextSMin(w, s)

	// Capped should be <= min(sMin, uncapped)
	expected := math.Min(sMin, uncapped)
	if capped > expected+epsilon {
		t.Errorf("StabilityAfterForgettingCapped = %f, should be <= min(sMin=%f, sf=%f) = %f",
			capped, sMin, uncapped, expected)
	}

	// Should be at least MinStability
	if capped < MinStability {
		t.Errorf("StabilityAfterForgettingCapped = %f, below MinStability", capped)
	}
}

func TestShortTermStability(t *testing.T) {
	w := DefaultWeights

	s := 5.0

	// Good should increase stability
	goodS := ShortTermStability(w, s, Good)
	// Again should decrease stability
	againS := ShortTermStability(w, s, Again)

	if againS >= goodS {
		t.Errorf("Again stability (%f) should be < Good stability (%f)", againS, goodS)
	}

	// All should be >= MinStability
	for _, rating := range []Rating{Again, Hard, Good, Easy} {
		got := ShortTermStability(w, s, rating)
		if got < MinStability {
			t.Errorf("ShortTermStability(rating=%d) = %f, below MinStability", rating, got)
		}
	}
}

func TestMinStability(t *testing.T) {
	if MinStability != 0.1 {
		t.Errorf("MinStability = %f, want 0.1", MinStability)
	}
}
