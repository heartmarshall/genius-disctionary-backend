package dictionary

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database"
	"github.com/heartmarshall/my-english/internal/model"
	"github.com/heartmarshall/my-english/internal/service/types"
)

// addSenseTx выполняет логику добавления смысла внутри транзакции.
func (s *Service) addSenseTx(ctx context.Context, input AddSenseInput, entryID uuid.UUID) (*model.Sense, error) {
	var createdSense *model.Sense

	err := s.tx.RunInTx(ctx, func(ctx context.Context, _ database.Querier) error {
		// Проверяем существование записи
		_, err := s.repos.Dictionary.GetByID(ctx, entryID)
		if err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("get entry by ID: %w", err)
		}

		// Создаем смысл
		sense := buildSense(entryID, SenseInput{
			Definition:   input.Definition,
			PartOfSpeech: input.PartOfSpeech,
			SourceSlug:   input.SourceSlug,
			Translations: input.Translations,
			Examples:     input.Examples,
		})

		createdSense, err = s.repos.Senses.Create(ctx, sense)
		if err != nil {
			return fmt.Errorf("create sense: %w", err)
		}

		// Создаем переводы
		if err := s.createTranslations(ctx, createdSense.ID, input.Translations); err != nil {
			return fmt.Errorf("create translations: %w", err)
		}

		// Создаем примеры
		if err := s.createExamples(ctx, createdSense.ID, input.Examples); err != nil {
			return fmt.Errorf("create examples: %w", err)
		}

		// Создаем аудит-лог для Entry (добавлен новый sense)
		changes := model.JSON{
			types.AuditFieldAction:  types.AuditActionSenseAdded,
			types.AuditFieldSenseID: createdSense.ID.String(),
		}
		if createdSense.Definition != nil {
			changes[types.AuditFieldDefinition] = *createdSense.Definition
		}
		if createdSense.PartOfSpeech != nil {
			changes[types.AuditFieldPartOfSpeech] = *createdSense.PartOfSpeech
		}
		if len(input.Translations) > 0 {
			changes[types.AuditFieldTranslationsCount] = len(input.Translations)
		}
		if len(input.Examples) > 0 {
			changes[types.AuditFieldExamplesCount] = len(input.Examples)
		}

		// Также создаем отдельный аудит-лог для самого sense
		senseChanges := buildCreateChanges(createdSense)
		if err := s.createAuditLogForEntity(ctx, model.EntitySense, createdSense.ID, model.ActionCreate, senseChanges); err != nil {
			return fmt.Errorf("create audit log for sense: %w", err)
		}

		if err := s.createAuditLog(ctx, entryID, model.ActionUpdate, changes); err != nil {
			return fmt.Errorf("create audit log: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return createdSense, nil
}

// addExamplesTx выполняет логику добавления примеров внутри транзакции.
func (s *Service) addExamplesTx(ctx context.Context, input AddExamplesInput, senseID uuid.UUID) error {
	return s.tx.RunInTx(ctx, func(ctx context.Context, _ database.Querier) error {
		// Проверяем существование смысла
		sense, err := s.repos.Senses.GetByID(ctx, senseID)
		if err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("get sense by ID: %w", err)
		}

		// Создаем примеры
		if err := s.createExamples(ctx, senseID, input.Examples); err != nil {
			return fmt.Errorf("create examples: %w", err)
		}

		// Получаем созданные примеры для детального аудита
		createdExamples, err := s.repos.Examples.ListBySenseIDs(ctx, []uuid.UUID{senseID})
		if err == nil && len(createdExamples) > 0 {
			// Создаем аудит-лог для каждого созданного примера
			for _, example := range createdExamples {
				exampleChanges := buildCreateChanges(&example)
				if err := s.createAuditLogForEntity(ctx, model.EntityExample, example.ID, model.ActionCreate, exampleChanges); err != nil {
					return fmt.Errorf("create audit log for example: %w", err)
				}
			}
		}

		// Создаем аудит-лог для Entry (добавлены примеры)
		changes := model.JSON{
			types.AuditFieldAction:        types.AuditActionExamplesAdded,
			types.AuditFieldSenseID:       senseID.String(),
			types.AuditFieldExamplesCount: len(input.Examples),
		}
		if err := s.createAuditLog(ctx, sense.EntryID, model.ActionUpdate, changes); err != nil {
			return fmt.Errorf("create audit log: %w", err)
		}

		return nil
	})
}

// addTranslationsTx выполняет логику добавления переводов внутри транзакции.
func (s *Service) addTranslationsTx(ctx context.Context, input AddTranslationsInput, senseID uuid.UUID) error {
	return s.tx.RunInTx(ctx, func(ctx context.Context, _ database.Querier) error {
		// Проверяем существование смысла
		sense, err := s.repos.Senses.GetByID(ctx, senseID)
		if err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("get sense by ID: %w", err)
		}

		// Создаем переводы
		if err := s.createTranslations(ctx, senseID, input.Translations); err != nil {
			return fmt.Errorf("create translations: %w", err)
		}

		// Получаем созданные переводы для детального аудита
		createdTranslations, err := s.repos.Translations.ListBySenseIDs(ctx, []uuid.UUID{senseID})
		translationDetails := make([]model.JSON, 0)
		if err == nil && len(createdTranslations) > 0 {
			// Записываем информацию о переводах в аудит Entry
			// (Translation не имеет отдельного EntityType, поэтому не создаем отдельный аудит)
			for _, tr := range createdTranslations {
				translationDetails = append(translationDetails, model.JSON{
					"translation_id": tr.ID.String(),
					"text":           tr.Text,
					"source_slug":    tr.SourceSlug,
				})
			}
		}

		// Создаем аудит-лог для Entry (добавлены переводы)
		changes := model.JSON{
			types.AuditFieldAction:            types.AuditActionTranslationsAdded,
			types.AuditFieldSenseID:           senseID.String(),
			types.AuditFieldTranslationsCount: len(input.Translations),
		}
		if len(translationDetails) > 0 {
			changes[types.AuditFieldTranslations] = translationDetails
		}
		if err := s.createAuditLog(ctx, sense.EntryID, model.ActionUpdate, changes); err != nil {
			return fmt.Errorf("create audit log: %w", err)
		}

		return nil
	})
}

// addImagesTx выполняет логику добавления изображений внутри транзакции.
func (s *Service) addImagesTx(ctx context.Context, input AddImagesInput, entryID uuid.UUID) error {
	return s.tx.RunInTx(ctx, func(ctx context.Context, _ database.Querier) error {
		// Проверяем существование записи
		_, err := s.repos.Dictionary.GetByID(ctx, entryID)
		if err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("get entry by ID: %w", err)
		}

		// Создаем изображения
		if err := s.createImages(ctx, entryID, input.Images); err != nil {
			return fmt.Errorf("create images: %w", err)
		}

		// Получаем созданные изображения для детального аудита
		createdImages, err := s.repos.Images.ListByEntryIDs(ctx, []uuid.UUID{entryID})
		if err == nil && len(createdImages) > 0 {
			// Создаем аудит-лог для каждого созданного изображения
			for _, image := range createdImages {
				imageChanges := buildCreateChanges(&image)
				if err := s.createAuditLogForEntity(ctx, model.EntityImage, image.ID, model.ActionCreate, imageChanges); err != nil {
					return fmt.Errorf("create audit log for image: %w", err)
				}
			}
		}

		// Создаем аудит-лог для Entry (добавлены изображения)
		changes := model.JSON{
			types.AuditFieldAction:      types.AuditActionImagesAdded,
			types.AuditFieldImagesCount: len(input.Images),
		}
		if err := s.createAuditLog(ctx, entryID, model.ActionUpdate, changes); err != nil {
			return fmt.Errorf("create audit log: %w", err)
		}

		return nil
	})
}

