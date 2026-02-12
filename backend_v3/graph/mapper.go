package graph

import (
	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/graph/model"
	"github.com/heartmarshall/my-english/internal/service/dictionary"
	"github.com/heartmarshall/my-english/internal/service/topic"
)

// mapCreateWordInput конвертирует GraphQL input в сервисный input
func mapCreateWordInput(input model.CreateWordInput) dictionary.CreateWordInput {
	topicIDs := make([]string, 0, len(input.TopicIDs))
	for _, id := range input.TopicIDs {
		topicIDs = append(topicIDs, id.String())
	}
	return dictionary.CreateWordInput{
		Text:           input.Text,
		Notes:          input.Notes,
		Senses:         mapSensesInput(input.Senses),
		Images:         mapImagesInput(input.Images),
		Pronunciations: mapPronunciationsInput(input.Pronunciations),
		TopicIDs:       topicIDs,
		CreateCard:     input.CreateCard,
	}
}

// mapUpdateWordInput конвертирует GraphQL input в сервисный input
func mapUpdateWordInput(id string, input model.UpdateWordInput) dictionary.UpdateWordInput {
	var topicIDs []string
	if input.TopicIDs != nil {
		topicIDs = make([]string, 0, len(input.TopicIDs))
		for _, topicID := range input.TopicIDs {
			topicIDs = append(topicIDs, topicID.String())
		}
	}
	return dictionary.UpdateWordInput{
		ID:             id,
		Text:           input.Text,
		Notes:          input.Notes,
		Senses:         mapSensesInput(input.Senses),
		Images:         mapImagesInput(input.Images),
		Pronunciations: mapPronunciationsInputForUpdate(input.Pronunciations),
		TopicIDs:       topicIDs,
	}
}

func mapSensesInput(inputs []*model.SenseInput) []dictionary.SenseInput {
	if len(inputs) == 0 {
		return nil
	}
	res := make([]dictionary.SenseInput, len(inputs))
	for i, in := range inputs {
		if in == nil {
			continue
		}
		res[i] = dictionary.SenseInput{
			Definition:   in.Definition, // ИСПРАВЛЕНО: передаем указатель как есть
			PartOfSpeech: in.PartOfSpeech,
			SourceSlug:   getString(in.SourceSlug),
			Translations: mapTranslationsInput(in.Translations),
			Examples:     mapExamplesInput(in.Examples),
		}
	}
	return res
}

func mapTranslationsInput(inputs []*model.TranslationInput) []dictionary.TranslationInput {
	if len(inputs) == 0 {
		return nil
	}
	res := make([]dictionary.TranslationInput, len(inputs))
	for i, in := range inputs {
		if in == nil {
			continue
		}
		res[i] = dictionary.TranslationInput{
			Text:       in.Text,
			SourceSlug: getString(in.SourceSlug),
		}
	}
	return res
}

func mapExamplesInput(inputs []*model.ExampleInput) []dictionary.ExampleInput {
	if len(inputs) == 0 {
		return nil
	}
	res := make([]dictionary.ExampleInput, len(inputs))
	for i, in := range inputs {
		if in == nil {
			continue
		}
		res[i] = dictionary.ExampleInput{
			Sentence:    in.Sentence,
			Translation: in.Translation,
			SourceSlug:  getString(in.SourceSlug),
		}
	}
	return res
}

func mapImagesInput(inputs []*model.ImageInput) []dictionary.ImageInput {
	if inputs == nil {
		// nil означает, что поле не было передано в GraphQL - не трогаем существующие
		return nil
	}
	// Пустой массив означает "удалить все" - возвращаем пустой слайс (не nil)
	if len(inputs) == 0 {
		return []dictionary.ImageInput{}
	}
	res := make([]dictionary.ImageInput, 0, len(inputs))
	for _, in := range inputs {
		if in == nil {
			continue
		}
		res = append(res, dictionary.ImageInput{
			URL:        in.URL,
			Caption:    in.Caption,
			SourceSlug: getString(in.SourceSlug),
		})
	}
	return res
}

func mapPronunciationsInput(inputs []*model.PronunciationInput) []dictionary.PronunciationInput {
	if len(inputs) == 0 {
		return nil
	}
	res := make([]dictionary.PronunciationInput, len(inputs))
	for i, in := range inputs {
		if in == nil {
			continue
		}
		// Преобразуем nullable AudioURL в строку (nil -> пустая строка)
		audioURL := ""
		if in.AudioURL != nil {
			audioURL = *in.AudioURL
		}

		// Transcription теперь обязательное поле (string)
		transcription := in.Transcription

		res[i] = dictionary.PronunciationInput{
			AudioURL:      audioURL,
			Transcription: &transcription,
			Region:        in.Region,
			SourceSlug:    getString(in.SourceSlug),
		}
	}
	return res
}

// mapPronunciationsInputForUpdate мапит произношения для обновления с учетом nil vs пустого массива
func mapPronunciationsInputForUpdate(inputs []*model.PronunciationInput) []dictionary.PronunciationInput {
	if inputs == nil {
		// nil означает, что поле не было передано в GraphQL - не трогаем существующие
		return nil
	}
	// Пустой массив означает "удалить все" - возвращаем пустой слайс (не nil)
	if len(inputs) == 0 {
		return []dictionary.PronunciationInput{}
	}
	res := make([]dictionary.PronunciationInput, 0, len(inputs))
	for _, in := range inputs {
		if in == nil {
			continue
		}
		// Преобразуем nullable AudioURL в строку (nil -> пустая строка)
		audioURL := ""
		if in.AudioURL != nil {
			audioURL = *in.AudioURL
		}

		// Transcription теперь обязательное поле (string)
		transcription := in.Transcription

		res = append(res, dictionary.PronunciationInput{
			AudioURL:      audioURL,
			Transcription: &transcription,
			Region:        in.Region,
			SourceSlug:    getString(in.SourceSlug),
		})
	}
	return res
}

// mapDictionaryFilter мапит фильтр для поиска
func mapDictionaryFilter(f *model.WordFilter) dictionary.DictionaryFilter {
	if f == nil {
		return dictionary.DictionaryFilter{}
	}

	// Маппим topicIDs (уже []uuid.UUID, просто присваиваем)
	var topicIDs []uuid.UUID
	if len(f.TopicIDs) > 0 {
		topicIDs = f.TopicIDs
	}

	return dictionary.DictionaryFilter{
		Search:       getString(f.Search),
		PartOfSpeech: f.PartOfSpeech,
		HasCard:      f.HasCard,
		TopicIDs:     topicIDs,
		Limit:        getInt(f.Limit, 20),
		Offset:       getInt(f.Offset, 0),
		SortBy:       f.SortBy,
		SortDir:      f.SortDir,
	}
}

// Helpers

func getString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func getInt(i *int, def int) int {
	if i == nil {
		return def
	}
	return *i
}

// mapCreateTopicInput конвертирует GraphQL input в сервисный input
func mapCreateTopicInput(input model.CreateTopicInput) topic.CreateTopicInput {
	return topic.CreateTopicInput{
		Name:        input.Name,
		Description: input.Description,
	}
}

// mapUpdateTopicInput конвертирует GraphQL input в сервисный input
func mapUpdateTopicInput(input model.UpdateTopicInput) topic.UpdateTopicInput {
	return topic.UpdateTopicInput{
		ID:          input.ID.String(),
		Name:        input.Name,
		Description: input.Description,
	}
}
