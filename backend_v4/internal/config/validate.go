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

	if c.Auth.PasswordHashCost < 4 || c.Auth.PasswordHashCost > 31 {
		return fmt.Errorf("auth.password_hash_cost must be between 4 and 31 (got %d)", c.Auth.PasswordHashCost)
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
	if s.DefaultRetention <= 0 || s.DefaultRetention >= 1 {
		return fmt.Errorf("default_retention must be between 0 and 1 exclusive (got %v)", s.DefaultRetention)
	}
	if s.MaxIntervalDays <= 0 {
		return fmt.Errorf("max_interval_days must be > 0 (got %d)", s.MaxIntervalDays)
	}
	if s.NewCardsPerDay < 0 {
		return fmt.Errorf("new_cards_per_day must be >= 0 (got %d)", s.NewCardsPerDay)
	}
	if s.UndoWindowMinutes < 1 {
		return fmt.Errorf("undo_window_minutes must be >= 1")
	}

	steps, err := ParseLearningSteps(s.LearningStepsRaw)
	if err != nil {
		return fmt.Errorf("learning_steps: %w", err)
	}
	s.LearningSteps = steps

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
