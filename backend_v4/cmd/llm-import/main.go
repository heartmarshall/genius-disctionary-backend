// Command llm-import ingests LLM-generated word entries into the reference catalog.
// It reads *.json files from a configured output directory, validates and maps them
// to domain types, then bulk-inserts them into PostgreSQL.
//
// Flags:
//
//	--config         path to app config YAML (optional; falls back to env)
//	--import-config  path to llm-import config YAML (optional; falls back to env)
//
// Exit codes: 0 = success, 1 = error.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/refentry"
	"github.com/heartmarshall/myenglish-backend/internal/app"
	"github.com/heartmarshall/myenglish-backend/internal/config"
	"github.com/heartmarshall/myenglish-backend/internal/llm_importer"
)

func main() {
	importConfigPath := flag.String("import-config", "", "path to llm-import config YAML")
	flag.Parse()

	// Load app config (for DB connection and logging).
	appCfg, err := config.Load()
	if err != nil {
		log.Fatalf("load app config: %v", err)
	}

	logger := app.NewLogger(appCfg.Log)

	// Load llm-import config.
	importCfg, err := llm_importer.LoadConfig(*importConfigPath)
	if err != nil {
		logger.Error("load import config", "error", err)
		os.Exit(1)
	}

	// 30-minute context timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Connect to DB.
	pool, err := postgres.NewPool(ctx, appCfg.Database)
	if err != nil {
		logger.Error("connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	txm := postgres.NewTxManager(pool)
	repo := refentry.New(pool, txm)

	if importCfg.DryRun {
		logger.Info("dry-run mode: no DB writes")
	}

	if _, err := llm_importer.Run(ctx, importCfg, repo, logger); err != nil {
		logger.Error("import failed", "error", err)
		os.Exit(1)
	}
}
