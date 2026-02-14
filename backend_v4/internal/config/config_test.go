package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// validEnv sets the minimum required env vars for a valid config.
// Returns a cleanup function that restores the previous env state.
func validEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_DSN", "postgres://u:p@localhost:5432/testdb")
	t.Setenv("AUTH_JWT_SECRET", "this-is-a-very-long-jwt-secret-for-testing-32+")
	t.Setenv("AUTH_GOOGLE_CLIENT_ID", "google-id")
	t.Setenv("AUTH_GOOGLE_CLIENT_SECRET", "google-secret")
}

func writeYAML(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
	return path
}

const validYAML = `
server:
  host: "127.0.0.1"
  port: 9090
  read_timeout: "5s"
  write_timeout: "15s"
  idle_timeout: "30s"
  shutdown_timeout: "5s"

database:
  dsn: "postgres://u:p@localhost:5432/testdb"
  max_conns: 10
  min_conns: 2

auth:
  jwt_secret: "this-is-a-very-long-jwt-secret-for-testing-32+"
  google_client_id: "gid"
  google_client_secret: "gsecret"

dictionary:
  max_entries_per_user: 5000
  default_ease_factor: 2.5
  import_chunk_size: 100
  export_max_entries: 8000
  hard_delete_retention_days: 60

graphql:
  playground_enabled: true
  introspection_enabled: true
  complexity_limit: 200

log:
  level: "debug"
  format: "text"

srs:
  default_ease_factor: 2.5
  min_ease_factor: 1.3
  max_interval_days: 365
  graduating_interval: 1
  learning_steps: "1m,10m"
  new_cards_per_day: 20
  reviews_per_day: 200
`

func TestLoad_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := writeYAML(t, dir, validYAML)
	t.Setenv("CONFIG_PATH", path)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Server
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("server.host = %q, want %q", cfg.Server.Host, "127.0.0.1")
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("server.port = %d, want %d", cfg.Server.Port, 9090)
	}
	if cfg.Server.ReadTimeout != 5*time.Second {
		t.Errorf("server.read_timeout = %v, want %v", cfg.Server.ReadTimeout, 5*time.Second)
	}

	// Database
	if cfg.Database.DSN != "postgres://u:p@localhost:5432/testdb" {
		t.Errorf("database.dsn = %q", cfg.Database.DSN)
	}
	if cfg.Database.MaxConns != 10 {
		t.Errorf("database.max_conns = %d, want 10", cfg.Database.MaxConns)
	}

	// Auth
	if cfg.Auth.GoogleClientID != "gid" {
		t.Errorf("auth.google_client_id = %q", cfg.Auth.GoogleClientID)
	}

	// Dictionary
	if cfg.Dictionary.MaxEntriesPerUser != 5000 {
		t.Errorf("dictionary.max_entries_per_user = %d, want 5000", cfg.Dictionary.MaxEntriesPerUser)
	}
	if cfg.Dictionary.DefaultEaseFactor != 2.5 {
		t.Errorf("dictionary.default_ease_factor = %v, want 2.5", cfg.Dictionary.DefaultEaseFactor)
	}
	if cfg.Dictionary.ImportChunkSize != 100 {
		t.Errorf("dictionary.import_chunk_size = %d, want 100", cfg.Dictionary.ImportChunkSize)
	}
	if cfg.Dictionary.ExportMaxEntries != 8000 {
		t.Errorf("dictionary.export_max_entries = %d, want 8000", cfg.Dictionary.ExportMaxEntries)
	}
	if cfg.Dictionary.HardDeleteRetentionDays != 60 {
		t.Errorf("dictionary.hard_delete_retention_days = %d, want 60", cfg.Dictionary.HardDeleteRetentionDays)
	}

	// GraphQL
	if !cfg.GraphQL.PlaygroundEnabled {
		t.Error("graphql.playground_enabled should be true")
	}
	if cfg.GraphQL.ComplexityLimit != 200 {
		t.Errorf("graphql.complexity_limit = %d, want 200", cfg.GraphQL.ComplexityLimit)
	}

	// Log
	if cfg.Log.Level != "debug" {
		t.Errorf("log.level = %q, want %q", cfg.Log.Level, "debug")
	}
	if cfg.Log.Format != "text" {
		t.Errorf("log.format = %q, want %q", cfg.Log.Format, "text")
	}

	// SRS
	if cfg.SRS.DefaultEaseFactor != 2.5 {
		t.Errorf("srs.default_ease_factor = %v, want 2.5", cfg.SRS.DefaultEaseFactor)
	}
	if len(cfg.SRS.LearningSteps) != 2 {
		t.Fatalf("srs.learning_steps len = %d, want 2", len(cfg.SRS.LearningSteps))
	}
	if cfg.SRS.LearningSteps[0] != time.Minute {
		t.Errorf("srs.learning_steps[0] = %v, want 1m", cfg.SRS.LearningSteps[0])
	}
	if cfg.SRS.LearningSteps[1] != 10*time.Minute {
		t.Errorf("srs.learning_steps[1] = %v, want 10m", cfg.SRS.LearningSteps[1])
	}
}

