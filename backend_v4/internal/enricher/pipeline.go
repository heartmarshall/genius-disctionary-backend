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
	TotalWords int
	Written    int
	Skipped    int // already existed in enrich-output/
	BatchFiles int
}

// Run loads datasets and generates enrich-output/<word>.json for all words in the word list.
// In manual mode it also writes batch word list files.
// In api mode it additionally calls the LLM API (see llm_client.go).
func Run(cfg *Config, log *slog.Logger) (PipelineResult, error) {
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

	// 4. Build context files + batch word lists.
	var batch []string
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

		batch = append(batch, word)
		if len(batch) >= cfg.BatchSize {
			if err := writeBatchWordList(cfg.EnrichOutputDir, batchNum, batch); err != nil {
				log.Warn("write batch word list", slog.Int("batch", batchNum), slog.String("error", err.Error()))
			} else {
				result.BatchFiles++
			}
			batch = batch[:0]
			batchNum++
		}
	}

	// Flush remaining batch.
	if len(batch) > 0 {
		if err := writeBatchWordList(cfg.EnrichOutputDir, batchNum, batch); err != nil {
			log.Warn("write batch word list", slog.Int("batch", batchNum), slog.String("error", err.Error()))
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

// writeBatchWordList writes a text file listing the words in one batch.
func writeBatchWordList(dir string, batchNum int, words []string) error {
	path := filepath.Join(dir, fmt.Sprintf("batch_%04d_words.txt", batchNum))
	return os.WriteFile(path, []byte(strings.Join(words, "\n")), 0644)
}
