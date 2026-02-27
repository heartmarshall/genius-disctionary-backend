package config

import (
	"slices"
	"time"
)

// Config is the root application configuration.
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Auth       AuthConfig       `yaml:"auth"`
	Dictionary DictionaryConfig `yaml:"dictionary"`
	GraphQL    GraphQLConfig    `yaml:"graphql"`
	Log        LogConfig        `yaml:"log"`
	SRS        SRSConfig        `yaml:"srs"`
	CORS       CORSConfig       `yaml:"cors"`
	RateLimit  RateLimitConfig  `yaml:"rate_limit"`
}

// CORSConfig holds CORS settings.
type CORSConfig struct {
	AllowedOrigins   string `yaml:"allowed_origins"   env:"CORS_ALLOWED_ORIGINS"   env-default:"*"`
	AllowedMethods   string `yaml:"allowed_methods"   env:"CORS_ALLOWED_METHODS"   env-default:"GET,POST,OPTIONS"`
	AllowedHeaders   string `yaml:"allowed_headers"   env:"CORS_ALLOWED_HEADERS"   env-default:"Authorization,Content-Type"`
	AllowCredentials bool   `yaml:"allow_credentials" env:"CORS_ALLOW_CREDENTIALS" env-default:"true"`
	MaxAge           int    `yaml:"max_age"           env:"CORS_MAX_AGE"           env-default:"86400"`
}

// RateLimitConfig holds rate limiting settings for auth endpoints.
type RateLimitConfig struct {
	Enabled         bool          `yaml:"enabled"          env:"RATE_LIMIT_ENABLED"          env-default:"true"`
	Register        int           `yaml:"register"         env:"RATE_LIMIT_REGISTER"         env-default:"5"`
	Login           int           `yaml:"login"            env:"RATE_LIMIT_LOGIN"             env-default:"10"`
	Refresh         int           `yaml:"refresh"          env:"RATE_LIMIT_REFRESH"           env-default:"20"`
	CleanupInterval time.Duration `yaml:"cleanup_interval" env:"RATE_LIMIT_CLEANUP_INTERVAL"  env-default:"5m"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host            string        `yaml:"host"             env:"SERVER_HOST"             env-default:"0.0.0.0"`
	Port            int           `yaml:"port"             env:"SERVER_PORT"             env-default:"8080"`
	ReadTimeout     time.Duration `yaml:"read_timeout"     env:"SERVER_READ_TIMEOUT"     env-default:"10s"`
	WriteTimeout    time.Duration `yaml:"write_timeout"    env:"SERVER_WRITE_TIMEOUT"    env-default:"30s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"     env:"SERVER_IDLE_TIMEOUT"     env-default:"60s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"SERVER_SHUTDOWN_TIMEOUT" env-default:"10s"`
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	DSN             string        `yaml:"dsn"                env:"DATABASE_DSN"                env-required:"true"`
	MaxConns        int32         `yaml:"max_conns"          env:"DATABASE_MAX_CONNS"          env-default:"25"`
	MinConns        int32         `yaml:"min_conns"          env:"DATABASE_MIN_CONNS"          env-default:"5"`
	MaxConnLifetime time.Duration `yaml:"max_conn_lifetime"  env:"DATABASE_MAX_CONN_LIFETIME"  env-default:"1h"`
	MaxConnIdleTime time.Duration `yaml:"max_conn_idle_time" env:"DATABASE_MAX_CONN_IDLE_TIME" env-default:"30m"`
}

// AuthConfig holds authentication and OAuth settings.
type AuthConfig struct {
	JWTSecret          string        `yaml:"jwt_secret"           env:"AUTH_JWT_SECRET"           env-required:"true"`
	JWTIssuer          string        `yaml:"jwt_issuer"           env:"AUTH_JWT_ISSUER"           env-default:"myenglish"`
	AccessTokenTTL     time.Duration `yaml:"access_token_ttl"     env:"AUTH_ACCESS_TOKEN_TTL"     env-default:"15m"`
	RefreshTokenTTL    time.Duration `yaml:"refresh_token_ttl"    env:"AUTH_REFRESH_TOKEN_TTL"    env-default:"720h"`
	PasswordHashCost   int           `yaml:"password_hash_cost"   env:"AUTH_PASSWORD_HASH_COST"   env-default:"12"`
	GoogleClientID     string        `yaml:"google_client_id"     env:"AUTH_GOOGLE_CLIENT_ID"`
	GoogleClientSecret string        `yaml:"google_client_secret" env:"AUTH_GOOGLE_CLIENT_SECRET"`
	GoogleRedirectURI  string        `yaml:"google_redirect_uri"  env:"AUTH_GOOGLE_REDIRECT_URI"`
	AppleKeyID         string        `yaml:"apple_key_id"         env:"AUTH_APPLE_KEY_ID"`
	AppleTeamID        string        `yaml:"apple_team_id"        env:"AUTH_APPLE_TEAM_ID"`
	ApplePrivateKey    string        `yaml:"apple_private_key"    env:"AUTH_APPLE_PRIVATE_KEY"`
}

