// Package wordnet parses Open English WordNet GWN-LMF JSON files into word relations.
// Pure function: file path in, domain structs out. No database dependencies.
package wordnet

import (
	"encoding/json"
	"fmt"
	"os"

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
	TotalSynsets     int
	TotalEntries     int
	TotalRelations   int
	FilteredByKnown int
	SelfReferential  int
	Duplicates       int
}

// GWN-LMF JSON internal types for deserialization.

type gwnDocument struct {
	Graph []gwnLexicon `json:"@graph"`
}

type gwnLexicon struct {
	Entries []gwnEntry  `json:"entry"`
	Synsets []gwnSynset `json:"synset"`
}

type gwnEntry struct {
	ID    string   `json:"@id"`
	Lemma gwnLemma `json:"lemma"`
	Sense []gwnSense `json:"sense"`
}

type gwnLemma struct {
	WrittenForm string `json:"writtenForm"`
}

type gwnSense struct {
	ID        string        `json:"@id"`
	Synset    string        `json:"synset"`
	Relations []gwnRelation `json:"relations"`
}

type gwnSynset struct {
	ID        string        `json:"@id"`
	Relations []gwnRelation `json:"relations"`
}

type gwnRelation struct {
	RelType string `json:"relType"`
	Target  string `json:"target"`
}

// dedupKey is used for deduplicating relations.
type dedupKey struct {
	source   string
	target   string
	relType  string
}

// Parse reads a GWN-LMF JSON file and extracts relations for known words.
func Parse(filePath string, knownWords map[string]bool) (ParseResult, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return ParseResult{}, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	var doc gwnDocument
	if err := json.NewDecoder(f).Decode(&doc); err != nil {
		return ParseResult{}, fmt.Errorf("decode JSON: %w", err)
	}

	if len(knownWords) == 0 {
		return ParseResult{}, nil
	}

	var result ParseResult
	seen := make(map[dedupKey]bool)

	// addRelation attempts to add a relation, applying filtering rules.
	addRelation := func(source, target, relType string) {
		// Skip self-referential.
		if source == target {
			result.Stats.SelfReferential++
			return
		}

		// Both words must be known.
		if !knownWords[source] || !knownWords[target] {
			result.Stats.FilteredByKnown++
			return
		}

		// Apply directionality conventions.
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

	for _, lex := range doc.Graph {
		result.Stats.TotalEntries += len(lex.Entries)
		result.Stats.TotalSynsets += len(lex.Synsets)

		// Step 1: Build senseID → word mapping.
		senseToWord := make(map[string]string)
		for _, entry := range lex.Entries {
			word := domain.NormalizeText(entry.Lemma.WrittenForm)
			for _, sense := range entry.Sense {
				senseToWord[sense.ID] = word
			}
		}

		// Step 2: Build synsetID → []word mapping (from senses' synset references).
		synsetToWords := make(map[string][]string)
		synsetWordSeen := make(map[string]map[string]bool)
		for _, entry := range lex.Entries {
			word := domain.NormalizeText(entry.Lemma.WrittenForm)
			for _, sense := range entry.Sense {
				synsetID := sense.Synset
				if synsetWordSeen[synsetID] == nil {
					synsetWordSeen[synsetID] = make(map[string]bool)
				}
				if !synsetWordSeen[synsetID][word] {
					synsetWordSeen[synsetID][word] = true
					synsetToWords[synsetID] = append(synsetToWords[synsetID], word)
				}
			}
		}

		// Step 3: Extract synonyms (words sharing the same synset).
		for _, words := range synsetToWords {
			if len(words) < 2 {
				continue
			}
			for i := 0; i < len(words); i++ {
				for j := i + 1; j < len(words); j++ {
					addRelation(words[i], words[j], "synonym")
				}
			}
		}

		// Step 4: Extract sense-level relations (antonyms, derived forms).
		for _, entry := range lex.Entries {
			word := domain.NormalizeText(entry.Lemma.WrittenForm)
			for _, sense := range entry.Sense {
				for _, rel := range sense.Relations {
					targetWord, ok := senseToWord[rel.Target]
					if !ok {
						continue
					}

					switch rel.RelType {
					case "antonym":
						addRelation(word, targetWord, "antonym")
					case "derivation":
						addRelation(word, targetWord, "derived")
					}
				}
			}
		}

		// Step 5: Extract synset-level relations (hypernyms).
		for _, synset := range lex.Synsets {
			for _, rel := range synset.Relations {
				if rel.RelType != "hypernym" {
					continue
				}

				sourceWords := synsetToWords[synset.ID]
				targetWords := synsetToWords[rel.Target]

				for _, sw := range sourceWords {
					for _, tw := range targetWords {
						addRelation(sw, tw, "hypernym")
					}
				}
			}
		}
	}

	result.Stats.TotalRelations = len(result.Relations)
	return result, nil
}

// applyDirectionality applies the directionality conventions for relation types.
// - Synonyms/antonyms/derived: alphabetically smaller word as source.
// - Hypernyms: specific → general (kept as-is, since GWN-LMF already stores "has hypernym").
func applyDirectionality(source, target, relType string) (string, string) {
	switch relType {
	case "synonym", "antonym", "derived":
		if source > target {
			return target, source
		}
		return source, target
	case "hypernym":
		// Already in correct direction: source is specific, target is general.
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