// addPronunciationsTx выполняет логику добавления произношений внутри транзакции.
func (s *Service) addPronunciationsTx(ctx context.Context, input AddPronunciationsInput, entryID uuid.UUID) error {
	return s.tx.RunInTx(ctx, func(ctx context.Context, _ database.Querier) error {
		// Проверяем существование записи
		_, err := s.repos.Dictionary.GetByID(ctx, entryID)
		if err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("get entry by ID: %w", err)
		}

		// Создаем произношения
		if err := s.createPronunciations(ctx, entryID, input.Pronunciations); err != nil {
			return fmt.Errorf("create pronunciations: %w", err)
		}

		// Получаем созданные произношения для детального аудита
		createdPronunciations, err := s.repos.Pronunciations.ListByEntryIDs(ctx, []uuid.UUID{entryID})
		if err == nil && len(createdPronunciations) > 0 {
			// Создаем аудит-лог для каждого созданного произношения
			for _, pronunciation := range createdPronunciations {
				pronunciationChanges := buildCreateChanges(&pronunciation)
				if err := s.createAuditLogForEntity(ctx, model.EntityPronunciation, pronunciation.ID, model.ActionCreate, pronunciationChanges); err != nil {
					return fmt.Errorf("create audit log for pronunciation: %w", err)
				}
			}
		}

		// Создаем аудит-лог для Entry (добавлены произношения)
		changes := model.JSON{
			types.AuditFieldAction:              types.AuditActionPronunciationsAdded,
			types.AuditFieldPronunciationsCount: len(input.Pronunciations),
		}
		if err := s.createAuditLog(ctx, entryID, model.ActionUpdate, changes); err != nil {
			return fmt.Errorf("create audit log: %w", err)
		}

		return nil
	})
}

