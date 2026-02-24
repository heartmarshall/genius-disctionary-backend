package wiktionary

import (
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

const sourceSlug = "wiktionary"

// DomainResult holds the flat slices ready for batch insert.
type DomainResult struct {
	Entries        []domain.RefEntry
	Senses         []domain.RefSense
	Translations   []domain.RefTranslation
	Examples       []domain.RefExample
	Pronunciations []domain.RefPronunciation
}

// senseKey is used for deduplicating senses within a single entry.
type senseKey struct {
	definition   string
	partOfSpeech domain.PartOfSpeech
}

// mergedSense accumulates data from duplicate senses.
type mergedSense struct {
	id           uuid.UUID
	definition   string
	pos          domain.PartOfSpeech
	translations []string
	examples     []string
}

// ToDomainEntries converts parsed Wiktionary entries into flat domain slices
// suitable for batch insertion. Senses with identical (definition, partOfSpeech)
// within the same entry are merged: their examples and translations are combined.
func ToDomainEntries(entries []ParsedEntry) DomainResult {
	if len(entries) == 0 {
		return DomainResult{}
	}

	now := time.Now()

	var result DomainResult

	for i := range entries {
		pe := &entries[i]
		entryID := uuid.New()

		result.Entries = append(result.Entries, domain.RefEntry{
			ID:             entryID,
			Text:           pe.Word,
			TextNormalized: domain.NormalizeText(pe.Word),
			IsCoreLexicon:  false,
			CreatedAt:      now,
		})

		// Deduplicate senses by (definition, partOfSpeech).
		seenSenses := make(map[senseKey]int) // key → index in merged slice
		var merged []mergedSense

		for pgIdx := range pe.POSGroups {
			pg := &pe.POSGroups[pgIdx]
			pos := MapPOS(pg.POS)

			for sIdx := range pg.Senses {
				ps := &pg.Senses[sIdx]

				if len(ps.Glosses) == 0 {
					continue
				}

				def := TruncateDefinition(StripMarkup(ps.Glosses[0]), 5000)
				key := senseKey{definition: def, partOfSpeech: pos}

				if idx, exists := seenSenses[key]; exists {
					// Merge examples and translations into existing sense.
					merged[idx].examples = append(merged[idx].examples, ps.Examples...)
					merged[idx].translations = append(merged[idx].translations, ps.Translations...)
				} else {
					seenSenses[key] = len(merged)
					merged = append(merged, mergedSense{
						id:           uuid.New(),
						definition:   def,
						pos:          pos,
						translations: append([]string(nil), ps.Translations...),
						examples:     append([]string(nil), ps.Examples...),
					})
				}
			}
		}

		// Convert merged senses to domain structs.
		for sensePos, ms := range merged {
			pos := ms.pos //nolint:copyloopvar // need addressable copy for pointer
			result.Senses = append(result.Senses, domain.RefSense{
				ID:           ms.id,
				RefEntryID:   entryID,
				Definition:   ms.definition,
				PartOfSpeech: &pos,
				SourceSlug:   sourceSlug,
				Position:     sensePos,
				CreatedAt:    now,
			})

			// Deduplicated translations.
			for trIdx, tr := range DeduplicateStrings(ms.translations) {
				result.Translations = append(result.Translations, domain.RefTranslation{
					ID:         uuid.New(),
					RefSenseID: ms.id,
					Text:       tr,
					SourceSlug: sourceSlug,
					Position:   trIdx,
				})
			}

			// Deduplicated examples.
			for exIdx, ex := range DeduplicateStrings(ms.examples) {
				result.Examples = append(result.Examples, domain.RefExample{
					ID:          uuid.New(),
					RefSenseID:  ms.id,
					Sentence:    StripMarkup(ex),
					Translation: nil,
					SourceSlug:  sourceSlug,
					Position:    exIdx,
				})
			}
		}

		// Pronunciations.
		for sndIdx := range pe.Sounds {
			snd := &pe.Sounds[sndIdx]

			var region *string
			if snd.Region != "" {
				region = &snd.Region
			}

			result.Pronunciations = append(result.Pronunciations, domain.RefPronunciation{
				ID:            uuid.New(),
				RefEntryID:    entryID,
				Transcription: &snd.IPA,
				AudioURL:      nil,
				Region:        region,
				SourceSlug:    sourceSlug,
			})
		}
	}

	return result
}
