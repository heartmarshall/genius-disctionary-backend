# LLM Enrichment Pipeline — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a two-tool pipeline — `cmd/enrich/` assembles per-word context from datasets, `cmd/llm-import/` bulk-inserts LLM-generated JSON into the ref catalog.

**Architecture:** Approach A (two independent CLIs). `cmd/enrich/` reuses existing seeder parsers to produce `enrich-output/<word>.json` context files and batch prompt files. `cmd/llm-import/` validates + maps LLM output JSON and inserts via existing `RefEntryBulkRepo`. Both tools are idempotent.

**Tech Stack:** Go 1.24, pgx/v5 (via existing bulk repo), cleanenv (config), `github.com/anthropics/anthropic-sdk-go` (API mode only), goose migrations.

---

## Task 1: DB Migration + Domain Model + BulkInsertSenses

**Files:**
- Create: `backend_v4/migrations/00015_add_notes_to_ref_senses.sql`
- Modify: `backend_v4/internal/domain/reference.go:25-37`
- Modify: `backend_v4/internal/adapter/postgres/refentry/repo_bulk.go:46-68`

**Step 1: Create the migration**

```sql
-- backend_v4/migrations/00015_add_notes_to_ref_senses.sql
-- +goose Up
ALTER TABLE ref_senses ADD COLUMN notes TEXT;

-- +goose Down
ALTER TABLE ref_senses DROP COLUMN notes;
```

**Step 2: Apply migration**

```bash
cd backend_v4 && make migrate-up
```

Expected: `OK   00015_add_notes_to_ref_senses.sql`

**Step 3: Add Notes field to RefSense**

In `backend_v4/internal/domain/reference.go`, add `Notes *string` to `RefSense`:

```go
type RefSense struct {
	ID           uuid.UUID
	RefEntryID   uuid.UUID
	Definition   string
	PartOfSpeech *PartOfSpeech
	CEFRLevel    *string
	Notes        *string   // ← add this line
	SourceSlug   string
	Position     int
	CreatedAt    time.Time

	Translations []RefTranslation
	Examples     []RefExample
}
```

**Step 4: Update BulkInsertSenses SQL**

In `backend_v4/internal/adapter/postgres/refentry/repo_bulk.go`, replace the `batch.Queue` call inside `BulkInsertSenses`:

Old:
```go
batch.Queue(
    `INSERT INTO ref_senses (id, ref_entry_id, definition, part_of_speech, cefr_level, source_slug, position, created_at)
     VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
     ON CONFLICT (id) DO NOTHING`,
    s.ID, s.RefEntryID, s.Definition, pos, s.CEFRLevel, s.SourceSlug, s.Position, s.CreatedAt,
)
```

New:
```go
batch.Queue(
    `INSERT INTO ref_senses (id, ref_entry_id, definition, part_of_speech, cefr_level, notes, source_slug, position, created_at)
     VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
     ON CONFLICT (id) DO NOTHING`,
    s.ID, s.RefEntryID, s.Definition, pos, s.CEFRLevel, s.Notes, s.SourceSlug, s.Position, s.CreatedAt,
)
```

**Step 5: Run tests to verify no regressions**

```bash
cd backend_v4 && make test
```

Expected: all tests pass.

**Step 6: Commit**

```bash
git add backend_v4/migrations/00015_add_notes_to_ref_senses.sql \
        backend_v4/internal/domain/reference.go \
        backend_v4/internal/adapter/postgres/refentry/repo_bulk.go
git commit -m "feat(domain): add notes field to ref_senses"
```

---

## Task 2: llm-import — JSON Model + Validator

**Files:**
- Create: `backend_v4/internal/llm_importer/model.go`
- Create: `backend_v4/internal/llm_importer/validator.go`
- Create: `backend_v4/internal/llm_importer/validator_test.go`

**Step 1: Create model.go (JSON input types)**

```go
// backend_v4/internal/llm_importer/model.go
package llm_importer

// LLMWordEntry is the top-level JSON document produced by the LLM.
// One file per word: llm-output/<word>.json
type LLMWordEntry struct {
	Word       string     `json:"word"`
	SourceSlug string     `json:"source_slug"` // e.g. "llm"
	Senses     []LLMSense `json:"senses"`
}

// LLMSense is one sense within an LLM word entry.
type LLMSense struct {
	POS          string       `json:"pos"`                    // must match domain.PartOfSpeech values (uppercase)
	Definition   string       `json:"definition"`
	CEFRLevel    string       `json:"cefr_level,omitempty"`   // A1..C2
	Notes        string       `json:"notes,omitempty"`
	Translations []string     `json:"translations"`
	Examples     []LLMExample `json:"examples,omitempty"`
}

// LLMExample is one usage example within a sense.
type LLMExample struct {
	Sentence    string `json:"sentence"`
	Translation string `json:"translation,omitempty"`
}
```

**Step 2: Write failing tests for validator**

