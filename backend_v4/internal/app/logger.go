package app

import (
	"log/slog"
	"os"
	"strings"

	"github.com/heartmarshall/myenglish-backend/internal/config"
)

// NewLogger creates a *slog.Logger based on the provided LogConfig
// and sets it as the default logger via slog.SetDefault.
//
// Format "json" produces structured JSON output (production).
// Format "text" produces human-readable output with source info (development).
// Level is one of: debug, info, warn, error (case-insensitive); defaults to info.
// Output is always os.Stderr.
func NewLogger(cfg config.LogConfig) *slog.Logger {
	level := parseLevel(cfg.Level)

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: strings.EqualFold(cfg.Format, "text"),
	}

	var handler slog.Handler
	if strings.EqualFold(cfg.Format, "json") {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
