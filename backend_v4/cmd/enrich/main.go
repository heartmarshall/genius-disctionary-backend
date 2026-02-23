// Command enrich generates per-word enrichment context files from linguistic datasets.
// It reads a word list and assembles Wiktionary/WordNet/CMU data for each word into
// enrich-output/<word>.json files, plus batch word list files for manual LLM processing.
// In api mode it automatically calls the configured LLM API.
//
// Flags:
//
//	--enrich-config  path to enrich YAML config (optional; falls back to env)
//
// Exit codes: 0 = success, 1 = error.
package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/heartmarshall/myenglish-backend/internal/app"
	"github.com/heartmarshall/myenglish-backend/internal/config"
	"github.com/heartmarshall/myenglish-backend/internal/enricher"
)

func main() {
	enrichConfigPath := flag.String("enrich-config", "", "path to enrich YAML config")
	flag.Parse()

	appCfg, err := config.Load()
	if err != nil {
		// Fall back to a basic logger before the app logger is available.
		slog.Error("load app config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger := app.NewLogger(appCfg.Log)

	cfg, err := enricher.LoadConfig(*enrichConfigPath)
	if err != nil {
		logger.Error("load enrich config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if _, err := enricher.Run(cfg, logger); err != nil {
		logger.Error("enrichment failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
