package dictionary

import (
	"strings"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/model"
)

// normalizeText нормализует текст для поиска и сравнения.
// Приводит к нижнему регистру и удаляет пробелы по краям.
func normalizeText(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// buildEntry создает модель DictionaryEntry из входных данных.
func buildDictionaryEntry(textRaw, textNorm string) *model.DictionaryEntry {
	return &model.DictionaryEntry{
		Text:           textRaw,
		TextNormalized: textNorm,
	}
}

// buildDictionaryEntryWithNotes создает модель DictionaryEntry с заметками.
func buildDictionaryEntryWithNotes(textRaw, textNorm string, notes *string) *model.DictionaryEntry {
	entry := buildDictionaryEntry(textRaw, textNorm)
	entry.Notes = notes
	return entry
}

// buildSense создает модель Sense из входных данных.
func buildSense(entryID uuid.UUID, senseIn SenseInput) *model.Sense {
	return &model.Sense{
		EntryID:      entryID,
		Definition:   senseIn.Definition,
		PartOfSpeech: senseIn.PartOfSpeech,
		SourceSlug:   senseIn.SourceSlug,
	}
}

// buildTranslations создает слайс моделей Translation из входных данных.
func buildTranslations(senseID uuid.UUID, translations []TranslationInput) []model.Translation {
	if len(translations) == 0 {
		return nil
	}

	result := make([]model.Translation, len(translations))
	for i, tr := range translations {
		result[i] = model.Translation{
			SenseID:    senseID,
			Text:       tr.Text,
			SourceSlug: tr.SourceSlug,
		}
	}
	return result
}

// buildExamples создает слайс моделей Example из входных данных.
func buildExamples(senseID uuid.UUID, examples []ExampleInput) []model.Example {
	if len(examples) == 0 {
		return nil
	}

	result := make([]model.Example, len(examples))
	for i, ex := range examples {
		result[i] = model.Example{
			SenseID:     senseID,
			Sentence:    ex.Sentence,
			Translation: ex.Translation,
			SourceSlug:  ex.SourceSlug,
		}
	}
	return result
}

// buildImages создает слайс моделей Image из входных данных.
func buildImages(entryID uuid.UUID, images []ImageInput) []model.Image {
	if len(images) == 0 {
		return nil
	}

	result := make([]model.Image, len(images))
	for i, img := range images {
		result[i] = model.Image{
			EntryID:    entryID,
			URL:        img.URL,
			Caption:    img.Caption,
			SourceSlug: img.SourceSlug,
		}
	}
	return result
}

// buildPronunciations создает слайс моделей Pronunciation из входных данных.
func buildPronunciations(entryID uuid.UUID, pronunciations []PronunciationInput) []model.Pronunciation {
	if len(pronunciations) == 0 {
		return nil
	}

	result := make([]model.Pronunciation, len(pronunciations))
	for i, p := range pronunciations {
		// Преобразуем пустую строку audioURL в nil для nullable поля
		var audioURL *string
		if strings.TrimSpace(p.AudioURL) != "" {
			audioURL = &p.AudioURL
		}

		// Transcription теперь обязательное поле (не может быть nil)
		transcription := ""
		if p.Transcription != nil {
			transcription = *p.Transcription
		}

		result[i] = model.Pronunciation{
			EntryID:       entryID,
			AudioURL:      audioURL,
			Transcription: transcription,
			Region:        p.Region,
			SourceSlug:    p.SourceSlug,
		}
	}
	return result
}
