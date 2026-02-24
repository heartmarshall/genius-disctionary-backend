// Command seeder populates the reference catalog with data from external
// linguistic datasets (Wiktionary, NGSL, NAWL, CMU, WordNet, Tatoeba).
// It is intended to be run offline, not as part of the main server.
//
// Flags:
//
//	--phase          comma-separated list of phases to run (default: all)
//	--dry-run        parse datasets without writing to DB
//	--seeder-config  path to seeder YAML config file
//
// Exit codes: 0 = success, 1 = error.
package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/refentry"
	"github.com/heartmarshall/myenglish-backend/internal/app"
	"github.com/heartmarshall/myenglish-backend/internal/config"
	"github.com/heartmarshall/myenglish-backend/internal/app/seeder"
)

// Compile-time interface assertion.
var _ seeder.RefEntryBulkRepo = (*refentry.Repo)(nil)

func main() {
	phaseFlag := flag.String("phase", "", "comma-separated phases to run (default: all)")
	dryRunFlag := flag.Bool("dry-run", false, "parse datasets without writing to DB")
	seederConfigFlag := flag.String("seeder-config", "", "path to seeder YAML config file")
	flag.Parse()

	// Load app config (for DB connection).
	appCfg, err := config.Load()
	if err != nil {
		log.Fatalf("load app config: %v", err)
	}

	logger := app.NewLogger(appCfg.Log)

	// Load seeder config.
	seederCfg, err := seeder.LoadConfig(*seederConfigFlag)
	if err != nil {
		logger.Error("load seeder config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// CLI flags override config.
	if *dryRunFlag {
		seederCfg.DryRun = true
	}

	// Parse phase filter.
	var phases []string
	if *phaseFlag != "" {
		phases = strings.Split(*phaseFlag, ",")
		for i := range phases {
			phases[i] = strings.TrimSpace(phases[i])
		}
	}

	// 30-minute context timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Connect to DB.
	pool, err := postgres.NewPool(ctx, appCfg.Database)
	if err != nil {
		logger.Error("connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	txm := postgres.NewTxManager(pool)
	repo := refentry.New(pool, txm)

	// Run pipeline.
	pipeline := seeder.NewPipeline(logger, repo, *seederCfg)
	if err := pipeline.Run(ctx, phases); err != nil {
		logger.Error("pipeline failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if pipeline.HasErrors() {
		logger.Warn("pipeline completed with errors")
		os.Exit(1)
	}

	logger.Info("pipeline completed successfully")
}