```go
// backend_v4/internal/llm_importer/validator_test.go
package llm_importer

import "testing"

func TestValidate_valid(t *testing.T) {
	entry := LLMWordEntry{
		Word:       "abandon",
		SourceSlug: "llm",
		Senses: []LLMSense{
			{
				POS:          "VERB",
				Definition:   "To leave permanently.",
				CEFRLevel:    "B1",
				Notes:        "Often used with 'to'.",
				Translations: []string{"бросать"},
				Examples: []LLMExample{
					{Sentence: "She abandoned the car.", Translation: "Она бросила машину."},
				},
			},
		},
	}
	if err := Validate(entry); err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}
}

func TestValidate_missingWord(t *testing.T) {
	entry := LLMWordEntry{SourceSlug: "llm", Senses: []LLMSense{{POS: "NOUN", Definition: "x"}}}
	if err := Validate(entry); err == nil {
		t.Error("Validate() expected error for empty word")
	}
}

func TestValidate_noSenses(t *testing.T) {
	entry := LLMWordEntry{Word: "run", SourceSlug: "llm", Senses: nil}
	if err := Validate(entry); err == nil {
		t.Error("Validate() expected error for empty senses")
	}
}

func TestValidate_invalidPOS(t *testing.T) {
	entry := LLMWordEntry{
		Word: "run", SourceSlug: "llm",
		Senses: []LLMSense{{POS: "BANANA", Definition: "x"}},
	}
	if err := Validate(entry); err == nil {
		t.Error("Validate() expected error for invalid POS")
	}
}

func TestValidate_invalidCEFR(t *testing.T) {
	entry := LLMWordEntry{
		Word: "run", SourceSlug: "llm",
		Senses: []LLMSense{{POS: "VERB", Definition: "x", CEFRLevel: "Z9"}},
	}
	if err := Validate(entry); err == nil {
		t.Error("Validate() expected error for invalid CEFR level")
	}
}

func TestValidate_emptyDefinition(t *testing.T) {
	entry := LLMWordEntry{
		Word: "run", SourceSlug: "llm",
		Senses: []LLMSense{{POS: "VERB", Definition: ""}},
	}
	if err := Validate(entry); err == nil {
		t.Error("Validate() expected error for empty definition")
	}
}
```

**Step 3: Run to verify tests fail**

```bash
cd backend_v4 && go test ./internal/llm_importer/ -run TestValidate -v
```

Expected: compilation error (Validate not defined yet).

**Step 4: Implement validator**

```go
// backend_v4/internal/llm_importer/validator.go
package llm_importer

import (
	"fmt"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

var validCEFR = map[string]bool{
	"A1": true, "A2": true, "B1": true, "B2": true, "C1": true, "C2": true,
}

// Validate checks that an LLMWordEntry has all required fields with valid values.
func Validate(e LLMWordEntry) error {
	if e.Word == "" {
		return fmt.Errorf("word is empty")
	}
	if len(e.Senses) == 0 {
		return fmt.Errorf("word %q has no senses", e.Word)
	}
	for i, s := range e.Senses {
		if s.Definition == "" {
			return fmt.Errorf("sense %d of %q has empty definition", i, e.Word)
		}
		pos := domain.PartOfSpeech(s.POS)
		if !pos.IsValid() {
			return fmt.Errorf("sense %d of %q has invalid POS %q", i, e.Word, s.POS)
		}
		if s.CEFRLevel != "" && !validCEFR[s.CEFRLevel] {
			return fmt.Errorf("sense %d of %q has invalid CEFR level %q", i, e.Word, s.CEFRLevel)
		}
	}
	return nil
}
```

**Step 5: Run tests to verify they pass**

```bash
cd backend_v4 && go test ./internal/llm_importer/ -run TestValidate -v
```

Expected: all `TestValidate_*` PASS.

**Step 6: Commit**

```bash
git add backend_v4/internal/llm_importer/
git commit -m "feat(llm-import): add JSON model and validator"
```

---

## Task 3: llm-import — Mapper

**Files:**
- Create: `backend_v4/internal/llm_importer/mapper.go`
- Create: `backend_v4/internal/llm_importer/mapper_test.go`

**Step 1: Write failing test for mapper**

