// Package fsrs implements the FSRS-5 spaced repetition algorithm.
// Core formulas match the go-fsrs reference exactly.
package fsrs

import (
	"fmt"
	"math"
)

// MinStability is the floor for stability values (reference: max(w[r-1], 0.1)).
const MinStability = 0.1

// DefaultWeights are the default FSRS-5 model weights (w[0]..w[18]).
var DefaultWeights = [19]float64{
	0.4072,  // w0  - initial stability for Again
	1.1829,  // w1  - initial stability for Hard
	3.1262,  // w2  - initial stability for Good
	15.4722, // w3  - initial stability for Easy
	7.2102,  // w4  - initial difficulty mean reversion
	0.5316,  // w5  - initial difficulty slope
	1.0651,  // w6  - difficulty update: D - w6*(G-3)
	0.0046,  // w7  - difficulty mean reversion weight
	1.5418,  // w8  - recall stability: exp(w8)
	0.1594,  // w9  - recall stability: S^(-w9)
	1.01,    // w10 - recall stability: exp(w10*(1-R)) - 1
	2.1791,  // w11 - forget stability: multiplier
	0.0292,  // w12 - forget stability: D^(-w12)
	0.2788,  // w13 - forget stability: (S+1)^w13 - 1
	0.2229,  // w14 - forget stability: exp(w14*(1-R))
	0.2604,  // w15 - recall stability: hard penalty
	3.3928,  // w16 - recall stability: easy bonus
	0.2223,  // w17 - short-term stability / nextSMin
	0.6744,  // w18 - short-term stability / nextSMin
}

// Rating represents the user's recall quality.
type Rating int

const (
	Again Rating = 1
	Hard  Rating = 2
	Good  Rating = 3
	Easy  Rating = 4
)

// Retrievability calculates the probability of recall.
//
//	R(t, S) = (1 + t/(9*S))^(-1)
func Retrievability(elapsedDays int, stability float64) float64 {
	if stability <= 0 {
		return 0
	}
	return math.Pow(1+float64(elapsedDays)/(9*stability), -1)
}

// NextInterval converts stability and desired retention to an interval in days.
//
//	I(S, r) = round(9 * S * (1/r - 1))
func NextInterval(stability, requestRetention float64) int {
	if requestRetention <= 0 || requestRetention >= 1 {
		return 1
	}
	interval := 9 * stability * (1/requestRetention - 1)
	return max(1, int(math.Round(interval)))
}

// InitialStability returns the starting stability for a given first rating.
//
//	S0(G) = w[G-1]  (clamped to MinStability)
func InitialStability(w [19]float64, rating Rating) float64 {
	idx := int(rating) - 1
	if idx < 0 || idx > 3 {
		idx = 2 // default to Good
	}
	return math.Max(MinStability, w[idx])
}

// InitialDifficulty returns the starting difficulty for a given first rating.
//
//	D0(G) = w4 - exp(w5 * (G - 1)) + 1
//	clamped to [1, 10]
func InitialDifficulty(w [19]float64, rating Rating) float64 {
	d := w[4] - math.Exp(w[5]*float64(rating-1)) + 1
	return clampDifficulty(d)
}

// NextDifficulty calculates the new difficulty after a review.
//
//	D'(D, G) = w7 * D0(4) + (1 - w7) * (D - w6 * (G - 3))
//	clamped to [1, 10]
//
// Uses mean reversion toward D0(4) to prevent difficulty from drifting.
func NextDifficulty(w [19]float64, d float64, rating Rating) float64 {
	d0Easy := InitialDifficulty(w, Easy)
	newD := w[7]*d0Easy + (1-w[7])*(d-w[6]*(float64(rating)-3))
	return clampDifficulty(newD)
}

// StabilityAfterRecall calculates post-recall stability (when rating >= Hard).
//
//	S'r(S, D, R, G) = S * (e^(w8) * (11-D) * S^(-w9) * (e^(w10*(1-R)) - 1) * hardPenalty * easyBonus + 1)
//
// hardPenalty = w15 if G==Hard, else 1
// easyBonus  = w16 if G==Easy, else 1
func StabilityAfterRecall(w [19]float64, s, d, r float64, rating Rating) float64 {
	hardPenalty := 1.0
	if rating == Hard {
		hardPenalty = w[15]
	}
	easyBonus := 1.0
	if rating == Easy {
		easyBonus = w[16]
	}

	newS := s * (math.Exp(w[8]) *
		(11-d) *
		math.Pow(s, -w[9]) *
		(math.Exp(w[10]*(1-r)) - 1) *
		hardPenalty *
		easyBonus +
		1)

	return math.Max(MinStability, newS)
}

// StabilityAfterForgetting calculates post-lapse stability (when rating == Again).
//
//	S'f(S, D, R) = w11 * D^(-w12) * ((S+1)^w13 - 1) * e^(w14*(1-R))
func StabilityAfterForgetting(w [19]float64, s, d, r float64) float64 {
	newS := w[11] *
		math.Pow(d, -w[12]) *
		(math.Pow(s+1, w[13]) - 1) *
		math.Exp(w[14]*(1-r))
	return math.Max(MinStability, newS)
}

// NextSMin returns the maximum allowed post-lapse stability.
// This caps the forget stability to prevent it from exceeding the pre-lapse value.
//
//	nextSMin = S / exp(w17 * w18)
func NextSMin(w [19]float64, stability float64) float64 {
	return stability / math.Exp(w[17]*w[18])
}

// StabilityAfterForgettingCapped applies the nextSMin cap to forget stability.
//
//	newS = min(S/exp(w17*w18), S'f)
func StabilityAfterForgettingCapped(w [19]float64, s, d, r float64) float64 {
	sMin := NextSMin(w, s)
	sf := StabilityAfterForgetting(w, s, d, r)
	return math.Max(MinStability, math.Min(sMin, sf))
}

// ShortTermStability calculates stability for short-term reviews (learning/relearning).
//
//	S'st(S, G) = S * e^(w17 * (G - 3 + w18))
func ShortTermStability(w [19]float64, s float64, rating Rating) float64 {
	newS := s * math.Exp(w[17]*(float64(rating)-3+w[18]))
	return math.Max(MinStability, newS)
}

// ValidateWeights checks that all 19 FSRS weights are finite and non-NaN.
func ValidateWeights(w [19]float64) error {
	for i, v := range w {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("weight w[%d] is invalid: %v", i, v)
		}
	}
	if w[0] <= 0 || w[1] <= 0 || w[2] <= 0 || w[3] <= 0 {
		return fmt.Errorf("initial stability weights w[0]-w[3] must be positive")
	}
	return nil
}

// clampDifficulty constrains difficulty to [1, 10].
func clampDifficulty(d float64) float64 {
	return math.Max(1, math.Min(10, d))
}
