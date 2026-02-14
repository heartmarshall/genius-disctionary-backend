package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/config"
)

// Run is the application entry point. It loads configuration, initializes
// the logger, connects to the database, and waits for a shutdown signal.
func Run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := NewLogger(cfg.Log)

	logger.Info("starting application",
		slog.String("version", BuildVersion()),
		slog.String("log_level", cfg.Log.Level),
	)

	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	logger.Info("database connected",
		slog.Int("max_conns", int(cfg.Database.MaxConns)),
	)

	<-ctx.Done()
	logger.Info("shutting down")

	return nil
}
