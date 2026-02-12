package dictionary

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database"
	"github.com/heartmarshall/my-english/internal/model"
	"github.com/heartmarshall/my-english/internal/service/types"
)

// updateWordTx выполняет логику обновления слова внутри транзакции.
func (s *Service) updateWordTx(ctx context.Context, input UpdateWordInput, entryID uuid.UUID) (*model.DictionaryEntry, error) {
	var updatedEntry *model.DictionaryEntry

	err := s.tx.RunInTx(ctx, func(ctx context.Context, _ database.Querier) error {
		// Получаем существующую запись
		existingEntry, err := s.repos.Dictionary.GetByID(ctx, entryID)
		if err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("get entry by ID: %w", err)
		}

		// Определяем текст для обновления
		textRaw := existingEntry.Text
		textNorm := existingEntry.TextNormalized

		if input.Text != nil {
			textRaw = normalizeText(*input.Text)
			textNorm = textRaw

			// Проверяем, не существует ли уже слово с таким нормализованным текстом (кроме текущего)
			existingByText, err := s.repos.Dictionary.FindByNormalizedText(ctx, textNorm)
			if err != nil && !database.IsNotFoundError(err) {
				// TODO: нужно сделать кастомные ошибки чтобы на фронте было легче их отлавливать
				return fmt.Errorf("check duplicate text: %w", err)
			}
			if existingByText != nil && existingByText.ID != entryID {
				return types.ErrAlreadyExists
			}
		}

		// Определяем заметки для обновления
		notes := existingEntry.Notes
		if input.Notes != nil {
			notesTrimmed := strings.TrimSpace(*input.Notes)
			if notesTrimmed == "" {
				notes = nil // Пустая строка означает удаление заметок
			} else {
				notes = &notesTrimmed
			}
		}

		// Обновляем основную запись
		entry := buildDictionaryEntryWithNotes(textRaw, textNorm, notes)
		updatedEntry, err = s.repos.Dictionary.Update(ctx, entryID, entry)
		if err != nil {
			if database.IsDuplicateError(err) {
				return types.ErrAlreadyExists
			}
			return fmt.Errorf("update entry: %w", err)
		}

		// Удаляем и пересоздаем связанные сущности
		if len(input.Senses) > 0 {
			if err := s.recreateSenses(ctx, entryID, input.Senses); err != nil {
				return fmt.Errorf("recreate senses: %w", err)
			}
		}

		// Если Images передано, пересоздаем изображения
		// nil означает "не трогать" (поле не было передано в GraphQL)
		// Пустой слайс означает "удалить все" (поле было передано пустым массивом)
		// Не-nil слайс означает "заменить на новые"
		if input.Images != nil {
			if err := s.recreateImages(ctx, entryID, input.Images); err != nil {
				return fmt.Errorf("recreate images: %w", err)
			}
		}

		// Если Pronunciations передано, пересоздаем произношения
		// nil означает "не трогать" (поле не было передано в GraphQL)
		// Пустой слайс означает "удалить все" (поле было передано пустым массивом)
		// Не-nil слайс означает "заменить на новые"
		if input.Pronunciations != nil {
			if err := s.recreatePronunciations(ctx, entryID, input.Pronunciations); err != nil {
				return fmt.Errorf("recreate pronunciations: %w", err)
			}
		}

		// Если TopicIDs передано, пересоздаем топики
		// nil означает "не трогать" (поле не было передано в GraphQL)
		// Пустой слайс означает "удалить все" (поле было передано пустым массивом)
		// Не-nil слайс означает "заменить на новые"
		if input.TopicIDs != nil {
			if err := s.recreateTopics(ctx, entryID, input.TopicIDs); err != nil {
				return fmt.Errorf("recreate topics: %w", err)
			}
		}

		// Создаем аудит-лог с детальными изменениями полей
		changes := diffDictionaryEntry(existingEntry, updatedEntry)

		// Если были изменения в связанных сущностях, получаем детальную информацию
		if len(input.Senses) > 0 {
			// Получаем существующие senses для сравнения
			existingSenses, err := s.repos.Senses.ListByEntryIDs(ctx, []uuid.UUID{entryID})
			if err == nil {
				// Создаем список удаленных и созданных senses
				existingSenseMap := make(map[uuid.UUID]*model.Sense)
				for i := range existingSenses {
					existingSenseMap[existingSenses[i].ID] = &existingSenses[i]
				}

				// Здесь мы не можем точно сопоставить старые и новые senses,
				// так как они пересоздаются. Записываем факт обновления.
				changes[types.AuditFieldSensesRecreated] = true
				changes[types.AuditFieldSensesOldCount] = len(existingSenses)
				changes[types.AuditFieldSensesNewCount] = len(input.Senses)
			}
		}

		if input.Images != nil {
			existingImages, err := s.repos.Images.ListByEntryIDs(ctx, []uuid.UUID{entryID})
			if err == nil {
				changes[types.AuditFieldImagesRecreated] = true
				changes[types.AuditFieldImagesOldCount] = len(existingImages)
				changes[types.AuditFieldImagesNewCount] = len(input.Images)
			}
		}

		if len(input.Pronunciations) > 0 {
			existingPronunciations, err := s.repos.Pronunciations.ListByEntryIDs(ctx, []uuid.UUID{entryID})
			if err == nil {
				changes[types.AuditFieldPronunciationsRecreated] = true
				changes[types.AuditFieldPronunciationsOldCount] = len(existingPronunciations)
				changes[types.AuditFieldPronunciationsNewCount] = len(input.Pronunciations)
			}
		}

		if input.TopicIDs != nil {
			existingTopics, err := s.repos.Topics.ListByEntryIDs(ctx, []uuid.UUID{entryID})
			if err == nil {
				changes[types.AuditFieldTopicsRecreated] = true
				changes[types.AuditFieldTopicsOldCount] = len(existingTopics)
				changes[types.AuditFieldTopicsNewCount] = len(input.TopicIDs)
			}
		}

		// Создаем аудит-лог только если были какие-либо изменения
		if len(changes) > 0 {
			if err := s.createAuditLog(ctx, entryID, model.ActionUpdate, changes); err != nil {
				return fmt.Errorf("create audit log: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return updatedEntry, nil
}

// recreateSenses удаляет существующие смыслы и создает новые.
// CASCADE удаление автоматически удалит связанные переводы и примеры.
func (s *Service) recreateSenses(ctx context.Context, entryID uuid.UUID, senses []SenseInput) error {
	existingSenses, err := s.repos.Senses.ListByEntryIDs(ctx, []uuid.UUID{entryID})
	if err != nil {
		return fmt.Errorf("list existing senses: %w", err)
	}

	// Удаляем существующие смыслы (CASCADE удалит переводы и примеры)
	for _, sense := range existingSenses {
		if err := s.repos.Senses.Delete(ctx, sense.ID); err != nil {
			return fmt.Errorf("delete sense %s: %w", sense.ID, err)
		}
	}

	// Создаем новые смыслы
	if err := s.createSenses(ctx, entryID, senses); err != nil {
		return fmt.Errorf("create new senses: %w", err)
	}

	return nil
}

// recreateImages удаляет существующие изображения и создает новые.
func (s *Service) recreateImages(ctx context.Context, entryID uuid.UUID, images []ImageInput) error {
	existingImages, err := s.repos.Images.ListByEntryIDs(ctx, []uuid.UUID{entryID})
	if err != nil {
		return fmt.Errorf("list existing images: %w", err)
	}

	// Удаляем существующие изображения
	for _, img := range existingImages {
		if err := s.repos.Images.Delete(ctx, img.ID); err != nil {
			return fmt.Errorf("delete image %s: %w", img.ID, err)
		}
	}

	// Создаем новые изображения
	if err := s.createImages(ctx, entryID, images); err != nil {
		return fmt.Errorf("create new images: %w", err)
	}

	return nil
}

// recreatePronunciations удаляет существующие произношения и создает новые.
func (s *Service) recreatePronunciations(ctx context.Context, entryID uuid.UUID, pronunciations []PronunciationInput) error {
	existingPronunciations, err := s.repos.Pronunciations.ListByEntryIDs(ctx, []uuid.UUID{entryID})
	if err != nil {
		return fmt.Errorf("list existing pronunciations: %w", err)
	}

	// Удаляем существующие произношения
	for _, pron := range existingPronunciations {
		if err := s.repos.Pronunciations.Delete(ctx, pron.ID); err != nil {
			return fmt.Errorf("delete pronunciation %s: %w", pron.ID, err)
		}
	}

	// Создаем новые произношения
	if err := s.createPronunciations(ctx, entryID, pronunciations); err != nil {
		return fmt.Errorf("create new pronunciations: %w", err)
	}

	return nil
}

// recreateTopics удаляет существующие топики и создает новые связи.
func (s *Service) recreateTopics(ctx context.Context, entryID uuid.UUID, topicIDs []string) error {
	// Удаляем все существующие связи с топиками
	if err := s.repos.Topics.UnbindAllFromEntry(ctx, entryID); err != nil {
		return fmt.Errorf("unbind all topics: %w", err)
	}

	// Создаем новые связи с топиками
	for _, topicIDStr := range topicIDs {
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

		if err := s.repos.Topics.BindToEntry(ctx, entryID, topicID); err != nil {
			return fmt.Errorf("bind topic %s: %w", topicID, err)
		}
	}

	return nil
}