// addTopicsTx выполняет логику добавления топиков внутри транзакции.
func (s *Service) addTopicsTx(ctx context.Context, input AddTopicsInput, entryID uuid.UUID) error {
	return s.tx.RunInTx(ctx, func(ctx context.Context, _ database.Querier) error {
		// Проверяем существование записи
		_, err := s.repos.Dictionary.GetByID(ctx, entryID)
		if err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("get entry by ID: %w", err)
		}

		// Добавляем топики
		addedTopicIDs := make([]string, 0, len(input.TopicIDs))
		for _, topicIDStr := range input.TopicIDs {
			topicID, err := uuid.Parse(topicIDStr)
			if err != nil {
				return types.NewValidationError("topicIDs", fmt.Sprintf("invalid topic id: %s", topicIDStr))
			}

			// Проверяем существование топика
			_, err = s.repos.Topics.GetByID(ctx, topicID)
			if err != nil {
				if database.IsNotFoundError(err) {
					return types.NewValidationError("topicIDs", fmt.Sprintf("topic not found: %s", topicIDStr))
				}
				return fmt.Errorf("get topic %s: %w", topicID, err)
			}

			// Привязываем топик (BindToEntry использует ON CONFLICT DO NOTHING, так что дубликаты игнорируются)
			if err := s.repos.Topics.BindToEntry(ctx, entryID, topicID); err != nil {
				return fmt.Errorf("bind topic %s: %w", topicID, err)
			}

			addedTopicIDs = append(addedTopicIDs, topicIDStr)
		}

		// Создаем аудит-лог для Entry (добавлены топики)
		changes := model.JSON{
			types.AuditFieldAction:      types.AuditActionTopicsAdded,
			types.AuditFieldTopicsCount: len(addedTopicIDs),
			types.AuditFieldTopics:      addedTopicIDs,
		}
		if err := s.createAuditLog(ctx, entryID, model.ActionUpdate, changes); err != nil {
			return fmt.Errorf("create audit log: %w", err)
		}

		return nil
	})
}

// removeTopicsTx выполняет логику удаления топиков внутри транзакции.
func (s *Service) removeTopicsTx(ctx context.Context, input RemoveTopicsInput, entryID uuid.UUID) error {
	return s.tx.RunInTx(ctx, func(ctx context.Context, _ database.Querier) error {
		// Проверяем существование записи
		_, err := s.repos.Dictionary.GetByID(ctx, entryID)
		if err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("get entry by ID: %w", err)
		}

		// Удаляем топики
		removedTopicIDs := make([]string, 0, len(input.TopicIDs))
		for _, topicIDStr := range input.TopicIDs {
			topicID, err := uuid.Parse(topicIDStr)
			if err != nil {
				return types.NewValidationError("topicIDs", fmt.Sprintf("invalid topic id: %s", topicIDStr))
			}

			// Отвязываем топик
			if err := s.repos.Topics.UnbindFromEntry(ctx, entryID, topicID); err != nil {
				// Игнорируем ошибку, если топик не был привязан
				if !database.IsNotFoundError(err) {
					return fmt.Errorf("unbind topic %s: %w", topicID, err)
				}
			}

			removedTopicIDs = append(removedTopicIDs, topicIDStr)
		}

		// Создаем аудит-лог для Entry (удалены топики)
		if len(removedTopicIDs) > 0 {
			changes := model.JSON{
				types.AuditFieldAction:      types.AuditActionTopicsRemoved,
				types.AuditFieldTopicsCount: len(removedTopicIDs),
				types.AuditFieldTopics:      removedTopicIDs,
			}
			if err := s.createAuditLog(ctx, entryID, model.ActionUpdate, changes); err != nil {
				return fmt.Errorf("create audit log: %w", err)
			}
		}

		return nil
	})
}
