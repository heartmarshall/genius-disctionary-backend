// Command cleanup physically removes soft-deleted entries and/or old audit
// log records older than their configured retention periods.
// It is intended to be invoked by an external cron job, not as an in-process
// goroutine.
//
// Flags:
//
//	--entries  cleanup soft-deleted entries (default: true)
//	--audit    cleanup audit_log entries   (default: false)
//
// Exit codes: 0 = success, 1 = error.
package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/audit"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/entry"
	"github.com/heartmarshall/myenglish-backend/internal/app"
	"github.com/heartmarshall/myenglish-backend/internal/config"
)

func main() {
	entriesFlag := flag.Bool("entries", true, "cleanup soft-deleted entries older than retention period")
	auditFlag := flag.Bool("audit", false, "cleanup audit_log entries older than retention period")
	flag.Parse()

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

	if *entriesFlag {
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

	if *auditFlag {
		auditRepo := audit.New(pool)
		threshold := time.Now().AddDate(0, 0, -cfg.Dictionary.AuditRetentionDays)

		deleted, err := auditRepo.DeleteOlderThan(ctx, threshold)
		if err != nil {
			logger.Error("audit cleanup failed",
				slog.String("error", err.Error()),
				slog.Time("threshold", threshold),
			)
			os.Exit(1)
		}

		logger.Info("audit cleanup completed",
			slog.Int64("deleted", deleted),
			slog.Time("threshold", threshold),
		)
	}
}
