package wiktionary

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

const (
	// coreWordBonus is added to quality scores for NGSL/NAWL words
	// to guarantee their inclusion in the top N.
	coreWordBonus = 1000.0

	// maxLineSize is the buffer size for bufio.Scanner (16 MB).
	maxLineSize = 16 << 20

	// maxDefinitionLen is the maximum length for definitions.
	maxDefinitionLen = 5000
)

// Parse performs a two-pass parse of a Kaikki JSONL file.
// Pass 1 scores entries and selects top N words.
// Pass 2 fully parses only selected words.
// coreWords is a set of NGSL/NAWL words guaranteed inclusion.
func Parse(filePath string, coreWords map[string]bool, topN int) ([]ParsedEntry, Stats, error) {
	scores, stats, err := scoringPass(filePath, coreWords)
	if err != nil {
		return nil, stats, fmt.Errorf("scoring pass: %w", err)
	}

	if len(scores) == 0 {
		return nil, stats, nil
	}

	selected := selectTopN(scores, coreWords, topN)

	entries, err := parsingPass(filePath, selected)
	if err != nil {
		return nil, stats, fmt.Errorf("parsing pass: %w", err)
	}

	stats.EntriesParsed = len(entries)
	return entries, stats, nil
}

// scoringPass streams the JSONL file, scoring each English entry.
// Returns cumulative scores per normalized word and parse statistics.
func scoringPass(filePath string, coreWords map[string]bool) (map[string]float64, Stats, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, Stats{}, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	scores := make(map[string]float64)
	var stats Stats

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, maxLineSize), maxLineSize)

	for scanner.Scan() {
		stats.TotalLines++
		line := scanner.Bytes()

		var entry kaikkiEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			stats.MalformedLines++
			continue
		}

		if entry.Lang != "English" {
			continue
		}
		stats.EnglishLines++

		word := domain.NormalizeText(entry.Word)
		if word == "" {
			continue
		}

		score := ScoreEntry(&entry)
		scores[word] += score
	}

	if err := scanner.Err(); err != nil {
		return nil, stats, fmt.Errorf("scanner error: %w", err)
	}

	// Apply core word bonus.
	for w := range coreWords {
		normalized := domain.NormalizeText(w)
		if _, ok := scores[normalized]; ok {
			scores[normalized] += coreWordBonus
		}
	}

	return scores, stats, nil
}

// selectTopN picks the top N words by score. Core words are guaranteed
// inclusion regardless of score. If core words exceed topN, all are included.
func selectTopN(scores map[string]float64, coreWords map[string]bool, topN int) map[string]bool {
	if len(scores) == 0 {
		return map[string]bool{}
	}

	type wordScore struct {
		word  string
		score float64
	}

	sorted := make([]wordScore, 0, len(scores))
	for w, s := range scores {
		sorted = append(sorted, wordScore{w, s})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].score > sorted[j].score
	})

	selected := make(map[string]bool, topN)

	// Guarantee all core words first.
	for w := range coreWords {
		normalized := domain.NormalizeText(w)
		if _, ok := scores[normalized]; ok {
			selected[normalized] = true
		}
	}

	// Fill remaining slots from sorted scores.
	for _, ws := range sorted {
		if len(selected) >= topN {
			break
		}
		selected[ws.word] = true
	}

	return selected
}

// parsingPass re-streams the file, fully parsing only entries for selected words.
// Entries with the same normalized word are merged (POS groups and sounds combined).
func parsingPass(filePath string, selected map[string]bool) ([]ParsedEntry, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	// Map from normalized word to index in entries slice.
	entryIndex := make(map[string]int)
	var entries []ParsedEntry

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, maxLineSize), maxLineSize)

	for scanner.Scan() {
		line := scanner.Bytes()

		var entry kaikkiEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		if entry.Lang != "English" {
			continue
		}

		word := domain.NormalizeText(entry.Word)
		if !selected[word] {
			continue
		}

		pg := buildPOSGroup(&entry)
		sounds := buildSounds(&entry)

		idx, exists := entryIndex[word]
		if !exists {
			entryIndex[word] = len(entries)
			entries = append(entries, ParsedEntry{
				Word:      entry.Word,
				POSGroups: []POSGroup{pg},
				Sounds:    sounds,
			})
		} else {
			entries[idx].POSGroups = append(entries[idx].POSGroups, pg)
			entries[idx].Sounds = mergeSounds(entries[idx].Sounds, sounds)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return entries, nil
}

// buildPOSGroup extracts senses from a Kaikki entry into a POSGroup.
func buildPOSGroup(entry *kaikkiEntry) POSGroup {
	pg := POSGroup{POS: entry.POS}

	for i := range entry.Senses {
		ks := &entry.Senses[i]

		// Skip senses without glosses.
		if len(ks.Glosses) == 0 {
			continue
		}

		ps := ParsedSense{}

		// Clean glosses.
		for _, g := range ks.Glosses {
			cleaned := TruncateDefinition(StripMarkup(g), maxDefinitionLen)
			if cleaned != "" {
				ps.Glosses = append(ps.Glosses, cleaned)
			}
		}
		if len(ps.Glosses) == 0 {
			continue
		}

		// Clean examples.
		for _, ex := range ks.Examples {
			cleaned := StripMarkup(ex.Text)
			if cleaned != "" {
				ps.Examples = append(ps.Examples, cleaned)
			}
		}

		// Collect Russian translations only, deduplicated.
		var ruTranslations []string
		for _, tr := range ks.Translations {
			if tr.Code == "ru" && tr.Word != "" {
				ruTranslations = append(ruTranslations, tr.Word)
			}
		}
		ps.Translations = DeduplicateStrings(ruTranslations)

		pg.Senses = append(pg.Senses, ps)
	}

	return pg
}

// buildSounds extracts IPA pronunciations from a Kaikki entry.
// Only phonemic transcriptions (wrapped in /slashes/) are kept;
// phonetic/allophonic variants in [brackets] are skipped.
func buildSounds(entry *kaikkiEntry) []Sound {
	var sounds []Sound
	for _, ks := range entry.Sounds {
		if ks.IPA == "" {
			continue
		}
		// Keep only phonemic transcriptions in /slashes/.
		if !strings.HasPrefix(ks.IPA, "/") {
			continue
		}
		sounds = append(sounds, Sound{
			IPA:    ks.IPA,
			Region: extractRegion(ks.Tags),
		})
	}
	return sounds
}

// extractRegion parses sound tags to determine US/UK region.
func extractRegion(tags []string) string {
	for _, tag := range tags {
		switch {
		case tag == "US" || tag == "General-American" || strings.Contains(tag, "GenAm"):
			return "US"
		case tag == "UK" || tag == "Received-Pronunciation" || strings.Contains(tag, "RP"):
			return "UK"
		}
	}
	return ""
}

// mergeSounds combines two sound slices, deduplicating by IPA+Region.
func mergeSounds(existing, new []Sound) []Sound {
	seen := make(map[string]bool, len(existing))
	for _, s := range existing {
		seen[s.IPA+"|"+s.Region] = true
	}
	for _, s := range new {
		key := s.IPA + "|" + s.Region
		if !seen[key] {
			existing = append(existing, s)
			seen[key] = true
		}
	}
	return existing
}
