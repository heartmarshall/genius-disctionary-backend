package config

import (
	"testing"
)

func TestAuthConfig_AllowedProviders_BothConfigured(t *testing.T) {
	t.Parallel()

	cfg := AuthConfig{
		GoogleClientID:     "google-id",
		GoogleClientSecret: "google-secret",
		AppleKeyID:         "apple-key",
		AppleTeamID:        "apple-team",
		ApplePrivateKey:    "apple-private-key",
	}

	allowed := cfg.AllowedProviders()

	if len(allowed) != 2 {
		t.Errorf("expected 2 providers, got %d: %v", len(allowed), allowed)
	}

	if allowed[0] != "google" || allowed[1] != "apple" {
		t.Errorf("expected [google, apple], got %v", allowed)
	}
}

func TestAuthConfig_AllowedProviders_OnlyGoogle(t *testing.T) {
	t.Parallel()

	cfg := AuthConfig{
		GoogleClientID:     "google-id",
		GoogleClientSecret: "google-secret",
	}

	allowed := cfg.AllowedProviders()

	if len(allowed) != 1 {
		t.Errorf("expected 1 provider, got %d: %v", len(allowed), allowed)
	}

	if allowed[0] != "google" {
		t.Errorf("expected [google], got %v", allowed)
	}
}

func TestAuthConfig_AllowedProviders_OnlyApple(t *testing.T) {
	t.Parallel()

	cfg := AuthConfig{
		AppleKeyID:      "apple-key",
		AppleTeamID:     "apple-team",
		ApplePrivateKey: "apple-private-key",
	}

	allowed := cfg.AllowedProviders()

	if len(allowed) != 1 {
		t.Errorf("expected 1 provider, got %d: %v", len(allowed), allowed)
	}

	if allowed[0] != "apple" {
		t.Errorf("expected [apple], got %v", allowed)
	}
}

func TestAuthConfig_AllowedProviders_NoneConfigured(t *testing.T) {
	t.Parallel()

	cfg := AuthConfig{}

	allowed := cfg.AllowedProviders()

	if len(allowed) != 0 {
		t.Errorf("expected 0 providers, got %d: %v", len(allowed), allowed)
	}
}

func TestAuthConfig_AllowedProviders_PartialGoogle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		cfg    AuthConfig
		expect int
	}{
		{
			name: "only client ID",
			cfg: AuthConfig{
				GoogleClientID: "google-id",
			},
			expect: 0,
		},
		{
			name: "only client secret",
			cfg: AuthConfig{
				GoogleClientSecret: "google-secret",
			},
			expect: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			allowed := tt.cfg.AllowedProviders()

			if len(allowed) != tt.expect {
				t.Errorf("expected %d providers, got %d: %v", tt.expect, len(allowed), allowed)
			}
		})
	}
}

func TestAuthConfig_AllowedProviders_PartialApple(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		cfg    AuthConfig
		expect int
	}{
		{
			name: "only key ID",
			cfg: AuthConfig{
				AppleKeyID: "apple-key",
			},
			expect: 0,
		},
		{
			name: "only team ID",
			cfg: AuthConfig{
				AppleTeamID: "apple-team",
			},
			expect: 0,
		},
		{
			name: "only private key",
			cfg: AuthConfig{
				ApplePrivateKey: "apple-private-key",
			},
			expect: 0,
		},
		{
			name: "key ID and team ID, missing private key",
			cfg: AuthConfig{
				AppleKeyID:  "apple-key",
				AppleTeamID: "apple-team",
			},
			expect: 0,
		},
		{
			name: "key ID and private key, missing team ID",
			cfg: AuthConfig{
				AppleKeyID:      "apple-key",
				ApplePrivateKey: "apple-private-key",
			},
			expect: 0,
		},
		{
			name: "team ID and private key, missing key ID",
			cfg: AuthConfig{
				AppleTeamID:     "apple-team",
				ApplePrivateKey: "apple-private-key",
			},
			expect: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			allowed := tt.cfg.AllowedProviders()

			if len(allowed) != tt.expect {
				t.Errorf("expected %d providers, got %d: %v", tt.expect, len(allowed), allowed)
			}
		})
	}
}

func TestAuthConfig_IsProviderAllowed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cfg      AuthConfig
		provider string
		want     bool
	}{
		{
			name: "google allowed",
			cfg: AuthConfig{
				GoogleClientID:     "google-id",
				GoogleClientSecret: "google-secret",
			},
			provider: "google",
			want:     true,
		},
		{
			name: "apple allowed",
			cfg: AuthConfig{
				AppleKeyID:      "apple-key",
				AppleTeamID:     "apple-team",
				ApplePrivateKey: "apple-private-key",
			},
			provider: "apple",
			want:     true,
		},
		{
			name: "google not allowed",
			cfg: AuthConfig{
				AppleKeyID:      "apple-key",
				AppleTeamID:     "apple-team",
				ApplePrivateKey: "apple-private-key",
			},
			provider: "google",
			want:     false,
		},
		{
			name: "apple not allowed",
			cfg: AuthConfig{
				GoogleClientID:     "google-id",
				GoogleClientSecret: "google-secret",
			},
			provider: "apple",
			want:     false,
		},
		{
			name:     "unknown provider",
			cfg:      AuthConfig{},
			provider: "facebook",
			want:     false,
		},
		{
			name: "both configured, check google",
			cfg: AuthConfig{
				GoogleClientID:     "google-id",
				GoogleClientSecret: "google-secret",
				AppleKeyID:         "apple-key",
				AppleTeamID:        "apple-team",
				ApplePrivateKey:    "apple-private-key",
			},
			provider: "google",
			want:     true,
		},
		{
			name: "both configured, check apple",
			cfg: AuthConfig{
				GoogleClientID:     "google-id",
				GoogleClientSecret: "google-secret",
				AppleKeyID:         "apple-key",
				AppleTeamID:        "apple-team",
				ApplePrivateKey:    "apple-private-key",
			},
			provider: "apple",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.cfg.IsProviderAllowed(tt.provider)

			if got != tt.want {
				t.Errorf("IsProviderAllowed(%q) = %v, want %v", tt.provider, got, tt.want)
			}
		})
	}
}