```go
// backend_v4/internal/llm_importer/mapper_test.go
package llm_importer

import (
	"testing"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

func TestMap_basicEntry(t *testing.T) {
	input := LLMWordEntry{
		Word:       "Abandon",
		SourceSlug: "llm",
		Senses: []LLMSense{
			{
				POS:          "VERB",
				Definition:   "To leave permanently.",
				CEFRLevel:    "B1",
				Notes:        "Often used with 'to'.",
				Translations: []string{"бросать", "покидать"},
				Examples: []LLMExample{
					{Sentence: "She abandoned the car.", Translation: "Она бросила машину."},
				},
			},
		},
	}

	result := Map(input)

	// Entry
	if result.Entry.Text != "Abandon" {
		t.Errorf("Entry.Text = %q, want %q", result.Entry.Text, "Abandon")
	}
	if result.Entry.TextNormalized != "abandon" {
		t.Errorf("Entry.TextNormalized = %q, want %q", result.Entry.TextNormalized, "abandon")
	}

	// Senses
	if len(result.Senses) != 1 {
		t.Fatalf("len(Senses) = %d, want 1", len(result.Senses))
	}
	s := result.Senses[0]
	if s.Definition != "To leave permanently." {
		t.Errorf("Sense.Definition = %q", s.Definition)
	}
	if *s.PartOfSpeech != domain.PartOfSpeechVerb {
		t.Errorf("Sense.PartOfSpeech = %q, want VERB", *s.PartOfSpeech)
	}
	if *s.CEFRLevel != "B1" {
		t.Errorf("Sense.CEFRLevel = %q, want B1", *s.CEFRLevel)
	}
	if *s.Notes != "Often used with 'to'." {
		t.Errorf("Sense.Notes = %q", *s.Notes)
	}
	if s.SourceSlug != "llm" {
		t.Errorf("Sense.SourceSlug = %q, want llm", s.SourceSlug)
	}

	// Translations
	if len(result.Translations) != 2 {
		t.Fatalf("len(Translations) = %d, want 2", len(result.Translations))
	}
	if result.Translations[0].Text != "бросать" {
		t.Errorf("Translations[0].Text = %q", result.Translations[0].Text)
	}

	// Examples
	if len(result.Examples) != 1 {
		t.Fatalf("len(Examples) = %d, want 1", len(result.Examples))
	}
	ex := result.Examples[0]
	if ex.Sentence != "She abandoned the car." {
		t.Errorf("Example.Sentence = %q", ex.Sentence)
	}
	if *ex.Translation != "Она бросила машину." {
		t.Errorf("Example.Translation = %q", *ex.Translation)
	}
}

func TestMap_emptyCEFRAndNotes(t *testing.T) {
	input := LLMWordEntry{
		Word: "run", SourceSlug: "llm",
		Senses: []LLMSense{{POS: "VERB", Definition: "To move fast."}},
	}
	result := Map(input)
	if result.Senses[0].CEFRLevel != nil {
		t.Error("CEFRLevel should be nil when empty string in JSON")
	}
	if result.Senses[0].Notes != nil {
		t.Error("Notes should be nil when empty string in JSON")
	}
}
```

**Step 2: Run to verify failure**

```bash
cd backend_v4 && go test ./internal/llm_importer/ -run TestMap -v
```

Expected: compilation error (Map not defined).

**Step 3: Implement mapper**

```go
// backend_v4/internal/llm_importer/mapper.go
package llm_importer

import (
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// MappedEntry holds domain types ready for bulk insert.
type MappedEntry struct {
	Entry        domain.RefEntry
	Senses       []domain.RefSense
	Translations []domain.RefTranslation
	Examples     []domain.RefExample
}

// Map converts an LLMWordEntry to domain types for insertion.
// Assumes the entry has been validated via Validate() first.
func Map(e LLMWordEntry) MappedEntry {
	now := time.Now()
	entryID := uuid.New()

	sourceSlug := e.SourceSlug
	if sourceSlug == "" {
		sourceSlug = "llm"
	}

	result := MappedEntry{
		Entry: domain.RefEntry{
			ID:             entryID,
			Text:           e.Word,
			TextNormalized: domain.NormalizeText(e.Word),
			IsCoreLexicon:  false,
			CreatedAt:      now,
		},
	}

	for i, s := range e.Senses {
		senseID := uuid.New()
		pos := domain.PartOfSpeech(s.POS)

		sense := domain.RefSense{
			ID:           senseID,
			RefEntryID:   entryID,
			Definition:   s.Definition,
			PartOfSpeech: &pos,
			SourceSlug:   sourceSlug,
			Position:     i,
			CreatedAt:    now,
		}
		if s.CEFRLevel != "" {
			level := s.CEFRLevel
			sense.CEFRLevel = &level
		}
		if s.Notes != "" {
			notes := s.Notes
			sense.Notes = &notes
		}
		result.Senses = append(result.Senses, sense)

		for j, tr := range s.Translations {
			result.Translations = append(result.Translations, domain.RefTranslation{
				ID:         uuid.New(),
				RefSenseID: senseID,
				Text:       tr,
				SourceSlug: sourceSlug,
				Position:   j,
			})
		}

		for j, ex := range s.Examples {
			var translation *string
			if ex.Translation != "" {
				t := ex.Translation
				translation = &t
			}
			result.Examples = append(result.Examples, domain.RefExample{
				ID:          uuid.New(),
				RefSenseID:  senseID,
				Sentence:    ex.Sentence,
				Translation: translation,
				SourceSlug:  sourceSlug,
				Position:    j,
			})
		}
	}

	return result
}
```

**Step 4: Run tests**

```bash
cd backend_v4 && go test ./internal/llm_importer/ -v
```

Expected: all tests PASS.

**Step 5: Commit**

```bash
git add backend_v4/internal/llm_importer/
git commit -m "feat(llm-import): add mapper"
```

---

## Task 4: llm-import — Config + Importer + main

**Files:**
- Create: `backend_v4/internal/llm_importer/config.go`
- Create: `backend_v4/internal/llm_importer/importer.go`
- Create: `backend_v4/cmd/llm-import/main.go`

**Step 1: Config**

