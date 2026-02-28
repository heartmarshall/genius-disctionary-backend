package fsrs

import (
	"encoding/binary"
	"hash/fnv"
	"math"
	"math/rand"
	"time"
)

// FuzzRange defines a tier for the 3-tier fuzz system.
type FuzzRange struct {
	Start  float64
	End    float64
	Factor float64
}

// fuzzRanges matches the go-fsrs reference 3-tier fuzz.
var fuzzRanges = []FuzzRange{
	{Start: 2.5, End: 7.0, Factor: 0.15},
	{Start: 7.0, End: 20.0, Factor: 0.10},
	{Start: 20.0, End: math.MaxFloat64, Factor: 0.05},
}

// getFuzzRange returns the min and max interval bounds after fuzz for a given interval.
func getFuzzRange(interval, elapsedDays, maximumInterval float64) (minIvl, maxIvl int) {
	if interval < 2.5 {
		return int(math.Round(interval)), int(math.Round(interval))
	}

	delta := 1.0
	for _, r := range fuzzRanges {
		delta += r.Factor * math.Max(math.Min(interval, r.End)-r.Start, 0.0)
	}

	minIvl = int(math.Round(interval - delta))
	maxIvl = int(math.Round(interval + delta))

	// Enforce: minIvl >= 2 (never fuzz below 2)
	if minIvl < 2 {
		minIvl = 2
	}

	// If interval > elapsedDays, ensure minIvl > elapsedDays
	if interval > elapsedDays {
		ed := int(elapsedDays)
		if minIvl <= ed {
			minIvl = ed + 1
		}
	}

	// Clamp maxIvl to maximumInterval
	maxInt := int(maximumInterval)
	if maxIvl > maxInt {
		maxIvl = maxInt
	}

	// Safety: ensure minIvl <= maxIvl
	if minIvl > maxIvl {
		minIvl = maxIvl
	}

	return minIvl, maxIvl
}

// applyFuzz applies 3-tier fuzz to an interval using a deterministic seed.
// Returns the original interval if < 2.5 or if fuzz is disabled.
func applyFuzz(interval, elapsedDays, maximumInterval float64, seed int64) float64 {
	if interval < 2.5 {
		return interval
	}

	minIvl, maxIvl := getFuzzRange(interval, elapsedDays, maximumInterval)

	if minIvl == maxIvl {
		return float64(minIvl)
	}

	//nolint:gosec // deterministic fuzz, not cryptographic
	rng := rand.New(rand.NewSource(seed))
	fuzzed := minIvl + rng.Intn(maxIvl-minIvl+1)

	return float64(fuzzed)
}

// FuzzSeed generates a deterministic seed from card state using FNV-1a hash.
func FuzzSeed(now time.Time, reps int, difficulty, stability float64) int64 {
	h := fnv.New64a()
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(now.Unix()))
	h.Write(b)
	binary.LittleEndian.PutUint64(b, uint64(reps))
	h.Write(b)
	binary.LittleEndian.PutUint64(b, math.Float64bits(difficulty))
	h.Write(b)
	binary.LittleEndian.PutUint64(b, math.Float64bits(stability))
	h.Write(b)
	return int64(h.Sum64())
}
