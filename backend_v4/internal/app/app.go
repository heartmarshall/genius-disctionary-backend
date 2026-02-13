package app

import (
	"context"
	"log/slog"

	"github.com/heartmarshall/myenglish-backend/internal/config"
)

// Run is the application entry point. It loads configuration, initializes
// the logger, and logs startup information. In later phases it will be
// extended to connect to the database, create services, and start the
// HTTP server.
func Run(_ context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := NewLogger(cfg.Log)

	logger.Info("starting application",
		slog.String("version", BuildVersion()),
		slog.String("log_level", cfg.Log.Level),
	)

	return nil
}
