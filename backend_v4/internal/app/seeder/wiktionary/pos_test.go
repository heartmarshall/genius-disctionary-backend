package wiktionary

import (
	"testing"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

func TestMapPOS(t *testing.T) {
	tests := []struct {
		input string
		want  domain.PartOfSpeech
	}{
		// Direct 1:1 mappings
		{"noun", domain.PartOfSpeechNoun},
		{"verb", domain.PartOfSpeechVerb},
		{"adj", domain.PartOfSpeechAdjective},
		{"adv", domain.PartOfSpeechAdverb},
		{"pron", domain.PartOfSpeechPronoun},
		{"prep", domain.PartOfSpeechPreposition},
		{"conj", domain.PartOfSpeechConjunction},
		{"intj", domain.PartOfSpeechInterjection},
		{"phrase", domain.PartOfSpeechPhrase},
		{"idiom", domain.PartOfSpeechIdiom},

		// Proper nouns → NOUN
		{"name", domain.PartOfSpeechNoun},

		// Proverb → PHRASE
		{"proverb", domain.PartOfSpeechPhrase},

		// Numeric / determiners / particles / articles → OTHER
		{"num", domain.PartOfSpeechOther},
		{"det", domain.PartOfSpeechOther},
		{"particle", domain.PartOfSpeechOther},
		{"article", domain.PartOfSpeechOther},

		// Morphological affixes → OTHER
		{"affix", domain.PartOfSpeechOther},
		{"prefix", domain.PartOfSpeechOther},
		{"suffix", domain.PartOfSpeechOther},
		{"infix", domain.PartOfSpeechOther},

		// Characters / symbols / punctuation → OTHER
		{"character", domain.PartOfSpeechOther},
		{"symbol", domain.PartOfSpeechOther},
		{"punctuation", domain.PartOfSpeechOther},

		// Misc → OTHER
		{"contraction", domain.PartOfSpeechOther},
		{"abbrev", domain.PartOfSpeechOther},

		// Unknown values → OTHER
		{"completely_unknown_pos", domain.PartOfSpeechOther},
		{"", domain.PartOfSpeechOther},

		// Case insensitivity
		{"NOUN", domain.PartOfSpeechNoun},
		{"Verb", domain.PartOfSpeechVerb},
		{"ADJ", domain.PartOfSpeechAdjective},
		{"Adv", domain.PartOfSpeechAdverb},
		{"PRON", domain.PartOfSpeechPronoun},
		{"Prep", domain.PartOfSpeechPreposition},
		{"CONJ", domain.PartOfSpeechConjunction},
		{"Intj", domain.PartOfSpeechInterjection},
		{"PHRASE", domain.PartOfSpeechPhrase},
		{"IDIOM", domain.PartOfSpeechIdiom},
		{"Name", domain.PartOfSpeechNoun},
		{"PROVERB", domain.PartOfSpeechPhrase},
		{"Prefix", domain.PartOfSpeechOther},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := MapPOS(tt.input)
			if got != tt.want {
				t.Errorf("MapPOS(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
