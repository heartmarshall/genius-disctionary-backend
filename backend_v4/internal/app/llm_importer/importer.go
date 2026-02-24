package llm_importer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/app/seeder"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// EnrichmentQueue is an optional dependency for updating enrichment queue status.
type EnrichmentQueue interface {
	MarkDone(ctx context.Context, refEntryID uuid.UUID) error
	MarkFailed(ctx context.Context, refEntryID uuid.UUID, errMsg string) error
}

// Result holds import statistics.
type Result struct {
	FilesProcessed int
	Inserted       int
	Replaced       int
	Skipped        int
	Errors         int
}

// Run scans llmOutputDir for *.json files, validates, maps, and imports them.
// For words that already exist in ref_entries, it replaces their content.
// For new words, it bulk-inserts them.
func Run(ctx context.Context, cfg *Config, repo seeder.RefEntryBulkRepo, queue EnrichmentQueue, log *slog.Logger) (Result, error) {
	files, err := filepath.Glob(filepath.Join(cfg.LLMOutputDir, "*.json"))
	if err != nil {
		return Result{}, fmt.Errorf("glob llm output dir: %w", err)
	}

	var result Result

	// Collect all entries first to batch-lookup existing ones.
	type parsedFile struct {
		path  string
		entry LLMWordEntry
	}
	var parsed []parsedFile

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

		parsed = append(parsed, parsedFile{path: path, entry: entry})
	}

	if len(parsed) == 0 {
		log.Info("no valid files to import")
		return result, nil
	}

	// Batch-lookup existing entries.
	texts := make([]string, len(parsed))
	for i, p := range parsed {
		texts[i] = domain.NormalizeText(p.entry.Word)
	}
	existingIDs, err := repo.GetEntryIDsByNormalizedTexts(ctx, texts)
	if err != nil {
		return result, fmt.Errorf("lookup existing entries: %w", err)
	}

	// Separate into replace vs insert.
	var (
		newEntries      []domain.RefEntry
		newSenses       []domain.RefSense
		newTranslations []domain.RefTranslation
		newExamples     []domain.RefExample
	)

	for _, p := range parsed {
		normalized := domain.NormalizeText(p.entry.Word)
		mapped := Map(p.entry)

		if existingID, exists := existingIDs[normalized]; exists {
			// Replace: rewrite senses/translations/examples for existing entry.
			if cfg.DryRun {
				result.Replaced++
				continue
			}

			// Remap senses to point to the existing entry ID.
			for i := range mapped.Senses {
				mapped.Senses[i].RefEntryID = existingID
			}

			if err := repo.ReplaceEntryContent(ctx, existingID, mapped.Senses, mapped.Translations, mapped.Examples); err != nil {
				log.Error("replace entry content", slog.String("word", p.entry.Word), slog.String("error", err.Error()))
				result.Errors++
				if queue != nil {
					_ = queue.MarkFailed(ctx, existingID, err.Error())
				}
				continue
			}
			result.Replaced++

			if queue != nil {
				_ = queue.MarkDone(ctx, existingID)
			}
		} else {
			// New entry: accumulate for bulk insert.
			newEntries = append(newEntries, mapped.Entry)
			newSenses = append(newSenses, mapped.Senses...)
			newTranslations = append(newTranslations, mapped.Translations...)
			newExamples = append(newExamples, mapped.Examples...)
		}
	}

	// Flush new entries via bulk insert.
	if !cfg.DryRun && len(newEntries) > 0 {
		n, err := repo.BulkInsertEntries(ctx, newEntries)
		if err != nil {
			return result, fmt.Errorf("bulk insert entries: %w", err)
		}
		result.Inserted += n
		result.Skipped += len(newEntries) - n

		if _, err := repo.BulkInsertSenses(ctx, newSenses); err != nil {
			return result, fmt.Errorf("bulk insert senses: %w", err)
		}
		if _, err := repo.BulkInsertTranslations(ctx, newTranslations); err != nil {
			return result, fmt.Errorf("bulk insert translations: %w", err)
		}
		if _, err := repo.BulkInsertExamples(ctx, newExamples); err != nil {
			return result, fmt.Errorf("bulk insert examples: %w", err)
		}
	}

	log.Info("llm-import complete",
		slog.Int("files", result.FilesProcessed),
		slog.Int("inserted", result.Inserted),
		slog.Int("replaced", result.Replaced),
		slog.Int("skipped", result.Skipped),
		slog.Int("errors", result.Errors),
	)
	return result, nil
}