func TestLoad_ENVOverridesYAML(t *testing.T) {
	dir := t.TempDir()
	path := writeYAML(t, dir, validYAML)
	t.Setenv("CONFIG_PATH", path)
	t.Setenv("SERVER_PORT", "3000")
	t.Setenv("LOG_LEVEL", "warn")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.Port != 3000 {
		t.Errorf("server.port = %d, want 3000 (ENV override)", cfg.Server.Port)
	}
	if cfg.Log.Level != "warn" {
		t.Errorf("log.level = %q, want %q (ENV override)", cfg.Log.Level, "warn")
	}
}

func TestLoad_NoFile_ENVOnly(t *testing.T) {
	validEnv(t)

	// Point CONFIG_PATH to a non-default location that doesn't exist
	// to trigger the explicit-path error; instead, unset CONFIG_PATH so
	// fallback kicks in and the file is just absent.
	t.Setenv("CONFIG_PATH", "")
	// Set working dir to a temp dir with no config.yaml
	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	_ = os.Chdir(t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("server.port = %d, want 8080 (default)", cfg.Server.Port)
	}
}

func TestLoad_ExplicitPathNotFound(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/nonexistent/config.yaml")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing explicit config path")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := writeYAML(t, dir, `{{{invalid yaml`)
	t.Setenv("CONFIG_PATH", path)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestValidate_JWTSecretTooShort(t *testing.T) {
	cfg := validConfig()
	cfg.Auth.JWTSecret = "short"

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for short JWT secret")
	}
}

func TestValidate_JWTSecretEmpty(t *testing.T) {
	cfg := validConfig()
	cfg.Auth.JWTSecret = ""

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for empty JWT secret")
	}
}

func TestValidate_NoOAuthProvider(t *testing.T) {
	cfg := validConfig()
	cfg.Auth.GoogleClientID = ""
	cfg.Auth.GoogleClientSecret = ""
	cfg.Auth.AppleKeyID = ""
	cfg.Auth.AppleTeamID = ""
	cfg.Auth.ApplePrivateKey = ""

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when no OAuth provider configured")
	}
}

func TestValidate_AppleOAuthOnly(t *testing.T) {
	cfg := validConfig()
	cfg.Auth.GoogleClientID = ""
	cfg.Auth.GoogleClientSecret = ""
	cfg.Auth.AppleKeyID = "key"
	cfg.Auth.AppleTeamID = "team"
	cfg.Auth.ApplePrivateKey = "pk"

	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error with Apple OAuth only: %v", err)
	}
}

func TestValidate_SRS_MinEaseFactorZero(t *testing.T) {
	cfg := validConfig()
	cfg.SRS.MinEaseFactor = 0

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for MinEaseFactor = 0")
	}
}

func TestValidate_SRS_MinEaseFactorNegative(t *testing.T) {
	cfg := validConfig()
	cfg.SRS.MinEaseFactor = -1

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for negative MinEaseFactor")
	}
}

func TestValidate_SRS_MaxIntervalDaysZero(t *testing.T) {
	cfg := validConfig()
	cfg.SRS.MaxIntervalDays = 0

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for MaxIntervalDays = 0")
	}
}

func TestValidate_SRS_NewCardsPerDayNegative(t *testing.T) {
	cfg := validConfig()
	cfg.SRS.NewCardsPerDay = -1

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for negative NewCardsPerDay")
	}
}

func TestValidate_Dictionary_MaxEntriesPerUserZero(t *testing.T) {
	cfg := validConfig()
	cfg.Dictionary.MaxEntriesPerUser = 0

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for MaxEntriesPerUser = 0")
	}
}

func TestValidate_Dictionary_MaxEntriesPerUserNegative(t *testing.T) {
	cfg := validConfig()
	cfg.Dictionary.MaxEntriesPerUser = -1

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for negative MaxEntriesPerUser")
	}
}

