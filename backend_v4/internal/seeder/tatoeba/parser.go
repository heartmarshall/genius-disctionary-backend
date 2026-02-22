// Package tatoeba parses Tatoeba EN-RU sentence pair TSV files.
// Pure function: file path in, domain structs out. No database dependencies.
package tatoeba

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

const (
	sourceSlug        = "tatoeba"
	DefaultMaxPerWord = 5
	maxSentenceLen    = 500
	positionOffset    = 1000
)

// SentencePair holds an English sentence with its Russian translation.
type SentencePair struct {
	English string
	Russian string
}

// ParseResult holds parsed Tatoeba sentence pairs grouped by word.
type ParseResult struct {
	Sentences map[string][]SentencePair
	Stats     Stats
}

// Stats holds parser statistics for logging.
type Stats struct {
	TotalLines   int
	SkippedLong  int
	MatchedWords int
	TotalPairs   int
}

// Parse reads a Tatoeba EN-RU TSV file and returns sentence pairs grouped by matching known words.
func Parse(filePath string, knownWords map[string]bool, maxPerWord int) (ParseResult, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return ParseResult{}, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	// Collect all pairs per word before limiting.
	allPairs := make(map[string][]SentencePair)

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var stats Stats

	for scanner.Scan() {
		stats.TotalLines++
		line := scanner.Text()

		fields := strings.SplitN(line, "\t", 4)
		if len(fields) < 4 {
			continue
		}

		enText := fields[1]
		ruText := fields[3]

		if len(enText) > maxSentenceLen {
			stats.SkippedLong++
			continue
		}

		tokens := tokenize(enText)
		for _, tok := range tokens {
			if knownWords[tok] {
				allPairs[tok] = append(allPairs[tok], SentencePair{
					English: enText,
					Russian: ruText,
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return ParseResult{}, fmt.Errorf("scanner error: %w", err)
	}

	// Sort by sentence length and limit per word.
	result := ParseResult{
		Sentences: make(map[string][]SentencePair, len(allPairs)),
	}

	for word, pairs := range allPairs {
		sort.Slice(pairs, func(i, j int) bool {
			return len(pairs[i].English) < len(pairs[j].English)
		})
		if len(pairs) > maxPerWord {
			pairs = pairs[:maxPerWord]
		}
		result.Sentences[word] = pairs
	}

	stats.MatchedWords = len(result.Sentences)
	for _, pairs := range result.Sentences {
		stats.TotalPairs += len(pairs)
	}
	result.Stats = stats

	return result, nil
}

// tokenize splits a sentence into unique lowercase word tokens.
// Keeps apostrophes within words (for contractions like "don't").
func tokenize(sentence string) []string {
	seen := make(map[string]bool)
	var tokens []string

	var word strings.Builder
	runes := []rune(sentence)

	for i := range len(runes) {
		r := runes[i]

		if r == '\'' {
			// Keep apostrophe only if it's between letters.
			if word.Len() > 0 && i+1 < len(runes) && unicode.IsLetter(runes[i+1]) {
				word.WriteRune(r)
				continue
			}
			// Otherwise treat as separator — flush current word.
			if word.Len() > 0 {
				w := strings.ToLower(word.String())
				if !seen[w] {
					seen[w] = true
					tokens = append(tokens, w)
				}
				word.Reset()
			}
			continue
		}

		if unicode.IsLetter(r) {
			word.WriteRune(r)
		} else {
			// Non-letter, non-apostrophe → flush.
			if word.Len() > 0 {
				w := strings.ToLower(word.String())
				if !seen[w] {
					seen[w] = true
					tokens = append(tokens, w)
				}
				word.Reset()
			}
		}
	}

	// Flush remaining.
	if word.Len() > 0 {
		w := strings.ToLower(word.String())
		if !seen[w] {
			seen[w] = true
			tokens = append(tokens, w)
		}
	}

	return tokens
}

// ToDomainExamples converts parsed sentence pairs to domain RefExample records.
func (r ParseResult) ToDomainExamples(entryIDMap map[string]uuid.UUID, senseIDMap map[uuid.UUID]uuid.UUID) []domain.RefExample {
	if len(entryIDMap) == 0 || len(senseIDMap) == 0 {
		return nil
	}

	var result []domain.RefExample

	for word, pairs := range r.Sentences {
		entryID, ok := entryIDMap[word]
		if !ok {
			continue
		}

		senseID, ok := senseIDMap[entryID]
		if !ok {
			continue
		}

		for i, pair := range pairs {
			translation := pair.Russian
			result = append(result, domain.RefExample{
				ID:          uuid.New(),
				RefSenseID:  senseID,
				Sentence:    pair.English,
				Translation: &translation,
				SourceSlug:  sourceSlug,
				Position:    positionOffset + i,
			})
		}
	}

	return result
}