```go
// backend_v4/internal/llm_importer/config.go
package llm_importer

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config holds llm-import settings.
type Config struct {
	LLMOutputDir string `yaml:"llm_output_dir" env:"LLM_IMPORT_OUTPUT_DIR" env-default:"./llm-output"`
	BatchSize    int    `yaml:"batch_size"      env:"LLM_IMPORT_BATCH_SIZE" env-default:"500"`
	DryRun       bool   `yaml:"dry_run"         env:"LLM_IMPORT_DRY_RUN"`
	SourceSlug   string `yaml:"source_slug"     env:"LLM_IMPORT_SOURCE_SLUG" env-default:"llm"`
}

// LoadConfig reads config from YAML file or environment variables.
func LoadConfig(path string) (*Config, error) {
	var cfg Config
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			if err := cleanenv.ReadConfig(path, &cfg); err != nil {
				return nil, fmt.Errorf("llm-import config: %w", err)
			}
			return &cfg, nil
		}
		return nil, fmt.Errorf("llm-import config: file %s not found", path)
	}
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("llm-import config: read env: %w", err)
	}
	return &cfg, nil
}
```

**Step 2: Importer**

```go
// backend_v4/internal/llm_importer/importer.go
package llm_importer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/seeder"
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
	entries, err := filepath.Glob(filepath.Join(cfg.LLMOutputDir, "*.json"))
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
		if n, err := repo.BulkInsertEntries(ctx, domainEntries); err != nil {
			return fmt.Errorf("bulk insert entries: %w", err)
		} else {
			result.Inserted += n
			result.Skipped += len(domainEntries) - n
		}
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

	for _, path := range entries {
		result.FilesProcessed++

		data, err := os.ReadFile(path)
		if err != nil {
			log.Error("read file", "path", path, "err", err)
			result.Errors++
			continue
		}

		var entry LLMWordEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			log.Error("unmarshal JSON", "path", path, "err", err)
			result.Errors++
			continue
		}

		if err := Validate(entry); err != nil {
			log.Error("invalid entry", "path", path, "err", err)
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
		"files", result.FilesProcessed,
		"inserted", result.Inserted,
		"skipped", result.Skipped,
		"errors", result.Errors,
	)
	return result, nil
}
```

**Step 3: main.go**

```go
// backend_v4/cmd/llm-import/main.go
package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/refentry"
	"github.com/heartmarshall/myenglish-backend/internal/app"
	"github.com/heartmarshall/myenglish-backend/internal/config"
	"github.com/heartmarshall/myenglish-backend/internal/llm_importer"
)

func main() {
	var (
		appConfigPath    = flag.String("config", "", "path to app config YAML")
		importConfigPath = flag.String("import-config", "", "path to llm-import config YAML")
	)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	appCfg, err := config.Load(*appConfigPath)
	if err != nil {
		log.Fatalf("load app config: %v", err)
	}

	importCfg, err := llm_importer.LoadConfig(*importConfigPath)
	if err != nil {
		log.Fatalf("load import config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	pool, err := postgres.NewPool(ctx, appCfg.DB.DSN)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	repo := refentry.NewRepo(pool)

	if importCfg.DryRun {
		logger.Info("dry-run mode: no DB writes")
	}

	result, err := llm_importer.Run(ctx, importCfg, repo, logger)
	if err != nil {
		log.Fatalf("import failed: %v", err)
	}

	_ = result
}
```

**Step 4: Check it compiles**

```bash
cd backend_v4 && go build ./cmd/llm-import/
```

Expected: builds without errors.

**Step 5: Commit**

```bash
git add backend_v4/internal/llm_importer/ backend_v4/cmd/llm-import/
git commit -m "feat(llm-import): add config, importer, and main"
```

---

## Task 5: enricher — Context Model

**Files:**
- Create: `backend_v4/internal/enricher/model.go`

**Step 1: Create model.go**

```go
// backend_v4/internal/enricher/model.go
package enricher

// EnrichContext is the per-word context file written to enrich-output/<word>.json.
// It is consumed by the LLM as input for generating LLMWordEntry JSON.
type EnrichContext struct {
	Word             string           `json:"word"`
	IPA              string           `json:"ipa,omitempty"`
	WiktionarySenses []WiktionarySense `json:"wiktionary_senses,omitempty"`
	Relations        Relations        `json:"relations,omitempty"`
}

// WiktionarySense holds one sense from Wiktionary for the LLM context.
type WiktionarySense struct {
	POS          string   `json:"pos"`
	Definition   string   `json:"definition"`
	Translations []string `json:"translations_ru,omitempty"`
}

// Relations holds semantic relations for a word from WordNet.
type Relations struct {
	Synonyms  []string `json:"synonyms,omitempty"`
	Antonyms  []string `json:"antonyms,omitempty"`
	Hypernyms []string `json:"hypernyms,omitempty"`
}
```

**Step 2: Commit**

```bash
git add backend_v4/internal/enricher/
git commit -m "feat(enrich): add context model"
```

---

## Task 6: enricher — Context Builder

**Files:**
- Create: `backend_v4/internal/enricher/context_builder.go`
- Create: `backend_v4/internal/enricher/context_builder_test.go`

**Step 1: Write failing tests**

