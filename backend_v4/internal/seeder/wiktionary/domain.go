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

// ToDomainEntries converts parsed Wiktionary entries into flat domain slices
// suitable for batch insertion. All entities within a single call share
// the same CreatedAt timestamp.
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

		// Position counter is sequential across all POS groups for this entry.
		sensePos := 0

		for pgIdx := range pe.POSGroups {
			pg := &pe.POSGroups[pgIdx]
			pos := MapPOS(pg.POS)

			for sIdx := range pg.Senses {
				ps := &pg.Senses[sIdx]

				// Skip senses with empty glosses.
				if len(ps.Glosses) == 0 {
					continue
				}

				senseID := uuid.New()
				def := TruncateDefinition(StripMarkup(ps.Glosses[0]), 5000)

				result.Senses = append(result.Senses, domain.RefSense{
					ID:           senseID,
					RefEntryID:   entryID,
					Definition:   def,
					PartOfSpeech: &pos,
					SourceSlug:   sourceSlug,
					Position:     sensePos,
					CreatedAt:    now,
				})
				sensePos++

				// Translations.
				for trIdx, tr := range ps.Translations {
					result.Translations = append(result.Translations, domain.RefTranslation{
						ID:         uuid.New(),
						RefSenseID: senseID,
						Text:       tr,
						SourceSlug: sourceSlug,
						Position:   trIdx,
					})
				}

				// Examples.
				for exIdx, ex := range ps.Examples {
					result.Examples = append(result.Examples, domain.RefExample{
						ID:          uuid.New(),
						RefSenseID:  senseID,
						Sentence:    StripMarkup(ex),
						Translation: nil,
						SourceSlug:  sourceSlug,
						Position:    exIdx,
					})
				}
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
