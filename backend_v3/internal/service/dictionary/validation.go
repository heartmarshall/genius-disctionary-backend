package dictionary

import (
	"fmt"
	"strings"

	"github.com/heartmarshall/my-english/internal/service/types"
)

const (
	// maxTextLength — максимальная длина текста слова
	maxTextLength = 500
)

// validateCreateWordInput валидирует входные данные для создания слова.
func validateCreateWordInput(input CreateWordInput) error {
	textRaw := strings.TrimSpace(input.Text)
	if textRaw == "" {
		return types.NewValidationError("text", "cannot be empty")
	}
	if len(textRaw) > maxTextLength {
		return types.NewValidationError("text", fmt.Sprintf("cannot exceed %d characters", maxTextLength))
	}

	// Валидация senses
	for i, sense := range input.Senses {
		if err := validateSenseInput(sense, i); err != nil {
			return err
		}
	}

	// Валидация images
	for i, img := range input.Images {
		if err := validateImageInput(img, i); err != nil {
			return err
		}
	}

	// Валидация pronunciations
	for i, pron := range input.Pronunciations {
		if err := validatePronunciationInput(pron, i); err != nil {
			return err
		}
	}

	return nil
}

// validateUpdateWordInput валидирует входные данные для обновления слова.
func validateUpdateWordInput(input UpdateWordInput) error {
	if input.ID == "" {
		return types.NewValidationError("id", "cannot be empty")
	}

	if input.Text != nil {
		textRaw := strings.TrimSpace(*input.Text)
		if textRaw == "" {
			return types.NewValidationError("text", "cannot be empty")
		}
		if len(textRaw) > maxTextLength {
			return types.NewValidationError("text", fmt.Sprintf("cannot exceed %d characters", maxTextLength))
		}
	}

	// Валидация senses
	for i, sense := range input.Senses {
		if err := validateSenseInput(sense, i); err != nil {
			return err
		}
	}

	// Валидация images
	for i, img := range input.Images {
		if err := validateImageInput(img, i); err != nil {
			return err
		}
	}

	// Валидация pronunciations
	for i, pron := range input.Pronunciations {
		if err := validatePronunciationInput(pron, i); err != nil {
			return err
		}
	}

	return nil
}

// validateSenseInput валидирует входные данные для смысла.
func validateSenseInput(sense SenseInput, index int) error {
	if sense.Definition != nil && strings.TrimSpace(*sense.Definition) == "" {
		return types.NewValidationError(
			fmt.Sprintf("senses[%d].definition", index),
			"cannot be empty if provided",
		)
	}
	if sense.SourceSlug == "" {
		return types.NewValidationError(
			fmt.Sprintf("senses[%d].sourceSlug", index),
			"is required",
		)
	}

	// Валидация translations
	for j, tr := range sense.Translations {
		if strings.TrimSpace(tr.Text) == "" {
			return types.NewValidationError(
				fmt.Sprintf("senses[%d].translations[%d].text", index, j),
				"cannot be empty",
			)
		}
		if tr.SourceSlug == "" {
			return types.NewValidationError(
				fmt.Sprintf("senses[%d].translations[%d].sourceSlug", index, j),
				"is required",
			)
		}
	}

	// Валидация examples
	for j, ex := range sense.Examples {
		if strings.TrimSpace(ex.Sentence) == "" {
			return types.NewValidationError(
				fmt.Sprintf("senses[%d].examples[%d].sentence", index, j),
				"cannot be empty",
			)
		}
		if ex.SourceSlug == "" {
			return types.NewValidationError(
				fmt.Sprintf("senses[%d].examples[%d].sourceSlug", index, j),
				"is required",
			)
		}
	}

	return nil
}

// validateImageInput валидирует входные данные для изображения.
func validateImageInput(img ImageInput, index int) error {
	if strings.TrimSpace(img.URL) == "" {
		return types.NewValidationError(
			fmt.Sprintf("images[%d].url", index),
			"cannot be empty",
		)
	}
	if img.SourceSlug == "" {
		return types.NewValidationError(
			fmt.Sprintf("images[%d].sourceSlug", index),
			"is required",
		)
	}
	return nil
}

// validatePronunciationInput валидирует входные данные для произношения.
func validatePronunciationInput(pron PronunciationInput, index int) error {
	// Transcription обязателен (хотя бы одно из transcription или audioURL должно быть заполнено)
	if pron.Transcription == nil || strings.TrimSpace(*pron.Transcription) == "" {
		return types.NewValidationError(
			fmt.Sprintf("pronunciations[%d].transcription", index),
			"is required",
		)
	}
	if pron.SourceSlug == "" {
		return types.NewValidationError(
			fmt.Sprintf("pronunciations[%d].sourceSlug", index),
			"is required",
		)
	}
	return nil
}

// validateAddSenseInput валидирует входные данные для добавления смысла.
func validateAddSenseInput(input AddSenseInput) error {
	if input.EntryID == "" {
		return types.NewValidationError("entryID", "cannot be empty")
	}
	if input.SourceSlug == "" {
		return types.NewValidationError("sourceSlug", "is required")
	}

	// Валидация translations
	for i, tr := range input.Translations {
		if strings.TrimSpace(tr.Text) == "" {
			return types.NewValidationError(
				fmt.Sprintf("translations[%d].text", i),
				"cannot be empty",
			)
		}
		if tr.SourceSlug == "" {
			return types.NewValidationError(
				fmt.Sprintf("translations[%d].sourceSlug", i),
				"is required",
			)
		}
	}

	// Валидация examples
	for i, ex := range input.Examples {
		if strings.TrimSpace(ex.Sentence) == "" {
			return types.NewValidationError(
				fmt.Sprintf("examples[%d].sentence", i),
				"cannot be empty",
			)
		}
		if ex.SourceSlug == "" {
			return types.NewValidationError(
				fmt.Sprintf("examples[%d].sourceSlug", i),
				"is required",
			)
		}
	}

	return nil
}