```go
// backend_v4/internal/enricher/context_builder_test.go
package enricher

import (
	"testing"

	"github.com/heartmarshall/myenglish-backend/internal/seeder/cmu"
	"github.com/heartmarshall/myenglish-backend/internal/seeder/wiktionary"
	"github.com/heartmarshall/myenglish-backend/internal/seeder/wordnet"
)

func TestBuildContext_fullWord(t *testing.T) {
	wiktMap := map[string]*wiktionary.ParsedEntry{
		"run": {
			Word: "run",
			POSGroups: []wiktionary.POSGroup{
				{
					POS: "verb",
					Senses: []wiktionary.ParsedSense{
						{Glosses: []string{"To move fast."}, Translations: []string{"бежать"}},
					},
				},
			},
			Sounds: []wiktionary.Sound{{IPA: "/rʌn/", Region: "US"}},
		},
	}

	relMap := map[string]map[string][]string{
		"run": {
			"synonym":  {"sprint", "jog"},
			"antonym":  {"walk"},
			"hypernym": {"move"},
		},
	}

	cmuResult := cmu.ParseResult{
		Pronunciations: map[string][]cmu.IPATranscription{
			"run": {{IPA: "/rʌn/", VariantIndex: 0}},
		},
	}

	ctx := BuildContext("run", wiktMap, relMap, cmuResult)

	if ctx.Word != "run" {
		t.Errorf("Word = %q, want %q", ctx.Word, "run")
	}
	if ctx.IPA != "/rʌn/" {
		t.Errorf("IPA = %q, want %q", ctx.IPA, "/rʌn/")
	}
	if len(ctx.WiktionarySenses) != 1 {
		t.Fatalf("len(WiktionarySenses) = %d, want 1", len(ctx.WiktionarySenses))
	}
	if ctx.WiktionarySenses[0].Definition != "To move fast." {
		t.Errorf("Sense.Definition = %q", ctx.WiktionarySenses[0].Definition)
	}
	if len(ctx.WiktionarySenses[0].Translations) != 1 || ctx.WiktionarySenses[0].Translations[0] != "бежать" {
		t.Errorf("Sense.Translations = %v", ctx.WiktionarySenses[0].Translations)
	}
	if len(ctx.Relations.Synonyms) != 2 {
		t.Errorf("Synonyms = %v", ctx.Relations.Synonyms)
	}
	if len(ctx.Relations.Antonyms) != 1 {
		t.Errorf("Antonyms = %v", ctx.Relations.Antonyms)
	}
}

func TestBuildContext_unknownWord(t *testing.T) {
	ctx := BuildContext("xyzzy",
		map[string]*wiktionary.ParsedEntry{},
		map[string]map[string][]string{},
		cmu.ParseResult{Pronunciations: map[string][]cmu.IPATranscription{}},
	)
	if ctx.Word != "xyzzy" {
		t.Errorf("Word = %q", ctx.Word)
	}
	if len(ctx.WiktionarySenses) != 0 {
		t.Errorf("expected no senses for unknown word")
	}
	if ctx.IPA != "" {
		t.Errorf("expected empty IPA for unknown word")
	}
}
```

**Step 2: Run to verify failure**

```bash
cd backend_v4 && go test ./internal/enricher/ -run TestBuildContext -v
```

Expected: compilation error (BuildContext not defined).

**Step 3: Implement context_builder.go**

```go
// backend_v4/internal/enricher/context_builder.go
package enricher

import (
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/seeder/cmu"
	"github.com/heartmarshall/myenglish-backend/internal/seeder/wiktionary"
)

// BuildContext assembles an EnrichContext for one word from pre-loaded dataset maps.
// All dataset maps are keyed by normalized (lowercase) word text.
//
//   - wiktMap:   normalized word → *wiktionary.ParsedEntry
//   - relMap:    normalized word → relation_type → []target_words
//   - cmuResult: cmu.ParseResult with Pronunciations map
func BuildContext(
	word string,
	wiktMap map[string]*wiktionary.ParsedEntry,
	relMap map[string]map[string][]string,
	cmuResult cmu.ParseResult,
) EnrichContext {
	normalized := domain.NormalizeText(word)

	ctx := EnrichContext{Word: word}

	// IPA: prefer CMU, fall back to Wiktionary Sound.
	if ipas, ok := cmuResult.Pronunciations[normalized]; ok && len(ipas) > 0 {
		ctx.IPA = ipas[0].IPA
	} else if entry, ok := wiktMap[normalized]; ok {
		for _, s := range entry.Sounds {
			if s.IPA != "" {
				ctx.IPA = s.IPA
				break
			}
		}
	}

	// Wiktionary senses.
	if entry, ok := wiktMap[normalized]; ok {
		for _, pg := range entry.POSGroups {
			for _, s := range pg.Senses {
				if len(s.Glosses) == 0 {
					continue
				}
				ctx.WiktionarySenses = append(ctx.WiktionarySenses, WiktionarySense{
					POS:          pg.POS,
					Definition:   wiktionary.StripMarkup(s.Glosses[0]),
					Translations: s.Translations,
				})
			}
		}
	}

	// Relations from WordNet.
	if rels, ok := relMap[normalized]; ok {
		ctx.Relations = Relations{
			Synonyms:  rels["synonym"],
			Antonyms:  rels["antonym"],
			Hypernyms: rels["hypernym"],
		}
	}

	return ctx
}
```

**Step 4: Run tests**

```bash
cd backend_v4 && go test ./internal/enricher/ -v
```

