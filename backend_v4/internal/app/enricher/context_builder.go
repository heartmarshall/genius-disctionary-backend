package enricher

import (
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/app/seeder/cmu"
	"github.com/heartmarshall/myenglish-backend/internal/app/seeder/wiktionary"
)

// BuildContext assembles an EnrichContext for one word from pre-loaded dataset maps.
// All maps are keyed by normalized (lowercase) word text.
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
				definition := wiktionary.StripMarkup(s.Glosses[0])
				ctx.WiktionarySenses = append(ctx.WiktionarySenses, WiktionarySense{
					POS:          pg.POS,
					Definition:   definition,
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