func TestValidate_Dictionary_DefaultEaseFactorTooLow(t *testing.T) {
	cfg := validConfig()
	cfg.Dictionary.DefaultEaseFactor = 0.5

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for DefaultEaseFactor < 1.0")
	}
}

func TestValidate_Dictionary_DefaultEaseFactorTooHigh(t *testing.T) {
	cfg := validConfig()
	cfg.Dictionary.DefaultEaseFactor = 5.1

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for DefaultEaseFactor > 5.0")
	}
}

func TestValidate_Dictionary_ImportChunkSizeZero(t *testing.T) {
	cfg := validConfig()
	cfg.Dictionary.ImportChunkSize = 0

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for ImportChunkSize = 0")
	}
}

func TestValidate_Dictionary_ImportChunkSizeTooLarge(t *testing.T) {
	cfg := validConfig()
	cfg.Dictionary.ImportChunkSize = 1001

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for ImportChunkSize > 1000")
	}
}

func TestValidate_Dictionary_ExportMaxEntriesZero(t *testing.T) {
	cfg := validConfig()
	cfg.Dictionary.ExportMaxEntries = 0

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for ExportMaxEntries = 0")
	}
}

func TestValidate_Dictionary_HardDeleteRetentionDaysZero(t *testing.T) {
	cfg := validConfig()
	cfg.Dictionary.HardDeleteRetentionDays = 0

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for HardDeleteRetentionDays = 0")
	}
}

func TestValidate_Dictionary_HardDeleteRetentionDaysNegative(t *testing.T) {
	cfg := validConfig()
	cfg.Dictionary.HardDeleteRetentionDays = -7

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for negative HardDeleteRetentionDays")
	}
}

func TestValidate_Dictionary_ValidBoundaryValues(t *testing.T) {
	cfg := validConfig()
	cfg.Dictionary.DefaultEaseFactor = 1.0
	cfg.Dictionary.ImportChunkSize = 1

	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error for boundary values: %v", err)
	}

	cfg.Dictionary.DefaultEaseFactor = 5.0
	cfg.Dictionary.ImportChunkSize = 1000

	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error for upper boundary values: %v", err)
	}
}

func TestParseLearningSteps_Valid(t *testing.T) {
	steps, err := ParseLearningSteps("1m,10m")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("len = %d, want 2", len(steps))
	}
	if steps[0] != time.Minute {
		t.Errorf("[0] = %v, want 1m", steps[0])
	}
	if steps[1] != 10*time.Minute {
		t.Errorf("[1] = %v, want 10m", steps[1])
	}
}

func TestParseLearningSteps_WithSpaces(t *testing.T) {
	steps, err := ParseLearningSteps(" 1m , 10m , 1h ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 3 {
		t.Fatalf("len = %d, want 3", len(steps))
	}
	if steps[2] != time.Hour {
		t.Errorf("[2] = %v, want 1h", steps[2])
	}
}

func TestParseLearningSteps_Empty(t *testing.T) {
	steps, err := ParseLearningSteps("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if steps != nil {
		t.Errorf("expected nil, got %v", steps)
	}
}

func TestParseLearningSteps_InvalidFormat(t *testing.T) {
	_, err := ParseLearningSteps("1m,invalid,10m")
	if err == nil {
		t.Fatal("expected error for invalid duration")
	}
}

func TestParseLearningSteps_SingleStep(t *testing.T) {
	steps, err := ParseLearningSteps("5m")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("len = %d, want 1", len(steps))
	}
	if steps[0] != 5*time.Minute {
		t.Errorf("[0] = %v, want 5m", steps[0])
	}
}

// validConfig returns a Config that passes all validation checks.
func validConfig() Config {
	return Config{
		Auth: AuthConfig{
			JWTSecret:          "this-is-a-very-long-jwt-secret-for-testing-32+",
			GoogleClientID:     "gid",
			GoogleClientSecret: "gsecret",
		},
		Dictionary: DictionaryConfig{
			MaxEntriesPerUser:       10000,
			DefaultEaseFactor:       2.5,
			ImportChunkSize:         50,
			ExportMaxEntries:        10000,
			HardDeleteRetentionDays: 30,
		},
		SRS: SRSConfig{
			DefaultEaseFactor:  2.5,
			MinEaseFactor:      1.3,
			MaxIntervalDays:    365,
			GraduatingInterval: 1,
			LearningStepsRaw:   "1m,10m",
			NewCardsPerDay:     20,
			ReviewsPerDay:      200,
		},
	}
}