Expected: all tests PASS.

**Step 5: Commit**

```bash
git add backend_v4/internal/enricher/
git commit -m "feat(enrich): add context builder"
```

---

## Task 7: enricher — Config + Pipeline (manual mode) + main

**Files:**
- Create: `backend_v4/internal/enricher/config.go`
- Create: `backend_v4/internal/enricher/pipeline.go`
- Create: `backend_v4/cmd/enrich/main.go`
- Create: `backend_v4/enrich.yaml`

**Step 1: Config**

```go
// backend_v4/internal/enricher/config.go
package enricher

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config holds enricher pipeline settings.
type Config struct {
	WordListPath    string `yaml:"word_list_path"    env:"ENRICH_WORD_LIST_PATH"`
	WiktionaryPath  string `yaml:"wiktionary_path"   env:"ENRICH_WIKTIONARY_PATH"`
	WordNetPath     string `yaml:"wordnet_path"      env:"ENRICH_WORDNET_PATH"`
	CMUPath         string `yaml:"cmu_path"          env:"ENRICH_CMU_PATH"`
	EnrichOutputDir string `yaml:"enrich_output_dir" env:"ENRICH_OUTPUT_DIR"  env-default:"./enrich-output"`
	LLMOutputDir    string `yaml:"llm_output_dir"    env:"ENRICH_LLM_OUTPUT_DIR" env-default:"./llm-output"`
	Mode            string `yaml:"mode"              env:"ENRICH_MODE"         env-default:"manual"`
	BatchSize       int    `yaml:"batch_size"        env:"ENRICH_BATCH_SIZE"   env-default:"50"`
	LLMAPIKey       string `yaml:"llm_api_key"       env:"ENRICH_LLM_API_KEY"`
	LLMModel        string `yaml:"llm_model"         env:"ENRICH_LLM_MODEL"    env-default:"claude-opus-4-6"`
}

// LoadConfig reads enricher config from YAML or environment variables.
func LoadConfig(path string) (*Config, error) {
	var cfg Config
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			if err := cleanenv.ReadConfig(path, &cfg); err != nil {
				return nil, fmt.Errorf("enrich config: %w", err)
			}
			return &cfg, nil
		}
		return nil, fmt.Errorf("enrich config: file %s not found", path)
	}
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("enrich config: read env: %w", err)
	}
	return &cfg, nil
}
```

**Step 2: Pipeline (manual mode)**

```go
// backend_v4/internal/enricher/pipeline.go
package enricher

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/seeder/cmu"
	"github.com/heartmarshall/myenglish-backend/internal/seeder/wordnet"
	"github.com/heartmarshall/myenglish-backend/internal/seeder/wiktionary"
)

// PipelineResult holds enrichment statistics.
type PipelineResult struct {
	TotalWords     int
	Written        int
	Skipped        int // already existed in enrich-output/
	BatchFiles     int
}

// Run loads datasets and generates enrich-output/<word>.json for all words in the word list.
// In manual mode it also writes batch prompt files.
// In api mode it additionally calls the LLM API (see llm_client.go).
func Run(cfg *Config, log *slog.Logger) (PipelineResult, error) {
	var result PipelineResult

	// 1. Read word list.
	words, err := readWordList(cfg.WordListPath)
	if err != nil {
		return result, fmt.Errorf("read word list: %w", err)
	}
	result.TotalWords = len(words)
	log.Info("word list loaded", "count", len(words))

	wordSet := make(map[string]bool, len(words))
	for _, w := range words {
		wordSet[domain.NormalizeText(w)] = true
	}

	// 2. Load datasets.
	log.Info("parsing wiktionary...")
	wiktEntries, _, err := wiktionary.Parse(cfg.WiktionaryPath, wordSet, len(wordSet)+10000)
	if err != nil {
		return result, fmt.Errorf("parse wiktionary: %w", err)
	}
	wiktMap := make(map[string]*wiktionary.ParsedEntry, len(wiktEntries))
	for i := range wiktEntries {
		wiktMap[domain.NormalizeText(wiktEntries[i].Word)] = &wiktEntries[i]
	}
	log.Info("wiktionary parsed", "entries", len(wiktMap))

	log.Info("parsing wordnet...")
	wnResult, err := wordnet.Parse(cfg.WordNetPath, wordSet)
	if err != nil {
		return result, fmt.Errorf("parse wordnet: %w", err)
	}
	relMap := buildRelationMap(wnResult)
	log.Info("wordnet parsed", "relations", len(wnResult.Relations))

	log.Info("parsing cmu...")
	cmuResult, err := cmu.Parse(cfg.CMUPath)
	if err != nil {
		return result, fmt.Errorf("parse cmu: %w", err)
	}
	log.Info("cmu parsed", "words", cmuResult.Stats.UniqueWords)

	// 3. Ensure output dirs exist.
	if err := os.MkdirAll(cfg.EnrichOutputDir, 0755); err != nil {
		return result, fmt.Errorf("create enrich-output dir: %w", err)
	}

	// 4. Build context files + batch prompts.
	var batch []string
	batchNum := 1

	for _, word := range words {
		outPath := filepath.Join(cfg.EnrichOutputDir, domain.NormalizeText(word)+".json")

		// Resume: skip if already generated.
		if _, err := os.Stat(outPath); err == nil {
			result.Skipped++
			continue
		}

		ctx := BuildContext(word, wiktMap, relMap, cmuResult)

		data, err := json.MarshalIndent(ctx, "", "  ")
		if err != nil {
			log.Error("marshal context", "word", word, "err", err)
			continue
		}
		if err := os.WriteFile(outPath, data, 0644); err != nil {
			log.Error("write context file", "word", word, "err", err)
			continue
		}
		result.Written++

		batch = append(batch, word)
		if len(batch) >= cfg.BatchSize {
			if err := writeBatchPrompt(cfg.EnrichOutputDir, batchNum, batch); err != nil {
				log.Warn("write batch prompt", "batch", batchNum, "err", err)
			} else {
				result.BatchFiles++
			}
			batch = batch[:0]
			batchNum++
		}
	}

	// Flush remaining batch.
	if len(batch) > 0 {
		if err := writeBatchPrompt(cfg.EnrichOutputDir, batchNum, batch); err != nil {
			log.Warn("write batch prompt", "batch", batchNum, "err", err)
		} else {
			result.BatchFiles++
		}
	}

	log.Info("enrichment complete",
		"total", result.TotalWords,
		"written", result.Written,
		"skipped", result.Skipped,
		"batch_files", result.BatchFiles,
	)
	return result, nil
}

func readWordList(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var words []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word != "" && !strings.HasPrefix(word, "#") {
			words = append(words, word)
		}
	}
	return words, scanner.Err()
}

// buildRelationMap converts flat []wordnet.Relation into word → type → []targets.
func buildRelationMap(result wordnet.ParseResult) map[string]map[string][]string {
	m := make(map[string]map[string][]string)
	for _, r := range result.Relations {
		if m[r.SourceWord] == nil {
			m[r.SourceWord] = make(map[string][]string)
		}
		m[r.SourceWord][r.RelationType] = append(m[r.SourceWord][r.RelationType], r.TargetWord)
	}
	return m
}

// writeBatchPrompt writes a text file with the list of words in a batch.
func writeBatchPrompt(dir string, batchNum int, words []string) error {
	path := filepath.Join(dir, fmt.Sprintf("batch_%04d_words.txt", batchNum))
	content := strings.Join(words, "\n")
	return os.WriteFile(path, []byte(content), 0644)
}
```

