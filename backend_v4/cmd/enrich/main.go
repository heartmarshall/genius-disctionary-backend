// Command enrich generates per-word enrichment context files from linguistic datasets.
// It reads a word list (from file or enrichment queue) and assembles
// Wiktionary/WordNet/CMU data for each word into enrich-output/<word>.json files,
// plus batch prompt files for manual LLM processing.
//
// Source modes (ENRICH_SOURCE or config source):
//
//	file  (default) — reads words from WordListPath
//	queue — claims words from enrichment_queue via DB
//
// Exit codes: 0 = success, 1 = error.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	enrichmentrepo "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/enrichment"
	"github.com/heartmarshall/myenglish-backend/internal/app/enricher"
	"github.com/heartmarshall/myenglish-backend/internal/config"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	enrichmentsvc "github.com/heartmarshall/myenglish-backend/internal/service/enrichment"
)

func main() {
	enrichConfigPath := flag.String("enrich-config", "", "path to enrich YAML config")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := enricher.LoadConfig(*enrichConfigPath)
	if err != nil {
		logger.Error("load enrich config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if cfg.Source == "queue" {
		runQueueMode(cfg, logger)
	} else {
		if _, err := enricher.Run(context.Background(), cfg, logger); err != nil {
			logger.Error("enrichment failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}
}

func runQueueMode(cfg *enricher.Config, logger *slog.Logger) {
	appCfg, err := config.Load()
	if err != nil {
		logger.Error("load app config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	pool, err := postgres.NewPool(ctx, appCfg.Database)
	if err != nil {
		logger.Error("connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	queueRepo := enrichmentrepo.New(pool)
	queueSvc := enrichmentsvc.NewService(logger, queueRepo)

	// Claim batch from queue.
	items, err := queueSvc.ClaimBatch(ctx, cfg.BatchSize)
	if err != nil {
		logger.Error("claim batch", slog.String("error", err.Error()))
		os.Exit(1)
	}
	if len(items) == 0 {
		logger.Info("no pending items in enrichment queue")
		return
	}

	// Collect ref_entry IDs.
	ids := make([]uuid.UUID, len(items))
	for i, item := range items {
		ids[i] = item.RefEntryID
	}

	// Query ref_entry texts.
	rows, err := pool.Query(ctx, `SELECT id, text FROM ref_entries WHERE id = ANY($1)`, ids)
	if err != nil {
		logger.Error("query ref entry texts", slog.String("error", err.Error()))
		markAllFailed(ctx, queueSvc, items, "failed to query ref entries", logger)
		os.Exit(1)
	}
	defer rows.Close()

	textByID := make(map[uuid.UUID]string, len(items))
	var words []string
	for rows.Next() {
		var id uuid.UUID
		var text string
		if err := rows.Scan(&id, &text); err != nil {
			logger.Error("scan ref entry", slog.String("error", err.Error()))
			continue
		}
		textByID[id] = text
		words = append(words, text)
	}
	if err := rows.Err(); err != nil {
		logger.Error("iterate ref entries", slog.String("error", err.Error()))
	}

	logger.Info("claimed words from queue", slog.Int("count", len(words)))

	// Run enrichment pipeline with claimed words.
	result, err := enricher.RunWithWords(ctx, cfg, words, logger)
	if err != nil {
		logger.Error("enrichment failed", slog.String("error", err.Error()))
		markAllFailed(ctx, queueSvc, items, err.Error(), logger)
		os.Exit(1)
	}

	logger.Info("queue enrichment complete",
		slog.Int("total", result.TotalWords),
		slog.Int("written", result.Written),
		slog.Int("skipped", result.Skipped),
	)
	// Items remain in 'processing' state until LLM output is imported
	// via llm-import, which marks them done/failed.
}

func markAllFailed(ctx context.Context, svc *enrichmentsvc.Service, items []domain.EnrichmentQueueItem, errMsg string, logger *slog.Logger) {
	for _, item := range items {
		if err := svc.MarkFailed(ctx, item.RefEntryID, errMsg); err != nil {
			logger.Error("mark failed", slog.String("ref_entry_id", item.RefEntryID.String()), slog.String("error", err.Error()))
		}
	}
}
