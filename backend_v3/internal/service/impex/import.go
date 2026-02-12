package impex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/service/dictionary"
)

// Import импортирует словарь из JSON файла.
// Стратегия: SKIP_EXISTING - пропускает существующие слова.
// Использует DictionaryService.CreateWord для переиспользования бизнес-логики.
func (s *Service) Import(ctx context.Context, r io.Reader) (*ImportReport, error) {
	report := &ImportReport{
		Errors: make([]string, 0),
	}

	// Загружаем все топики для маппинга name -> ID
	allTopics, err := s.repos.Topics.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all topics: %w", err)
	}
	topicNameToID := make(map[string]uuid.UUID)
	for _, topic := range allTopics {
		topicNameToID[topic.Name] = topic.ID
	}

	// Читаем JSON потоково через json.Decoder
	decoder := json.NewDecoder(r)

	// Проверяем, что это массив
	token, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("decode json token: %w", err)
	}

	if delim, ok := token.(json.Delim); !ok || delim != '[' {
		return nil, fmt.Errorf("expected array, got %T", token)
	}

	// Итерируемся по записям
	for decoder.More() {
		var exportEntry ExportEntry
		if err := decoder.Decode(&exportEntry); err != nil {
			if err == io.EOF {
				break
			}
			report.FailedCount++
			report.Errors = append(report.Errors, fmt.Sprintf("decode entry: %v", err))
			continue
		}

		report.TotalProcessed++

		// Проверяем наличие слова через DictionaryRepository.ExistsByNormalizedText
		textNorm := normalizeText(exportEntry.Entry.Text)
		exists, err := s.repos.Dictionary.ExistsByNormalizedText(ctx, textNorm)
		if err != nil {
			report.FailedCount++
			report.Errors = append(report.Errors, fmt.Sprintf("check existence for '%s': %v", exportEntry.Entry.Text, err))
			continue
		}

		if exists {
			// Слово уже существует - пропускаем
			report.SkippedCount++
			continue
		}

		// Маппим данные в dictionary.CreateWordInput
		input := mapExportEntryToCreateWordInput(exportEntry, topicNameToID)

		// Вызываем DictionaryService.CreateWord (использует существующую бизнес-логику)
		_, err = s.dictionarySvc.CreateWord(ctx, input)
		if err != nil {
			report.FailedCount++
			report.Errors = append(report.Errors, fmt.Sprintf("create word '%s': %v", exportEntry.Entry.Text, err))
			continue
		}

		report.SuccessCount++
	}

	// Проверяем закрывающую скобку массива
	token, err = decoder.Token()
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("decode closing bracket: %w", err)
	}

	return report, nil
}

// mapExportEntryToCreateWordInput маппит ExportEntry в CreateWordInput.
// topicNameToID используется для маппинга имен топиков в ID при импорте.
func mapExportEntryToCreateWordInput(entry ExportEntry, topicNameToID map[string]uuid.UUID) dictionary.CreateWordInput {
	input := dictionary.CreateWordInput{
		Text:           entry.Entry.Text,
		Notes:          entry.Entry.Notes,
		Senses:         make([]dictionary.SenseInput, 0, len(entry.Senses)),
		Images:         make([]dictionary.ImageInput, 0, len(entry.Images)),
		Pronunciations: make([]dictionary.PronunciationInput, 0, len(entry.Pronunciations)),
		TopicIDs:       make([]string, 0, len(entry.TopicNames)),
		CreateCard:     entry.Card != nil,
	}

	// Маппим topic names в IDs
	for _, topicName := range entry.TopicNames {
		if topicID, ok := topicNameToID[topicName]; ok {
			input.TopicIDs = append(input.TopicIDs, topicID.String())
		}
		// Если топик не найден, просто пропускаем его (не критичная ошибка)
	}

	// Маппим senses
	for _, sense := range entry.Senses {
		senseInput := dictionary.SenseInput{
			Definition:   sense.Definition,
			PartOfSpeech: sense.PartOfSpeech,
			SourceSlug:   sense.SourceSlug,
			Translations: make([]dictionary.TranslationInput, 0, len(sense.Translations)),
			Examples:     make([]dictionary.ExampleInput, 0, len(sense.Examples)),
		}

		// Маппим translations
		for _, tr := range sense.Translations {
			senseInput.Translations = append(senseInput.Translations, dictionary.TranslationInput{
				Text:       tr.Text,
				SourceSlug: tr.SourceSlug,
			})
		}

		// Маппим examples
		for _, ex := range sense.Examples {
			senseInput.Examples = append(senseInput.Examples, dictionary.ExampleInput{
				Sentence:    ex.Sentence,
				Translation: ex.Translation,
				SourceSlug:  ex.SourceSlug,
			})
		}

		input.Senses = append(input.Senses, senseInput)
	}

	// Маппим images
	for _, img := range entry.Images {
		input.Images = append(input.Images, dictionary.ImageInput{
			URL:        img.URL,
			Caption:    img.Caption,
			SourceSlug: img.SourceSlug,
		})
	}

	// Маппим pronunciations
	for _, pron := range entry.Pronunciations {
		// Преобразуем nullable AudioURL в строку (nil -> пустая строка)
		audioURL := ""
		if pron.AudioURL != nil {
			audioURL = *pron.AudioURL
		}

		// Transcription теперь обязательное поле (string)
		transcription := pron.Transcription

		input.Pronunciations = append(input.Pronunciations, dictionary.PronunciationInput{
			AudioURL:      audioURL,
			Transcription: &transcription,
			Region:        pron.Region,
			SourceSlug:    pron.SourceSlug,
		})
	}

	// Маппим card options (для восстановления SRS прогресса)
	if entry.Card != nil {
		status := entry.Card.Status
		intervalDays := entry.Card.IntervalDays
		easeFactor := entry.Card.EaseFactor
		nextReviewAt := entry.Card.NextReviewAt

		input.CardOptions = &dictionary.CardOptions{
			Status:       &status,
			IntervalDays: &intervalDays,
			EaseFactor:   &easeFactor,
			NextReviewAt: nextReviewAt,
		}
	}

	return input
}

// normalizeText нормализует текст слова (дублируем логику из dictionary service).
func normalizeText(text string) string {
	// Удаляем пробелы и приводим к нижнему регистру
	return strings.ToLower(strings.TrimSpace(text))
}