**Step 3: main.go**

```go
// backend_v4/cmd/enrich/main.go
package main

import (
	"flag"
	"log"
	"log/slog"
	"os"

	"github.com/heartmarshall/myenglish-backend/internal/enricher"
)

func main() {
	configPath := flag.String("enrich-config", "", "path to enrich YAML config")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := enricher.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	result, err := enricher.Run(cfg, logger)
	if err != nil {
		log.Fatalf("enrichment failed: %v", err)
	}

	_ = result

	if cfg.Mode == "api" {
		logger.Info("api mode: LLM API calls not yet implemented, run with --mode=manual")
	}
}
```

**Step 4: Sample enrich.yaml**

```yaml
# backend_v4/enrich.yaml
word_list_path: ../datasets/common_words.txt
wiktionary_path: ../datasets/kaikki.org-dictionary-English.jsonl
wordnet_path: ../datasets/english-wordnet-2025
cmu_path: ../datasets/cmudict.dict
enrich_output_dir: ./enrich-output
llm_output_dir: ./llm-output
mode: manual
batch_size: 50
llm_model: "claude-opus-4-6"
```

**Step 5: Check it compiles**

```bash
cd backend_v4 && go build ./cmd/enrich/
```

Expected: builds without errors.

**Step 6: Run unit tests**

```bash
cd backend_v4 && make test
```

Expected: all tests pass.

**Step 7: Commit**

```bash
git add backend_v4/internal/enricher/ backend_v4/cmd/enrich/ backend_v4/enrich.yaml
git commit -m "feat(enrich): add pipeline, config, and main (manual mode)"
```

---

## Task 8: enricher — API Mode (LLM Client)

**Files:**
- Create: `backend_v4/internal/enricher/llm_client.go`
- Modify: `backend_v4/internal/enricher/pipeline.go` (add API mode call)

**Step 1: Add Anthropic SDK dependency**

```bash
cd backend_v4 && go get github.com/anthropics/anthropic-sdk-go
```

**Step 2: Implement llm_client.go**

