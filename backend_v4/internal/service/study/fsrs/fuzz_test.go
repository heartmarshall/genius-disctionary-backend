package fsrs

import (
	"testing"
	"time"
)

func TestGetFuzzRange_NoFuzzBelow2_5(t *testing.T) {
	minI, maxI := getFuzzRange(2.0, 0, 365)
	if minI != 2 || maxI != 2 {
		t.Errorf("getFuzzRange(2.0) = (%d, %d), want (2, 2)", minI, maxI)
	}
}

func TestGetFuzzRange_SmallInterval(t *testing.T) {
	// interval=3, should have small delta (~1.075)
	minI, maxI := getFuzzRange(3.0, 0, 365)
	if minI < 2 || maxI > 5 {
		t.Errorf("getFuzzRange(3.0) = (%d, %d), unexpected range", minI, maxI)
	}
	if minI > maxI {
		t.Errorf("minIvl > maxIvl: %d > %d", minI, maxI)
	}
}

func TestGetFuzzRange_MediumInterval(t *testing.T) {
	// interval=10, tier 1 full (0.15*(7-2.5)=0.675) + tier 2 partial (0.10*(10-7)=0.3)
	minI, maxI := getFuzzRange(10.0, 0, 365)
	if minI < 7 || maxI > 13 {
		t.Errorf("getFuzzRange(10.0) = (%d, %d), unexpected range", minI, maxI)
	}
}

func TestGetFuzzRange_LargeInterval(t *testing.T) {
	// interval=30, all 3 tiers active
	minI, maxI := getFuzzRange(30.0, 0, 365)
	if minI < 26 || maxI > 34 {
		t.Errorf("getFuzzRange(30.0) = (%d, %d), unexpected range", minI, maxI)
	}
}

func TestGetFuzzRange_ClampsToMaxInterval(t *testing.T) {
	minI, maxI := getFuzzRange(360.0, 0, 365)
	if maxI > 365 {
		t.Errorf("maxIvl should be <= maximumInterval: got %d", maxI)
	}
	_ = minI
}

func TestGetFuzzRange_MinIvlGtElapsed(t *testing.T) {
	// If interval > elapsedDays, minIvl should be > elapsedDays
	minI, _ := getFuzzRange(5.0, 3.0, 365)
	if minI <= 3 {
		t.Errorf("minIvl should be > elapsedDays(3): got %d", minI)
	}
}

func TestApplyFuzz_NoFuzzSmallInterval(t *testing.T) {
	got := applyFuzz(2.0, 0, 365, 12345)
	if got != 2.0 {
		t.Errorf("applyFuzz(2.0) = %f, want 2.0", got)
	}
}

func TestApplyFuzz_Deterministic(t *testing.T) {
	seed := int64(42)
	r1 := applyFuzz(15.0, 5, 365, seed)
	r2 := applyFuzz(15.0, 5, 365, seed)
	if r1 != r2 {
		t.Errorf("applyFuzz should be deterministic: got %f and %f", r1, r2)
	}
}

func TestApplyFuzz_WithinRange(t *testing.T) {
	interval := 20.0
	seed := int64(99)
	minI, maxI := getFuzzRange(interval, 5, 365)

	for s := int64(0); s < 100; s++ {
		got := applyFuzz(interval, 5, 365, seed+s)
		if int(got) < minI || int(got) > maxI {
			t.Errorf("applyFuzz(seed=%d) = %f, outside [%d, %d]", seed+s, got, minI, maxI)
		}
	}
}

func TestFuzzSeed(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	s1 := FuzzSeed(now, 5, 5.0, 10.0)
	s2 := FuzzSeed(now, 5, 5.0, 10.0)
	if s1 != s2 {
		t.Errorf("FuzzSeed should be deterministic")
	}

	// Different inputs should (usually) produce different seeds
	s3 := FuzzSeed(now, 6, 5.0, 10.0)
	if s1 == s3 {
		t.Logf("Warning: different reps produced same seed (unlikely but possible)")
	}
}
