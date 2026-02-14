// Command cleanup physically removes soft-deleted entries older than the
// configured retention period. It is intended to be invoked by an external
// cron job, not as an in-process goroutine.
//
// Exit codes: 0 = success, 1 = error.
package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/entry"
	"github.com/heartmarshall/myenglish-backend/internal/app"
	"github.com/heartmarshall/myenglish-backend/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := app.NewLogger(cfg.Log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		logger.Error("connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	entryRepo := entry.New(pool)

	threshold := time.Now().AddDate(0, 0, -cfg.Dictionary.HardDeleteRetentionDays)

	deleted, err := entryRepo.HardDeleteOld(ctx, threshold)
	if err != nil {
		logger.Error("hard delete failed",
			slog.String("error", err.Error()),
			slog.Time("threshold", threshold),
		)
		os.Exit(1)
	}

	logger.Info("hard delete completed",
		slog.Int64("deleted", deleted),
		slog.Time("threshold", threshold),
	)
}
