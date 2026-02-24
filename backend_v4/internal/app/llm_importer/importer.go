package llm_importer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/app/seeder"
)

// Result holds import statistics.
type Result struct {
	FilesProcessed int
	Inserted       int
	Skipped        int
	Errors         int
}

// Run scans llmOutputDir for *.json files, validates, maps, and bulk-inserts them.
func Run(ctx context.Context, cfg *Config, repo seeder.RefEntryBulkRepo, log *slog.Logger) (Result, error) {
	files, err := filepath.Glob(filepath.Join(cfg.LLMOutputDir, "*.json"))
	if err != nil {
		return Result{}, fmt.Errorf("glob llm output dir: %w", err)
	}

	var result Result

	var (
		domainEntries      []domain.RefEntry
		domainSenses       []domain.RefSense
		domainTranslations []domain.RefTranslation
		domainExamples     []domain.RefExample
	)

	flush := func() error {
		if cfg.DryRun || len(domainEntries) == 0 {
			return nil
		}
		n, err := repo.BulkInsertEntries(ctx, domainEntries)
		if err != nil {
			return fmt.Errorf("bulk insert entries: %w", err)
		}
		result.Inserted += n
		result.Skipped += len(domainEntries) - n

		if _, err := repo.BulkInsertSenses(ctx, domainSenses); err != nil {
			return fmt.Errorf("bulk insert senses: %w", err)
		}
		if _, err := repo.BulkInsertTranslations(ctx, domainTranslations); err != nil {
			return fmt.Errorf("bulk insert translations: %w", err)
		}
		if _, err := repo.BulkInsertExamples(ctx, domainExamples); err != nil {
			return fmt.Errorf("bulk insert examples: %w", err)
		}

		domainEntries = domainEntries[:0]
		domainSenses = domainSenses[:0]
		domainTranslations = domainTranslations[:0]
		domainExamples = domainExamples[:0]
		return nil
	}

	for _, path := range files {
		result.FilesProcessed++

		data, err := os.ReadFile(path)
		if err != nil {
			log.Error("read file", slog.String("path", path), slog.String("error", err.Error()))
			result.Errors++
			continue
		}

		var entry LLMWordEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			log.Error("unmarshal JSON", slog.String("path", path), slog.String("error", err.Error()))
			result.Errors++
			continue
		}

		if entry.SourceSlug == "" {
			entry.SourceSlug = cfg.SourceSlug
		}

		if err := Validate(entry); err != nil {
			log.Error("invalid entry", slog.String("path", path), slog.String("error", err.Error()))
			result.Errors++
			continue
		}

		mapped := Map(entry)
		domainEntries = append(domainEntries, mapped.Entry)
		domainSenses = append(domainSenses, mapped.Senses...)
		domainTranslations = append(domainTranslations, mapped.Translations...)
		domainExamples = append(domainExamples, mapped.Examples...)

		if len(domainEntries) >= cfg.BatchSize {
			if err := flush(); err != nil {
				return result, err
			}
		}
	}

	if err := flush(); err != nil {
		return result, err
	}

	log.Info("llm-import complete",
		slog.Int("files", result.FilesProcessed),
		slog.Int("inserted", result.Inserted),
		slog.Int("skipped", result.Skipped),
		slog.Int("errors", result.Errors),
	)
	return result, nil
}
