package config

import (
	"fmt"
	"strings"
	"time"
)

// Validate performs business-rule validation on the loaded configuration.
// It must be called after loading; Load calls it automatically.
func (c *Config) Validate() error {
	if len(c.Auth.JWTSecret) < 32 {
		return fmt.Errorf("auth.jwt_secret must be at least 32 characters (got %d)", len(c.Auth.JWTSecret))
	}

	if len(c.Auth.AllowedProviders()) == 0 {
		return fmt.Errorf("at least one OAuth provider must be configured (Google or Apple)")
	}

	if err := c.Dictionary.validate(); err != nil {
		return fmt.Errorf("dictionary: %w", err)
	}

	if err := c.SRS.validate(); err != nil {
		return fmt.Errorf("srs: %w", err)
	}

	return nil
}

func (d *DictionaryConfig) validate() error {
	if d.MaxEntriesPerUser <= 0 {
		return fmt.Errorf("max_entries_per_user must be positive (got %d)", d.MaxEntriesPerUser)
	}
	if d.DefaultEaseFactor < 1.0 || d.DefaultEaseFactor > 5.0 {
		return fmt.Errorf("default_ease_factor must be between 1.0 and 5.0 (got %v)", d.DefaultEaseFactor)
	}
	if d.ImportChunkSize <= 0 || d.ImportChunkSize > 1000 {
		return fmt.Errorf("import_chunk_size must be between 1 and 1000 (got %d)", d.ImportChunkSize)
	}
	if d.ExportMaxEntries <= 0 {
		return fmt.Errorf("export_max_entries must be positive (got %d)", d.ExportMaxEntries)
	}
	if d.HardDeleteRetentionDays <= 0 {
		return fmt.Errorf("hard_delete_retention_days must be positive (got %d)", d.HardDeleteRetentionDays)
	}
	return nil
}

func (s *SRSConfig) validate() error {
	if s.MinEaseFactor <= 0 {
		return fmt.Errorf("min_ease_factor must be > 0 (got %v)", s.MinEaseFactor)
	}
	if s.MaxIntervalDays <= 0 {
		return fmt.Errorf("max_interval_days must be > 0 (got %d)", s.MaxIntervalDays)
	}
	if s.NewCardsPerDay < 0 {
		return fmt.Errorf("new_cards_per_day must be >= 0 (got %d)", s.NewCardsPerDay)
	}

	steps, err := ParseLearningSteps(s.LearningStepsRaw)
	if err != nil {
		return fmt.Errorf("learning_steps: %w", err)
	}
	s.LearningSteps = steps

	// New SRS fields validation
	if s.EasyInterval < 1 {
		return fmt.Errorf("easy_interval must be >= 1")
	}
	if s.IntervalModifier <= 0 {
		return fmt.Errorf("interval_modifier must be positive")
	}
	if s.HardIntervalModifier <= 0 {
		return fmt.Errorf("hard_interval_modifier must be positive")
	}
	if s.EasyBonus <= 0 {
		return fmt.Errorf("easy_bonus must be positive")
	}
	if s.LapseNewInterval < 0 || s.LapseNewInterval > 1 {
		return fmt.Errorf("lapse_new_interval must be between 0.0 and 1.0")
	}
	if s.UndoWindowMinutes < 1 {
		return fmt.Errorf("undo_window_minutes must be >= 1")
	}

	// Parse RelearningStepsRaw → RelearningSteps (similar to LearningSteps)
	if s.RelearningStepsRaw != "" {
		relearningSteps, err := ParseLearningSteps(s.RelearningStepsRaw)
		if err != nil {
			return fmt.Errorf("relearning_steps: %w", err)
		}
		s.RelearningSteps = relearningSteps
	}

	return nil
}

// ParseLearningSteps parses a comma-separated string of durations (e.g. "1m,10m")
// into a slice of time.Duration. An empty string returns a nil slice.
func ParseLearningSteps(raw string) ([]time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	steps := make([]time.Duration, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		d, err := time.ParseDuration(p)
		if err != nil {
			return nil, fmt.Errorf("invalid duration %q: %w", p, err)
		}
		steps = append(steps, d)
	}

	return steps, nil
}
