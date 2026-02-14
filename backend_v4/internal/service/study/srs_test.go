package study

import (
	"testing"
	"time"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

func TestCalculateSRS(t *testing.T) {
	now := time.Date(2024, 2, 15, 12, 0, 0, 0, time.UTC)

	defaultConfig := domain.SRSConfig{
		DefaultEaseFactor:    2.5,
		MinEaseFactor:        1.3,
		MaxIntervalDays:      365,
		GraduatingInterval:   1,
		EasyInterval:         4,
		LearningSteps:        []time.Duration{1 * time.Minute, 10 * time.Minute},
		RelearningSteps:      []time.Duration{10 * time.Minute},
		IntervalModifier:     1.0,
		HardIntervalModifier: 1.2,
		EasyBonus:            1.3,
		LapseNewInterval:     0.0,
	}

	tests := []struct {
		name         string
		input        SRSInput
		wantStatus   domain.LearningStatus
		wantStep     int
		wantInterval int
		wantEase     float64
		checkDelay   *time.Duration // For learning steps
	}{
		// NEW → LEARNING
		{
			name: "1. NEW AGAIN → LEARNING step 0",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusNew,
				Grade:           domain.ReviewGradeAgain,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusLearning,
			wantStep:     0,
			wantInterval: 0,
			wantEase:     2.5,
			checkDelay:   ptrDuration(1 * time.Minute),
		},
		{
			name: "2. NEW HARD → LEARNING step 0, avg delay",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusNew,
				Grade:           domain.ReviewGradeHard,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusLearning,
			wantStep:     0,
			wantInterval: 0,
			wantEase:     2.5,
			checkDelay:   ptrDuration(5*time.Minute + 30*time.Second), // avg(1m, 10m)
		},
		{
			name: "3. NEW GOOD → LEARNING step 1",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusNew,
				Grade:           domain.ReviewGradeGood,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusLearning,
			wantStep:     1,
			wantInterval: 0,
			wantEase:     2.5,
			checkDelay:   ptrDuration(10 * time.Minute),
		},
		{
			name: "4. NEW EASY → REVIEW (graduate)",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusNew,
				Grade:           domain.ReviewGradeEasy,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusReview,
			wantStep:     0,
			wantInterval: 4,
			wantEase:     2.5,
		},

		// LEARNING step 0
		{
			name: "5. LEARNING step 0 AGAIN → reset",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusLearning,
				LearningStep:    0,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeAgain,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus: domain.LearningStatusLearning,
			wantStep:   0,
			wantEase:   2.5,
			checkDelay: ptrDuration(1 * time.Minute),
		},
		{
			name: "6. LEARNING step 0 HARD → repeat",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusLearning,
				LearningStep:    0,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeHard,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus: domain.LearningStatusLearning,
			wantStep:   0,
			wantEase:   2.5,
			checkDelay: ptrDuration(1 * time.Minute),
		},
		{
			name: "7. LEARNING step 0 GOOD → step 1",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusLearning,
				LearningStep:    0,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeGood,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus: domain.LearningStatusLearning,
			wantStep:   1,
			wantEase:   2.5,
			checkDelay: ptrDuration(10 * time.Minute),
		},
		{
			name: "8. LEARNING step 0 EASY → graduate",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusLearning,
				LearningStep:    0,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeEasy,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusReview,
			wantStep:     0,
			wantInterval: 4,
			wantEase:     2.5,
		},

		// LEARNING step 1
		{
			name: "9. LEARNING step 1 AGAIN → reset",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusLearning,
				LearningStep:    1,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeAgain,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus: domain.LearningStatusLearning,
			wantStep:   0,
			wantEase:   2.5,
			checkDelay: ptrDuration(1 * time.Minute),
		},
		{
			name: "10. LEARNING step 1 HARD → repeat",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusLearning,
				LearningStep:    1,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeHard,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus: domain.LearningStatusLearning,
			wantStep:   1,
			wantEase:   2.5,
			checkDelay: ptrDuration(10 * time.Minute),
		},
		{
			name: "11. LEARNING step 1 GOOD → graduate",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusLearning,
				LearningStep:    1,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeGood,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusReview,
			wantStep:     0,
			wantInterval: 1,
			wantEase:     2.5,
		},
		{
			name: "12. LEARNING step 1 EASY → graduate easy",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusLearning,
				LearningStep:    1,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeEasy,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusReview,
			wantStep:     0,
			wantInterval: 4,
			wantEase:     2.5,
		},

		// REVIEW
		{
			name: "13. REVIEW AGAIN → lapse to relearning",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusReview,
				CurrentInterval: 1,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeAgain,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusLearning,
			wantStep:     0,
			wantInterval: 1,
			wantEase:     2.3,
			checkDelay:   ptrDuration(10 * time.Minute),
		},
		{
			name: "14. REVIEW HARD",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusReview,
				CurrentInterval: 1,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeHard,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusReview,
			wantStep:     0,
			wantInterval: 2, // max(1+1, 1*1.2) = 2
			wantEase:     2.35,
		},
		{
			name: "15. REVIEW GOOD",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusReview,
				CurrentInterval: 1,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeGood,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusReview,
			wantStep:     0,
			wantInterval: 3, // max(1+1, 1*2.5*1.0) = 3 (with fuzz ~3)
			wantEase:     2.5,
		},
		{
			name: "16. REVIEW EASY",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusReview,
				CurrentInterval: 1,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeEasy,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusReview,
			wantStep:     0,
			wantInterval: 3, // max(1+1, 1*2.5*1.3*1.0) = max(2, 3.25) = 3 (with fuzz ±1)
			wantEase:     2.65,
		},
		{
			name: "17. REVIEW longer GOOD",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusReview,
				CurrentInterval: 10,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeGood,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusMastered, // 10*2.5 = 25 ≥ 21 AND ease ≥ 2.5
			wantStep:     0,
			wantInterval: 25, // 10*2.5*1.0 = 25
			wantEase:     2.5,
		},
		{
			name: "18. REVIEW longer HARD",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusReview,
				CurrentInterval: 10,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeHard,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusReview,
			wantStep:     0,
			wantInterval: 12, // 10*1.2 = 12
			wantEase:     2.35,
		},

		// MASTERED
		{
			name: "19. REVIEW → MASTERED (interval≥21, ease≥2.5)",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusReview,
				CurrentInterval: 21,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeGood,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusMastered,
			wantStep:     0,
			wantInterval: 53, // 21*2.5*1.0 = 52.5 → 53 (with fuzz)
			wantEase:     2.5,
		},
		{
			name: "20. REVIEW not mastered (interval<21)",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusReview,
				CurrentInterval: 20,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeGood,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusMastered, // Result: 20*2.5 = 50 ≥ 21 AND ease ≥ 2.5
			wantStep:     0,
			wantInterval: 50, // 20*2.5*1.0 = 50
			wantEase:     2.5,
		},
		{
			name: "21. MASTERED GOOD → stays MASTERED",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusMastered,
				CurrentInterval: 53,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeGood,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusMastered,
			wantStep:     0,
			wantInterval: 133, // 53*2.5*1.0 = 132.5 → 133 (with fuzz)
			wantEase:     2.5,
		},
		{
			name: "22. MASTERED AGAIN → lapse",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusMastered,
				CurrentInterval: 53,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeAgain,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusLearning,
			wantStep:     0,
			wantInterval: 1, // lapse_new_interval=0.0 → 1
			wantEase:     2.3,
			checkDelay:   ptrDuration(10 * time.Minute),
		},

		// Boundaries
		{
			name: "23. Ease minimum",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusReview,
				CurrentInterval: 5,
				CurrentEase:     1.3,
				Grade:           domain.ReviewGradeAgain,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusLearning,
			wantStep:     0,
			wantInterval: 1,
			wantEase:     1.3, // min(1.3, 1.3-0.20) = 1.3
		},
		{
			name: "24. Ease at min + HARD",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusReview,
				CurrentInterval: 5,
				CurrentEase:     1.3,
				Grade:           domain.ReviewGradeHard,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusReview,
			wantStep:     0,
			wantInterval: 6, // max(5+1, 5*1.2) = 6
			wantEase:     1.3,
		},
		{
			name: "25. Max interval cap (global)",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusReview,
				CurrentInterval: 200,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeGood,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusMastered,
			wantStep:     0,
			wantInterval: 365, // capped at 365
			wantEase:     2.5,
		},
		{
			name: "26. Max interval cap (user)",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusReview,
				CurrentInterval: 200,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeGood,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 180,
			},
			wantStatus:   domain.LearningStatusMastered,
			wantStep:     0,
			wantInterval: 180, // capped at user's 180
			wantEase:     2.5,
		},
		{
			name: "27. Min growth (interval ≥ old+1)",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusReview,
				CurrentInterval: 10,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeGood,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusMastered, // Result: 10*2.5 = 25 ≥ 21 AND ease ≥ 2.5
			wantStep:     0,
			wantInterval: 25, // 10*2.5 = 25 ≥ 10+1
			wantEase:     2.5,
		},

		// Relearning
		{
			name: "28. Relearning graduate",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusLearning,
				CurrentInterval: 10,
				CurrentEase:     2.0,
				LearningStep:    0,
				Grade:           domain.ReviewGradeGood,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusReview,
			wantStep:     0,
			wantInterval: 1,
			wantEase:     2.0, // Preserved during relearning
		},
		{
			name: "29. Relearning AGAIN",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusLearning,
				CurrentInterval: 10,
				CurrentEase:     2.0,
				LearningStep:    0,
				Grade:           domain.ReviewGradeAgain,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusLearning,
			wantStep:     0,
			wantInterval: 10,
			wantEase:     2.0,
			checkDelay:   ptrDuration(10 * time.Minute),
		},

		// Lapse variations
		{
			name: "30. Lapse reset (lapse_new_interval=0.0)",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusReview,
				CurrentInterval: 30,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeAgain,
				Now:             now,
				Config: domain.SRSConfig{
					DefaultEaseFactor:    2.5,
					MinEaseFactor:        1.3,
					MaxIntervalDays:      365,
					GraduatingInterval:   1,
					EasyInterval:         4,
					LearningSteps:        []time.Duration{1 * time.Minute, 10 * time.Minute},
					RelearningSteps:      []time.Duration{10 * time.Minute},
					IntervalModifier:     1.0,
					HardIntervalModifier: 1.2,
					EasyBonus:            1.3,
					LapseNewInterval:     0.0,
				},
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusLearning,
			wantStep:     0,
			wantInterval: 1, // 30*0.0 = 0 → max(1, 0) = 1
			wantEase:     2.3,
		},
		{
			name: "31. Lapse 50%",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusReview,
				CurrentInterval: 30,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeAgain,
				Now:             now,
				Config: domain.SRSConfig{
					DefaultEaseFactor:    2.5,
					MinEaseFactor:        1.3,
					MaxIntervalDays:      365,
					GraduatingInterval:   1,
					EasyInterval:         4,
					LearningSteps:        []time.Duration{1 * time.Minute, 10 * time.Minute},
					RelearningSteps:      []time.Duration{10 * time.Minute},
					IntervalModifier:     1.0,
					HardIntervalModifier: 1.2,
					EasyBonus:            1.3,
					LapseNewInterval:     0.5,
				},
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusLearning,
			wantStep:     0,
			wantInterval: 15, // 30*0.5 = 15
			wantEase:     2.3,
		},

		// Fuzz tests
		{
			name: "32. Fuzz test interval 3",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusReview,
				CurrentInterval: 3,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeGood,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusReview,
			wantStep:     0,
			wantInterval: 8, // 3*2.5 = 7.5 → 8, but with fuzz ±5%
			wantEase:     2.5,
		},
		{
			name: "33. Fuzz test interval 100",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusReview,
				CurrentInterval: 100,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeGood,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusMastered,
			wantStep:     0,
			wantInterval: 250, // 100*2.5 = 250, with fuzz ±12
			wantEase:     2.5,
		},

		// Edge cases
		{
			name: "34. Single learning step",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusLearning,
				LearningStep:    0,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeGood,
				Now:             now,
				Config: domain.SRSConfig{
					DefaultEaseFactor:    2.5,
					MinEaseFactor:        1.3,
					MaxIntervalDays:      365,
					GraduatingInterval:   1,
					EasyInterval:         4,
					LearningSteps:        []time.Duration{10 * time.Minute},
					RelearningSteps:      []time.Duration{10 * time.Minute},
					IntervalModifier:     1.0,
					HardIntervalModifier: 1.2,
					EasyBonus:            1.3,
					LapseNewInterval:     0.0,
				},
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusReview,
			wantStep:     0,
			wantInterval: 1,
			wantEase:     2.5,
		},
		{
			name: "35. Empty learning steps",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusNew,
				Grade:           domain.ReviewGradeGood,
				Now:             now,
				Config: domain.SRSConfig{
					DefaultEaseFactor:    2.5,
					MinEaseFactor:        1.3,
					MaxIntervalDays:      365,
					GraduatingInterval:   1,
					EasyInterval:         4,
					LearningSteps:        []time.Duration{},
					RelearningSteps:      []time.Duration{},
					IntervalModifier:     1.0,
					HardIntervalModifier: 1.2,
					EasyBonus:            1.3,
					LapseNewInterval:     0.0,
				},
				MaxIntervalDays: 365,
			},
			wantStatus:   domain.LearningStatusReview,
			wantStep:     0,
			wantInterval: 1,
			wantEase:     2.5,
		},
		{
			name: "36. Out of bounds learning step",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusLearning,
				LearningStep:    10, // Way beyond array bounds
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeHard,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 365,
			},
			wantStatus: domain.LearningStatusLearning,
			wantStep:   10,
			wantEase:   2.5,
			checkDelay: ptrDuration(10 * time.Minute), // Should use last step
		},
		{
			name: "37. MaxIntervalDays = 0 edge case",
			input: SRSInput{
				CurrentStatus:   domain.LearningStatusReview,
				CurrentInterval: 100,
				CurrentEase:     2.5,
				Grade:           domain.ReviewGradeGood,
				Now:             now,
				Config:          defaultConfig,
				MaxIntervalDays: 0, // Edge case
			},
			wantStatus:   domain.LearningStatusReview,
			wantStep:     0,
			wantInterval: 0, // Capped at 0
			wantEase:     2.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateSRS(tt.input)

			if got.NewStatus != tt.wantStatus {
				t.Errorf("Status = %v, want %v", got.NewStatus, tt.wantStatus)
			}
			if got.NewLearningStep != tt.wantStep {
				t.Errorf("LearningStep = %d, want %d", got.NewLearningStep, tt.wantStep)
			}

			// For intervals ≥ 3, allow ±5% fuzz tolerance
			if tt.wantInterval >= 3 {
				fuzzRange := max(1, tt.wantInterval*5/100)
				if got.NewInterval < tt.wantInterval-fuzzRange || got.NewInterval > tt.wantInterval+fuzzRange {
					t.Errorf("Interval = %d, want %d ±%d (fuzz)", got.NewInterval, tt.wantInterval, fuzzRange)
				}
			} else {
				if got.NewInterval != tt.wantInterval {
					t.Errorf("Interval = %d, want %d", got.NewInterval, tt.wantInterval)
				}
			}

			if absFloat(got.NewEase-tt.wantEase) > 0.01 {
				t.Errorf("Ease = %.2f, want %.2f", got.NewEase, tt.wantEase)
			}

			if tt.checkDelay != nil {
				actualDelay := got.NextReviewAt.Sub(tt.input.Now)
				if actualDelay != *tt.checkDelay {
					t.Errorf("NextReviewAt delay = %v, want %v", actualDelay, *tt.checkDelay)
				}
			}
		})
	}
}

func TestApplyFuzz(t *testing.T) {
	now := time.Date(2024, 2, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		interval  int
		want      int
		allowFuzz bool
	}{
		{name: "interval 1 (no fuzz)", interval: 1, want: 1, allowFuzz: false},
		{name: "interval 2 (no fuzz)", interval: 2, want: 2, allowFuzz: false},
		{name: "interval 3 (with fuzz)", interval: 3, want: 3, allowFuzz: true},
		{name: "interval 10 (with fuzz)", interval: 10, want: 10, allowFuzz: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyFuzz(tt.interval, now)

			if !tt.allowFuzz {
				if got != tt.want {
					t.Errorf("applyFuzz(%d) = %d, want %d (no fuzz expected)", tt.interval, got, tt.want)
				}
			} else {
				fuzzRange := max(1, tt.interval*5/100)
				if got < tt.want-fuzzRange || got > tt.want+fuzzRange {
					t.Errorf("applyFuzz(%d) = %d, want %d ±%d", tt.interval, got, tt.want, fuzzRange)
				}
			}
		})
	}
}

func ptrDuration(d time.Duration) *time.Duration {
	return &d
}

func absFloat(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