```go
// backend_v4/internal/enricher/llm_client.go
package enricher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// callLLM sends one EnrichContext to Claude and saves the LLM output JSON.
// Output is saved to llmOutputDir/<normalized_word>.json.
// If the file already exists, it is skipped (resume support).
func callLLM(ctx context.Context, cfg *Config, enrichCtx EnrichContext, log *slog.Logger) error {
	normalized := domain.NormalizeText(enrichCtx.Word)
	outPath := filepath.Join(cfg.LLMOutputDir, normalized+".json")

	// Resume: skip if already done.
	if _, err := os.Stat(outPath); err == nil {
		return nil
	}

	if err := os.MkdirAll(cfg.LLMOutputDir, 0755); err != nil {
		return fmt.Errorf("create llm-output dir: %w", err)
	}

	contextJSON, err := json.MarshalIndent(enrichCtx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal context: %w", err)
	}

	prompt := buildPrompt(enrichCtx.Word, string(contextJSON))

	client := anthropic.NewClient(option.WithAPIKey(cfg.LLMAPIKey))

	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.F(anthropic.Model(cfg.LLMModel)),
		MaxTokens: anthropic.F(int64(2048)),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		}),
	})
	if err != nil {
		return fmt.Errorf("llm api call for %q: %w", enrichCtx.Word, err)
	}

	if len(msg.Content) == 0 {
		return fmt.Errorf("empty response for %q", enrichCtx.Word)
	}

	responseText := msg.Content[0].Text

	// Extract JSON from the response (between first { and last }).
	jsonStr, err := extractJSON(responseText)
	if err != nil {
		return fmt.Errorf("extract json from response for %q: %w", enrichCtx.Word, err)
	}

	if err := os.WriteFile(outPath, []byte(jsonStr), 0644); err != nil {
		return fmt.Errorf("write llm output for %q: %w", enrichCtx.Word, err)
	}

	log.Info("llm output saved", "word", enrichCtx.Word, "path", outPath)
	return nil
}

// buildPrompt creates the LLM prompt for a single word.
func buildPrompt(word, contextJSON string) string {
	return fmt.Sprintf(`You are a professional English-Russian dictionary editor.

Given the word "%s" and its context data from reference datasets, produce an improved dictionary entry in JSON format.

Context data:
%s

Output ONLY a valid JSON object matching this exact schema:
{
  "word": "<word>",
  "source_slug": "llm",
  "senses": [
    {
      "pos": "<NOUN|VERB|ADJECTIVE|ADVERB|...>",
      "definition": "<clear English definition suitable for B1+ learners>",
      "cefr_level": "<A1|A2|B1|B2|C1|C2 or empty>",
      "notes": "<learning note in Russian: usage tips, collocations, common mistakes>",
      "translations": ["<Russian translation 1>", "<Russian translation 2>"],
      "examples": [
        {"sentence": "<English example>", "translation": "<Russian translation>"}
      ]
    }
  ]
}

Rules:
- Improve definitions to be clearer and more useful for language learners
- Provide 2-4 high-quality Russian translations per sense
- Write notes in Russian, focusing on practical usage
- Generate 1-3 natural example sentences with Russian translations
- Use uppercase POS values matching: NOUN, VERB, ADJECTIVE, ADVERB, PRONOUN, PREPOSITION, CONJUNCTION, INTERJECTION, PHRASE, IDIOM, OTHER
- Output ONLY the JSON, no markdown, no explanations`, word, contextJSON)
}

// extractJSON finds the first complete JSON object in a string.
func extractJSON(s string) (string, error) {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || end <= start {
		return "", fmt.Errorf("no JSON object found in response")
	}
	return s[start : end+1], nil
}
```

**Step 3: Wire API mode into pipeline.go**

In `backend_v4/internal/enricher/pipeline.go`, replace the `if cfg.Mode == "api"` stub in `main.go` with a real call inside the word loop. After the context file is written, add:

```go
// In the word loop in Run(), after os.WriteFile(outPath, data, 0644):
if cfg.Mode == "api" {
    if err := callLLM(context.Background(), cfg, ctx, log); err != nil {
        log.Error("llm api call", "word", word, "err", err)
    }
}
```

Also add `"context"` to imports in `pipeline.go`.

**Step 4: Build to verify**

```bash
cd backend_v4 && go build ./cmd/enrich/
```

Expected: builds without errors.

**Step 5: Run all tests**

```bash
cd backend_v4 && make test
```

Expected: all tests pass.

**Step 6: Commit**

```bash
git add backend_v4/internal/enricher/ backend_v4/go.mod backend_v4/go.sum
git commit -m "feat(enrich): add LLM API client (api mode)"
```

---

## Summary of New Files

```
backend_v4/
├── migrations/
│   └── 00015_add_notes_to_ref_senses.sql
├── cmd/
│   ├── enrich/main.go
│   └── llm-import/main.go
├── internal/
│   ├── enricher/
│   │   ├── model.go
│   │   ├── config.go
│   │   ├── context_builder.go
│   │   ├── context_builder_test.go
│   │   ├── pipeline.go
│   │   └── llm_client.go
│   └── llm_importer/
│       ├── model.go
│       ├── validator.go
│       ├── validator_test.go
│       ├── mapper.go
│       ├── mapper_test.go
│       ├── config.go
│       └── importer.go
└── enrich.yaml
```

## Running the Full Pipeline

```bash
# Step 1: Generate enrichment context files (manual mode)
cd backend_v4
go run ./cmd/enrich/ --enrich-config=enrich.yaml
# Output: enrich-output/<word>.json + enrich-output/batch_NNN_words.txt

# Step 2: Feed batch prompts to LLM, save responses to llm-output/<word>.json

# Step 3: Import LLM output into DB
go run ./cmd/llm-import/ --import-config=llm-import.yaml

# Or fully automated (API mode):
ENRICH_LLM_API_KEY=sk-... go run ./cmd/enrich/ --enrich-config=enrich.yaml
# (set mode: api in enrich.yaml)
```