// validateAddExamplesInput валидирует входные данные для добавления примеров.
func validateAddExamplesInput(input AddExamplesInput) error {
	if input.SenseID == "" {
		return types.NewValidationError("senseID", "cannot be empty")
	}
	if len(input.Examples) == 0 {
		return types.NewValidationError("examples", "cannot be empty")
	}

	for i, ex := range input.Examples {
		if strings.TrimSpace(ex.Sentence) == "" {
			return types.NewValidationError(
				fmt.Sprintf("examples[%d].sentence", i),
				"cannot be empty",
			)
		}
		if ex.SourceSlug == "" {
			return types.NewValidationError(
				fmt.Sprintf("examples[%d].sourceSlug", i),
				"is required",
			)
		}
	}

	return nil
}

// validateAddTranslationsInput валидирует входные данные для добавления переводов.
func validateAddTranslationsInput(input AddTranslationsInput) error {
	if input.SenseID == "" {
		return types.NewValidationError("senseID", "cannot be empty")
	}
	if len(input.Translations) == 0 {
		return types.NewValidationError("translations", "cannot be empty")
	}

	for i, tr := range input.Translations {
		if strings.TrimSpace(tr.Text) == "" {
			return types.NewValidationError(
				fmt.Sprintf("translations[%d].text", i),
				"cannot be empty",
			)
		}
		if tr.SourceSlug == "" {
			return types.NewValidationError(
				fmt.Sprintf("translations[%d].sourceSlug", i),
				"is required",
			)
		}
	}

	return nil
}

// validateAddImagesInput валидирует входные данные для добавления изображений.
func validateAddImagesInput(input AddImagesInput) error {
	if input.EntryID == "" {
		return types.NewValidationError("entryID", "cannot be empty")
	}
	if len(input.Images) == 0 {
		return types.NewValidationError("images", "cannot be empty")
	}

	for i, img := range input.Images {
		if err := validateImageInput(img, i); err != nil {
			return err
		}
	}

	return nil
}

// validateAddPronunciationsInput валидирует входные данные для добавления произношений.
func validateAddPronunciationsInput(input AddPronunciationsInput) error {
	if input.EntryID == "" {
		return types.NewValidationError("entryID", "cannot be empty")
	}
	if len(input.Pronunciations) == 0 {
		return types.NewValidationError("pronunciations", "cannot be empty")
	}

	for i, pron := range input.Pronunciations {
		if err := validatePronunciationInput(pron, i); err != nil {
			return err
		}
	}

	return nil
}

// validateDeleteSenseInput валидирует входные данные для удаления смысла.
func validateDeleteSenseInput(input DeleteSenseInput) error {
	if input.ID == "" {
		return types.NewValidationError("id", "cannot be empty")
	}
	return nil
}

// validateDeleteExampleInput валидирует входные данные для удаления примера.
func validateDeleteExampleInput(input DeleteExampleInput) error {
	if input.ID == "" {
		return types.NewValidationError("id", "cannot be empty")
	}
	return nil
}

// validateDeleteTranslationInput валидирует входные данные для удаления перевода.
func validateDeleteTranslationInput(input DeleteTranslationInput) error {
	if input.ID == "" {
		return types.NewValidationError("id", "cannot be empty")
	}
	return nil
}

// validateDeleteImageInput валидирует входные данные для удаления изображения.
func validateDeleteImageInput(input DeleteImageInput) error {
	if input.ID == "" {
		return types.NewValidationError("id", "cannot be empty")
	}
	return nil
}

// validateDeletePronunciationInput валидирует входные данные для удаления произношения.
func validateDeletePronunciationInput(input DeletePronunciationInput) error {
	if input.ID == "" {
		return types.NewValidationError("id", "cannot be empty")
	}
	return nil
}

// validateAddTopicsInput валидирует входные данные для добавления топиков.
func validateAddTopicsInput(input AddTopicsInput) error {
	if input.EntryID == "" {
		return types.NewValidationError("entryID", "cannot be empty")
	}
	if len(input.TopicIDs) == 0 {
		return types.NewValidationError("topicIDs", "cannot be empty")
	}

	// Проверяем формат UUID для каждого topicID
	for i, topicID := range input.TopicIDs {
		if strings.TrimSpace(topicID) == "" {
			return types.NewValidationError(
				fmt.Sprintf("topicIDs[%d]", i),
				"cannot be empty",
			)
		}
		// UUID валидация будет выполнена при парсинге в транзакции
	}

	return nil
}

// validateRemoveTopicsInput валидирует входные данные для удаления топиков.
func validateRemoveTopicsInput(input RemoveTopicsInput) error {
	if input.EntryID == "" {
		return types.NewValidationError("entryID", "cannot be empty")
	}
	if len(input.TopicIDs) == 0 {
		return types.NewValidationError("topicIDs", "cannot be empty")
	}

	// Проверяем формат UUID для каждого topicID
	for i, topicID := range input.TopicIDs {
		if strings.TrimSpace(topicID) == "" {
			return types.NewValidationError(
				fmt.Sprintf("topicIDs[%d]", i),
				"cannot be empty",
			)
		}
		// UUID валидация будет выполнена при парсинге в транзакции
	}

	return nil
}
