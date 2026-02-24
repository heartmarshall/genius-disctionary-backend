package enricher

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/seeder/cmu"
	"github.com/heartmarshall/myenglish-backend/internal/seeder/wordnet"
	"github.com/heartmarshall/myenglish-backend/internal/seeder/wiktionary"
)

// PipelineResult holds enrichment statistics.
type PipelineResult struct {
	TotalWords int
	Written    int
	Skipped    int // already existed in enrich-output/
	BatchFiles int
}

// Run loads datasets and generates enrich-output/<word>.json for all words in the word list.
// In manual mode it also writes batch word list files.
// In api mode it additionally calls the LLM API (see llm_client.go).
func Run(ctx context.Context, cfg *Config, log *slog.Logger) (PipelineResult, error) {
	var result PipelineResult

	// 1. Read word list.
	words, err := readWordList(cfg.WordListPath)
	if err != nil {
		return result, fmt.Errorf("read word list: %w", err)
	}
	result.TotalWords = len(words)
	log.Info("word list loaded", slog.Int("count", len(words)))

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
	log.Info("wiktionary parsed", slog.Int("entries", len(wiktMap)))

	log.Info("parsing wordnet...")
	wnResult, err := wordnet.Parse(cfg.WordNetPath, wordSet)
	if err != nil {
		return result, fmt.Errorf("parse wordnet: %w", err)
	}
	relMap := buildRelationMap(wnResult)
	log.Info("wordnet parsed", slog.Int("relations", len(wnResult.Relations)))

	log.Info("parsing cmu...")
	cmuResult, err := cmu.Parse(cfg.CMUPath)
	if err != nil {
		return result, fmt.Errorf("parse cmu: %w", err)
	}
	log.Info("cmu parsed", slog.Int("words", cmuResult.Stats.UniqueWords))

	// 3. Ensure output dir exists.
	if err := os.MkdirAll(cfg.EnrichOutputDir, 0755); err != nil {
		return result, fmt.Errorf("create enrich-output dir: %w", err)
	}

	var llmClient anthropic.Client
	if cfg.Mode == "api" {
		llmClient = anthropic.NewClient(option.WithAPIKey(cfg.LLMAPIKey))
	}

	// 4. Build context files + batch prompts.
	var batch []EnrichContext
	batchNum := 1

	for _, word := range words {
		outPath := filepath.Join(cfg.EnrichOutputDir, domain.NormalizeText(word)+".json")

		// Resume: skip if already generated.
		if _, err := os.Stat(outPath); err == nil {
			result.Skipped++
			continue
		}

		enrichCtx := BuildContext(word, wiktMap, relMap, cmuResult)

		data, err := json.MarshalIndent(enrichCtx, "", "  ")
		if err != nil {
			log.Error("marshal context", slog.String("word", word), slog.String("error", err.Error()))
			continue
		}
		if err := os.WriteFile(outPath, data, 0644); err != nil {
			log.Error("write context file", slog.String("word", word), slog.String("error", err.Error()))
			continue
		}
		result.Written++

		if cfg.Mode == "api" {
			if err := callLLM(ctx, llmClient, cfg, enrichCtx, log); err != nil {
				log.Error("llm api call", slog.String("word", word), slog.String("error", err.Error()))
			}
		}

		batch = append(batch, enrichCtx)
		if len(batch) >= cfg.BatchSize {
			if err := writeBatchPrompt(cfg.EnrichOutputDir, batchNum, batch); err != nil {
				log.Warn("write batch prompt", slog.Int("batch", batchNum), slog.String("error", err.Error()))
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
			log.Warn("write batch prompt", slog.Int("batch", batchNum), slog.String("error", err.Error()))
		} else {
			result.BatchFiles++
		}
	}

	log.Info("enrichment complete",
		slog.Int("total", result.TotalWords),
		slog.Int("written", result.Written),
		slog.Int("skipped", result.Skipped),
		slog.Int("batch_files", result.BatchFiles),
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

// buildRelationMap converts flat []wordnet.Relation into word -> type -> []targets.
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

// writeBatchPrompt writes a ready-to-paste LLM prompt for all words in one batch.
// Output file: enrich-output/batch_NNNN_prompt.txt
func writeBatchPrompt(dir string, batchNum int, contexts []EnrichContext) error {
	path := filepath.Join(dir, fmt.Sprintf("batch_%04d_prompt.txt", batchNum))

	var b strings.Builder

	fmt.Fprintf(&b, `You are a professional English-Russian dictionary editor.
Below are %d English words with their context from reference datasets (IPA, existing definitions, Russian translations, semantic relations).

For EACH word produce an improved dictionary entry in JSON format.
Separate consecutive JSON objects with a line containing only "---".
Output NOTHING except the JSON objects and the "---" separators — no markdown, no explanations.

Each JSON must match this exact schema:
{
  "word": "<word>",
  "source_slug": "llm",
  "senses": [
    {
      "pos": "<NOUN|VERB|ADJECTIVE|ADVERB|PRONOUN|PREPOSITION|CONJUNCTION|INTERJECTION|PHRASE|IDIOM|OTHER>",
      "definition": "<clear English definition for B1+ learners>",
      "cefr_level": "<A1|A2|B1|B2|C1|C2>",
      "notes": "<learning tip in Russian: collocations, register, common mistakes>",
      "translations": ["<Russian 1>", "<Russian 2>"],
      "examples": [
        {"sentence": "<English example>", "translation": "<Russian translation>"}
      ]
    }
  ]
}

Rules:
- Rewrite definitions to be clearer and more useful for learners
- Provide 2-4 Russian translations per sense
- Write notes in Russian
- Generate 1-3 natural example sentences with Russian translations
- Preserve the word order — output exactly %d JSON objects in the same order

`, len(contexts), len(contexts))

	for i, ctx := range contexts {
		ctxJSON, err := json.MarshalIndent(ctx, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal context for %q: %w", ctx.Word, err)
		}
		fmt.Fprintf(&b, "=== WORD %d: %s ===\n%s\n", i+1, ctx.Word, ctxJSON)
		if i < len(contexts)-1 {
			b.WriteString("\n")
		}
	}

	return os.WriteFile(path, []byte(b.String()), 0644)
}
