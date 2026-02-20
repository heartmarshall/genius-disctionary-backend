package wiktionary

import "strings"

// ScoreEntry evaluates a Kaikki entry's usefulness based on the richness
// of its content. Higher scores indicate more valuable entries for import.
func ScoreEntry(entry *kaikkiEntry) float64 {
	var score float64

	for _, sense := range entry.Senses {
		// Each sense with at least one gloss: +1.0
		if len(sense.Glosses) > 0 {
			score += 1.0
		}

		// Each Russian translation: +0.5
		for _, tr := range sense.Translations {
			if tr.Code == "ru" {
				score += 0.5
			}
		}

		// Each example: +0.3
		score += 0.3 * float64(len(sense.Examples))
	}

	// Has IPA pronunciation: +2.0 (once, not per sound)
	for _, sound := range entry.Sounds {
		if sound.IPA != "" {
			score += 2.0
			break
		}
	}

	// Single-word entry bonus: +1.0
	if !strings.Contains(entry.Word, " ") {
		score += 1.0
	}

	return score
}
