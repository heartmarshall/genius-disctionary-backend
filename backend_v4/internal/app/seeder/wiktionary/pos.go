package wiktionary

import (
	"strings"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// posMap maps lowercase Wiktionary/Kaikki POS strings to domain PartOfSpeech values.
var posMap = map[string]domain.PartOfSpeech{
	// Direct 1:1 mappings
	"noun": domain.PartOfSpeechNoun,
	"verb": domain.PartOfSpeechVerb,
	"adj":  domain.PartOfSpeechAdjective,
	"adv":  domain.PartOfSpeechAdverb,
	"pron": domain.PartOfSpeechPronoun,
	"prep": domain.PartOfSpeechPreposition,
	"conj": domain.PartOfSpeechConjunction,
	"intj": domain.PartOfSpeechInterjection,

	// Multi-word categories
	"phrase": domain.PartOfSpeechPhrase,
	"idiom":  domain.PartOfSpeechIdiom,

	// Proper nouns → NOUN
	"name": domain.PartOfSpeechNoun,

	// Proverb → PHRASE
	"proverb": domain.PartOfSpeechPhrase,

	// Everything below maps to OTHER
	"num":          domain.PartOfSpeechOther,
	"det":          domain.PartOfSpeechOther,
	"particle":     domain.PartOfSpeechOther,
	"article":      domain.PartOfSpeechOther,
	"affix":        domain.PartOfSpeechOther,
	"prefix":       domain.PartOfSpeechOther,
	"suffix":       domain.PartOfSpeechOther,
	"infix":        domain.PartOfSpeechOther,
	"character":    domain.PartOfSpeechOther,
	"symbol":       domain.PartOfSpeechOther,
	"punctuation":  domain.PartOfSpeechOther,
	"contraction":  domain.PartOfSpeechOther,
	"abbrev":       domain.PartOfSpeechOther,
}

// MapPOS converts a Wiktionary/Kaikki POS string to the domain PartOfSpeech enum.
// The lookup is case-insensitive. Unknown or empty values map to PartOfSpeechOther.
func MapPOS(wiktionaryPOS string) domain.PartOfSpeech {
	if pos, ok := posMap[strings.ToLower(wiktionaryPOS)]; ok {
		return pos
	}
	return domain.PartOfSpeechOther
}
