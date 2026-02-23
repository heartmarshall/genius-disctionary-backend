package llm_importer

import (
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// MappedEntry holds domain types ready for bulk insert.
type MappedEntry struct {
	Entry        domain.RefEntry
	Senses       []domain.RefSense
	Translations []domain.RefTranslation
	Examples     []domain.RefExample
}

// Map converts an LLMWordEntry to domain types for insertion.
// Assumes the entry has been validated via Validate() first.
func Map(e LLMWordEntry) MappedEntry {
	now := time.Now()
	entryID := uuid.New()

	sourceSlug := e.SourceSlug
	if sourceSlug == "" {
		sourceSlug = "llm"
	}

	result := MappedEntry{
		Entry: domain.RefEntry{
			ID:             entryID,
			Text:           e.Word,
			TextNormalized: domain.NormalizeText(e.Word),
			IsCoreLexicon:  false,
			CreatedAt:      now,
		},
	}

	for i, s := range e.Senses {
		senseID := uuid.New()
		pos := domain.PartOfSpeech(s.POS)

		sense := domain.RefSense{
			ID:           senseID,
			RefEntryID:   entryID,
			Definition:   s.Definition,
			PartOfSpeech: &pos,
			SourceSlug:   sourceSlug,
			Position:     i,
			CreatedAt:    now,
		}
		if s.CEFRLevel != "" {
			level := s.CEFRLevel
			sense.CEFRLevel = &level
		}
		if s.Notes != "" {
			notes := s.Notes
			sense.Notes = &notes
		}
		result.Senses = append(result.Senses, sense)

		for j, tr := range s.Translations {
			result.Translations = append(result.Translations, domain.RefTranslation{
				ID:         uuid.New(),
				RefSenseID: senseID,
				Text:       tr,
				SourceSlug: sourceSlug,
				Position:   j,
			})
		}

		for j, ex := range s.Examples {
			var translation *string
			if ex.Translation != "" {
				tr := ex.Translation
				translation = &tr
			}
			result.Examples = append(result.Examples, domain.RefExample{
				ID:          uuid.New(),
				RefSenseID:  senseID,
				Sentence:    ex.Sentence,
				Translation: translation,
				SourceSlug:  sourceSlug,
				Position:    j,
			})
		}
	}

	return result
}
