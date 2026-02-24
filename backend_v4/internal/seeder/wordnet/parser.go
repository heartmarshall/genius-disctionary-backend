// Package wordnet parses Open English WordNet (OEWN 2025) JSON files into word relations.
// Pure function: directory path in, domain structs out. No database dependencies.
//
// Expected directory structure (as distributed by https://github.com/globalwordnet/english-wordnet):
//
//	entries-a.json … entries-z.json   — lemma entries keyed by word
//	noun.*.json, verb.*.json, …       — synsets keyed by synset ID
package wordnet

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

const sourceSlug = "wordnet"

// Relation represents a semantic relationship between two words.
type Relation struct {
	SourceWord   string
	TargetWord   string
	RelationType string // synonym, hypernym, antonym, derived
}

// ParseResult holds parsed WordNet relations.
type ParseResult struct {
	Relations []Relation
	Stats     Stats
}

// Stats holds parser statistics for logging.
type Stats struct {
	TotalSynsets    int
	TotalEntries    int
	TotalRelations  int
	FilteredByKnown int
	SelfReferential int
	Duplicates      int
}

// OEWN 2025 JSON deserialization types.

// oewnEntryFile represents an entries-*.json file: {"word": {"pos": {...}}}.
type oewnEntryFile map[string]map[string]json.RawMessage

// oewnPOSEntry holds senses for a single POS of a word.
type oewnPOSEntry struct {
	Sense []oewnSense `json:"sense"`
}

// oewnSense holds a single sense linking a word to a synset.
type oewnSense struct {
	ID         string   `json:"id"`
	Synset     string   `json:"synset"`
	Antonym    []string `json:"antonym"`
	Derivation []string `json:"derivation"`
}

// oewnSynset holds a single synset from a {pos}.{category}.json file.
type oewnSynset struct {
	Members  []string `json:"members"`
	Hypernym []string `json:"hypernym"`
}

// dedupKey is used for deduplicating relations.
type dedupKey struct {
	source  string
	target  string
	relType string
}

// Parse reads an OEWN 2025 JSON directory and extracts relations for known words.
func Parse(dirPath string, knownWords map[string]bool) (ParseResult, error) {
	if len(knownWords) == 0 {
		return ParseResult{}, nil
	}

	// Validate directory exists.
	info, err := os.Stat(dirPath)
	if err != nil {
		return ParseResult{}, fmt.Errorf("open directory: %w", err)
	}
	if !info.IsDir() {
		return ParseResult{}, fmt.Errorf("%s is not a directory", dirPath)
	}

	// Step 1: Read all entry files → build senseID→word mapping.
	entryFiles, err := filepath.Glob(filepath.Join(dirPath, "entries-*.json"))
	if err != nil {
		return ParseResult{}, fmt.Errorf("glob entry files: %w", err)
	}

	senseToWord := make(map[string]string)
	var stats Stats

	for _, path := range entryFiles {
		entries, err := readEntryFile(path)
		if err != nil {
			return ParseResult{}, fmt.Errorf("read %s: %w", filepath.Base(path), err)
		}

		for word, posMap := range entries {
			stats.TotalEntries++
			normalized := domain.NormalizeText(word)

			for _, raw := range posMap {
				var posEntry oewnPOSEntry
				if err := json.Unmarshal(raw, &posEntry); err != nil {
					continue
				}
				for _, sense := range posEntry.Sense {
					senseToWord[sense.ID] = normalized
				}
			}
		}
	}

	// Step 2: Read all synset files.
	synsetFiles, err := globSynsetFiles(dirPath)
	if err != nil {
		return ParseResult{}, fmt.Errorf("glob synset files: %w", err)
	}

	allSynsets := make(map[string]oewnSynset)

	for _, path := range synsetFiles {
		synsets, err := readSynsetFile(path)
		if err != nil {
			return ParseResult{}, fmt.Errorf("read %s: %w", filepath.Base(path), err)
		}

		for id, synset := range synsets {
			stats.TotalSynsets++
			allSynsets[id] = synset
		}
	}

	// Step 3: Extract relations.
	var result ParseResult
	seen := make(map[dedupKey]bool)

	addRelation := func(source, target, relType string) {
		if source == target {
			result.Stats.SelfReferential++
			return
		}
		if !knownWords[source] || !knownWords[target] {
			result.Stats.FilteredByKnown++
			return
		}
		source, target = applyDirectionality(source, target, relType)
		key := dedupKey{source: source, target: target, relType: relType}
		if seen[key] {
			result.Stats.Duplicates++
			return
		}
		seen[key] = true
		result.Relations = append(result.Relations, Relation{
			SourceWord:   source,
			TargetWord:   target,
			RelationType: relType,
		})
	}

	// Step 3a: Synonyms — words sharing the same synset (from members).
	for _, synset := range allSynsets {
		if len(synset.Members) < 2 {
			continue
		}
		normalized := make([]string, 0, len(synset.Members))
		for _, m := range synset.Members {
			normalized = append(normalized, domain.NormalizeText(m))
		}
		for i := 0; i < len(normalized); i++ {
			for j := i + 1; j < len(normalized); j++ {
				addRelation(normalized[i], normalized[j], "synonym")
			}
		}
	}

	// Step 3b: Antonyms and derived forms from sense-level relations.
	for _, path := range entryFiles {
		entries, err := readEntryFile(path)
		if err != nil {
			continue // already read successfully above
		}

		for word, posMap := range entries {
			normalized := domain.NormalizeText(word)
			for _, raw := range posMap {
				var posEntry oewnPOSEntry
				if err := json.Unmarshal(raw, &posEntry); err != nil {
					continue
				}
				for _, sense := range posEntry.Sense {
					for _, targetSenseID := range sense.Antonym {
						if targetWord, ok := senseToWord[targetSenseID]; ok {
							addRelation(normalized, targetWord, "antonym")
						}
					}
					for _, targetSenseID := range sense.Derivation {
						if targetWord, ok := senseToWord[targetSenseID]; ok {
							addRelation(normalized, targetWord, "derived")
						}
					}
				}
			}
		}
	}

	// Step 3c: Hypernyms from synset-level relations.
	// Build synsetID → []normalizedWord for resolving hypernym targets.
	synsetToWords := make(map[string][]string)
	for _, path := range entryFiles {
		entries, err := readEntryFile(path)
		if err != nil {
			continue
		}
		for word, posMap := range entries {
			normalized := domain.NormalizeText(word)
			for _, raw := range posMap {
				var posEntry oewnPOSEntry
				if err := json.Unmarshal(raw, &posEntry); err != nil {
					continue
				}
				for _, sense := range posEntry.Sense {
					synsetToWords[sense.Synset] = appendUnique(synsetToWords[sense.Synset], normalized)
				}
			}
		}
	}

	for synsetID, synset := range allSynsets {
		for _, hyperSynsetID := range synset.Hypernym {
			sourceWords := synsetToWords[synsetID]
			targetWords := synsetToWords[hyperSynsetID]
			for _, sw := range sourceWords {
				for _, tw := range targetWords {
					addRelation(sw, tw, "hypernym")
				}
			}
		}
	}

	result.Stats.TotalEntries = stats.TotalEntries
	result.Stats.TotalSynsets = stats.TotalSynsets
	result.Stats.TotalRelations = len(result.Relations)
	return result, nil
}