// DictionaryConfig holds dictionary service settings.
type DictionaryConfig struct {
	MaxEntriesPerUser       int `yaml:"max_entries_per_user"        env:"DICT_MAX_ENTRIES_PER_USER"       env-default:"10000"`
	ImportChunkSize         int `yaml:"import_chunk_size"           env:"DICT_IMPORT_CHUNK_SIZE"          env-default:"50"`
	ExportMaxEntries        int `yaml:"export_max_entries"          env:"DICT_EXPORT_MAX_ENTRIES"         env-default:"10000"`
	HardDeleteRetentionDays int `yaml:"hard_delete_retention_days"  env:"DICT_HARD_DELETE_RETENTION_DAYS" env-default:"30"`
	AuditRetentionDays      int `yaml:"audit_retention_days"        env:"AUDIT_RETENTION_DAYS"            env-default:"365"`
}

// GraphQLConfig holds GraphQL server settings.
type GraphQLConfig struct {
	PlaygroundEnabled     bool `yaml:"playground_enabled"     env:"GRAPHQL_PLAYGROUND_ENABLED"     env-default:"false"`
	IntrospectionEnabled  bool `yaml:"introspection_enabled"  env:"GRAPHQL_INTROSPECTION_ENABLED"  env-default:"false"`
	ComplexityLimit       int  `yaml:"complexity_limit"       env:"GRAPHQL_COMPLEXITY_LIMIT"       env-default:"300"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level  string `yaml:"level"  env:"LOG_LEVEL"  env-default:"info"`
	Format string `yaml:"format" env:"LOG_FORMAT" env-default:"json"`
}

// SRSConfig holds FSRS-5 spaced-repetition system parameters.
type SRSConfig struct {
	DefaultRetention   float64 `yaml:"default_retention"    env:"SRS_DEFAULT_RETENTION"     env-default:"0.9"`
	MaxIntervalDays    int     `yaml:"max_interval_days"    env:"SRS_MAX_INTERVAL"          env-default:"365"`
	EnableFuzz         bool    `yaml:"enable_fuzz"          env:"SRS_ENABLE_FUZZ"           env-default:"true"`
	LearningStepsRaw   string  `yaml:"learning_steps"       env:"SRS_LEARNING_STEPS"        env-default:"1m,10m"`
	RelearningStepsRaw string  `yaml:"relearning_steps"     env:"SRS_RELEARNING_STEPS"      env-default:"10m"`
	NewCardsPerDay     int     `yaml:"new_cards_per_day"    env:"SRS_NEW_CARDS_DAY"         env-default:"20"`
	ReviewsPerDay      int     `yaml:"reviews_per_day"      env:"SRS_REVIEWS_DAY"           env-default:"200"` // Not enforced in queue
	UndoWindowMinutes  int     `yaml:"undo_window_minutes"  env:"SRS_UNDO_WINDOW_MINUTES"   env-default:"10"`

	// LearningSteps is parsed from LearningStepsRaw during validation.
	LearningSteps []time.Duration `yaml:"-" env:"-"`
	// RelearningSteps is parsed from RelearningStepsRaw during validation.
	RelearningSteps []time.Duration `yaml:"-" env:"-"`
}

// AllowedProviders returns the list of configured OAuth providers.
// A provider is considered configured if ALL its required credentials are present.
func (c AuthConfig) AllowedProviders() []string {
	var providers []string
	if c.GoogleClientID != "" && c.GoogleClientSecret != "" {
		providers = append(providers, "google")
	}
	if c.AppleKeyID != "" && c.AppleTeamID != "" && c.ApplePrivateKey != "" {
		providers = append(providers, "apple")
	}
	return providers
}

// IsProviderAllowed checks if the given provider string is configured.
func (c AuthConfig) IsProviderAllowed(provider string) bool {
	return slices.Contains(c.AllowedProviders(), provider)
}
