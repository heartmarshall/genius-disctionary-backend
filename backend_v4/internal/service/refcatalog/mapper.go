package refcatalog

import (
	"strings"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/provider"
)

// mapToRefEntry converts a DictionaryResult and optional translations into a domain.RefEntry
// with a full tree of senses, examples, translations, and pronunciations.
func mapToRefEntry(textNormalized string, dict *provider.DictionaryResult, translations []string) *domain.RefEntry {
	entry := &domain.RefEntry{
		ID:             uuid.New(),
		Text:           dict.Word,
		TextNormalized: textNormalized,
		Senses:         make([]domain.RefSense, 0, len(dict.Senses)),
		Pronunciations: make([]domain.RefPronunciation, 0, len(dict.Pronunciations)),
	}

	for i, s := range dict.Senses {
		sense := domain.RefSense{
			ID:           uuid.New(),
			RefEntryID:   entry.ID,
			Definition:   s.Definition,
			PartOfSpeech: mapPartOfSpeech(s.PartOfSpeech),
			SourceSlug:   "freedict",
			Position:     i,
			Examples:     make([]domain.RefExample, 0, len(s.Examples)),
			Translations: []domain.RefTranslation{},
		}

		for j, ex := range s.Examples {
			sense.Examples = append(sense.Examples, domain.RefExample{
				ID:          uuid.New(),
				RefSenseID:  sense.ID,
				Sentence:    ex.Sentence,
				Translation: ex.Translation,
				SourceSlug:  "freedict",
				Position:    j,
			})
		}

		entry.Senses = append(entry.Senses, sense)
	}

	// Attach translations to the first sense (if any senses exist).
	if len(translations) > 0 && len(entry.Senses) > 0 {
		for i, t := range translations {
			entry.Senses[0].Translations = append(entry.Senses[0].Translations, domain.RefTranslation{
				ID:         uuid.New(),
				RefSenseID: entry.Senses[0].ID,
				Text:       t,
				SourceSlug: "translate",
				Position:   i,
			})
		}
	}

	for _, p := range dict.Pronunciations {
		pron := domain.RefPronunciation{
			ID:            uuid.New(),
			RefEntryID:    entry.ID,
			Transcription: p.Transcription,
			AudioURL:      p.AudioURL,
			Region:        p.Region,
			SourceSlug:    "freedict",
		}
		entry.Pronunciations = append(entry.Pronunciations, pron)
	}

	return entry
}

// mapPartOfSpeech converts a raw part-of-speech string pointer to a domain.PartOfSpeech pointer.
// nil input returns nil. Unknown values map to PartOfSpeechOther.
func mapPartOfSpeech(pos *string) *domain.PartOfSpeech {
	if pos == nil {
		return nil
	}
	upper := strings.ToUpper(*pos)
	mapped := domain.PartOfSpeech(upper)
	if mapped.IsValid() {
		return &mapped
	}
	other := domain.PartOfSpeechOther
	return &other
}
