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

	if !c.hasGoogleOAuth() && !c.hasAppleOAuth() {
		return fmt.Errorf("at least one OAuth provider must be configured (Google or Apple)")
	}

	if err := c.SRS.validate(); err != nil {
		return fmt.Errorf("srs: %w", err)
	}

	return nil
}

func (c *Config) hasGoogleOAuth() bool {
	return c.Auth.GoogleClientID != "" && c.Auth.GoogleClientSecret != ""
}

func (c *Config) hasAppleOAuth() bool {
	return c.Auth.AppleKeyID != "" && c.Auth.AppleTeamID != "" && c.Auth.ApplePrivateKey != ""
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