// applyDirectionality applies the directionality conventions for relation types.
func applyDirectionality(source, target, relType string) (string, string) {
	switch relType {
	case "synonym", "antonym", "derived":
		if source > target {
			return target, source
		}
		return source, target
	case "hypernym":
		return source, target
	default:
		return source, target
	}
}

// ToDomainRelations converts parsed relations to domain RefWordRelation records.
func (r ParseResult) ToDomainRelations(entryIDMap map[string]uuid.UUID) []domain.RefWordRelation {
	if len(entryIDMap) == 0 {
		return nil
	}

	var result []domain.RefWordRelation
	for _, rel := range r.Relations {
		sourceID, sourceOK := entryIDMap[rel.SourceWord]
		targetID, targetOK := entryIDMap[rel.TargetWord]
		if !sourceOK || !targetOK {
			continue
		}

		result = append(result, domain.RefWordRelation{
			ID:            uuid.New(),
			SourceEntryID: sourceID,
			TargetEntryID: targetID,
			RelationType:  rel.RelationType,
			SourceSlug:    sourceSlug,
		})
	}

	return result
}

// readEntryFile reads a single entries-*.json file.
func readEntryFile(path string) (oewnEntryFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	var entries oewnEntryFile
	if err := json.NewDecoder(f).Decode(&entries); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}
	return entries, nil
}

// readSynsetFile reads a single synset file ({pos}.{category}.json).
func readSynsetFile(path string) (map[string]oewnSynset, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	var synsets map[string]oewnSynset
	if err := json.NewDecoder(f).Decode(&synsets); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}
	return synsets, nil
}

// globSynsetFiles finds all synset files in the directory.
// Synset files follow the pattern: {pos}.{category}.json where pos is noun/verb/adj/adv.
func globSynsetFiles(dirPath string) ([]string, error) {
	var result []string
	for _, prefix := range []string{"noun.", "verb.", "adj.", "adv."} {
		matches, err := filepath.Glob(filepath.Join(dirPath, prefix+"*.json"))
		if err != nil {
			return nil, err
		}
		result = append(result, matches...)
	}
	// Also include frames.json-like files? No — only synset files matter.
	return result, nil
}

// appendUnique appends s to the slice only if not already present.
func appendUnique(sl []string, s string) []string {
	if slices.Contains(sl, s) {
		return sl
	}
	return append(sl, s)
}

