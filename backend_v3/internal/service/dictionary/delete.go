package dictionary

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database"
	"github.com/heartmarshall/my-english/internal/model"
	"github.com/heartmarshall/my-english/internal/service/types"
)

// deleteWordTx выполняет логику удаления слова внутри транзакции.
func (s *Service) deleteWordTx(ctx context.Context, entryID uuid.UUID) error {
	err := s.tx.RunInTx(ctx, func(ctx context.Context, _ database.Querier) error {
		// Получаем существующую запись для аудита
		existingEntry, err := s.repos.Dictionary.GetByID(ctx, entryID)
		if err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("get entry by ID: %w", err)
		}

		// Удаляем запись (CASCADE удалит связанные данные)
		if err := s.repos.Dictionary.Delete(ctx, entryID); err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("delete entry: %w", err)
		}

		// Создаем аудит-лог с полной информацией об удаленной сущности
		changes := buildDeleteChanges(existingEntry)
		if err := s.createAuditLog(ctx, entryID, model.ActionDelete, changes); err != nil {
			return fmt.Errorf("create audit log: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// deleteSenseTx выполняет логику удаления смысла внутри транзакции.
// CASCADE удаление автоматически удалит связанные переводы и примеры.
func (s *Service) deleteSenseTx(ctx context.Context, senseID uuid.UUID) error {
	return s.tx.RunInTx(ctx, func(ctx context.Context, _ database.Querier) error {
		// Получаем смысл для аудита
		sense, err := s.repos.Senses.GetByID(ctx, senseID)
		if err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("get sense by ID: %w", err)
		}

		// Удаляем смысл (CASCADE удалит переводы и примеры)
		if err := s.repos.Senses.Delete(ctx, senseID); err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("delete sense: %w", err)
		}

		// Создаем аудит-лог для Entry (удален sense)
		changes := model.JSON{
			types.AuditFieldAction: types.AuditActionSenseDeleted,
			types.AuditFieldSenseID: senseID.String(),
		}
		if sense.Definition != nil {
			changes[types.AuditFieldDefinition] = *sense.Definition
		}
		if sense.PartOfSpeech != nil {
			changes[types.AuditFieldPartOfSpeech] = *sense.PartOfSpeech
		}

		// Также создаем отдельный аудит-лог для самого sense
		senseChanges := buildDeleteChanges(sense)
		if err := s.createAuditLogForEntity(ctx, model.EntitySense, senseID, model.ActionDelete, senseChanges); err != nil {
			return fmt.Errorf("create audit log for sense: %w", err)
		}

		if err := s.createAuditLog(ctx, sense.EntryID, model.ActionUpdate, changes); err != nil {
			return fmt.Errorf("create audit log: %w", err)
		}

		return nil
	})
}

// deleteExampleTx выполняет логику удаления примера внутри транзакции.
func (s *Service) deleteExampleTx(ctx context.Context, exampleID uuid.UUID) error {
	return s.tx.RunInTx(ctx, func(ctx context.Context, _ database.Querier) error {
		// Получаем пример для аудита
		example, err := s.repos.Examples.GetByID(ctx, exampleID)
		if err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("get example by ID: %w", err)
		}

		// Получаем смысл для получения entryID
		sense, err := s.repos.Senses.GetByID(ctx, example.SenseID)
		if err != nil {
			return fmt.Errorf("get sense by ID: %w", err)
		}

		// Удаляем пример
		if err := s.repos.Examples.Delete(ctx, exampleID); err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("delete example: %w", err)
		}

		// Создаем аудит-лог для Entry (удален пример)
		changes := model.JSON{
			types.AuditFieldAction:   types.AuditActionExampleDeleted,
			types.AuditFieldExampleID: exampleID.String(),
			types.AuditFieldSentence: example.Sentence,
		}

		// Также создаем отдельный аудит-лог для самого примера
		exampleChanges := buildDeleteChanges(&example)
		if err := s.createAuditLogForEntity(ctx, model.EntityExample, exampleID, model.ActionDelete, exampleChanges); err != nil {
			return fmt.Errorf("create audit log for example: %w", err)
		}

		if err := s.createAuditLog(ctx, sense.EntryID, model.ActionUpdate, changes); err != nil {
			return fmt.Errorf("create audit log: %w", err)
		}

		return nil
	})
}

// deleteTranslationTx выполняет логику удаления перевода внутри транзакции.
func (s *Service) deleteTranslationTx(ctx context.Context, translationID uuid.UUID) error {
	return s.tx.RunInTx(ctx, func(ctx context.Context, _ database.Querier) error {
		// Получаем перевод для аудита
		translation, err := s.repos.Translations.GetByID(ctx, translationID)
		if err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("get translation by ID: %w", err)
		}

		// Получаем смысл для получения entryID
		sense, err := s.repos.Senses.GetByID(ctx, translation.SenseID)
		if err != nil {
			return fmt.Errorf("get sense by ID: %w", err)
		}

		// Удаляем перевод
		if err := s.repos.Translations.Delete(ctx, translationID); err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("delete translation: %w", err)
		}

		// Создаем аудит-лог для Entry (удален перевод)
		changes := model.JSON{
			types.AuditFieldAction:        types.AuditActionTranslationDeleted,
			types.AuditFieldTranslationID: translationID.String(),
			types.AuditFieldText:          translation.Text,
			types.AuditFieldSourceSlug:    translation.SourceSlug,
		}
		// Translation не имеет отдельного EntityType, поэтому не создаем отдельный аудит

		if err := s.createAuditLog(ctx, sense.EntryID, model.ActionUpdate, changes); err != nil {
			return fmt.Errorf("create audit log: %w", err)
		}

		return nil
	})
}

// deleteImageTx выполняет логику удаления изображения внутри транзакции.
func (s *Service) deleteImageTx(ctx context.Context, imageID uuid.UUID) error {
	return s.tx.RunInTx(ctx, func(ctx context.Context, _ database.Querier) error {
		// Получаем изображение для аудита
		image, err := s.repos.Images.GetByID(ctx, imageID)
		if err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("get image by ID: %w", err)
		}

		// Удаляем изображение
		if err := s.repos.Images.Delete(ctx, imageID); err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("delete image: %w", err)
		}

		// Создаем аудит-лог для Entry (удалено изображение)
		changes := model.JSON{
			types.AuditFieldAction: types.AuditActionImageDeleted,
			types.AuditFieldImageID: imageID.String(),
			types.AuditFieldURL:    image.URL,
		}

		// Также создаем отдельный аудит-лог для самого изображения
		imageChanges := buildDeleteChanges(&image)
		if err := s.createAuditLogForEntity(ctx, model.EntityImage, imageID, model.ActionDelete, imageChanges); err != nil {
			return fmt.Errorf("create audit log for image: %w", err)
		}

		if err := s.createAuditLog(ctx, image.EntryID, model.ActionUpdate, changes); err != nil {
			return fmt.Errorf("create audit log: %w", err)
		}

		return nil
	})
}

// deletePronunciationTx выполняет логику удаления произношения внутри транзакции.
func (s *Service) deletePronunciationTx(ctx context.Context, pronunciationID uuid.UUID) error {
	return s.tx.RunInTx(ctx, func(ctx context.Context, _ database.Querier) error {
		// Получаем произношение для аудита
		pronunciation, err := s.repos.Pronunciations.GetByID(ctx, pronunciationID)
		if err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("get pronunciation by ID: %w", err)
		}

		// Удаляем произношение
		if err := s.repos.Pronunciations.Delete(ctx, pronunciationID); err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("delete pronunciation: %w", err)
		}

		// Создаем аудит-лог для Entry (удалено произношение)
		changes := model.JSON{
			types.AuditFieldAction:          types.AuditActionPronunciationDeleted,
			types.AuditFieldPronunciationID: pronunciationID.String(),
			types.AuditFieldAudioURL:        pronunciation.AudioURL,
		}

		// Также создаем отдельный аудит-лог для самого произношения
		pronunciationChanges := buildDeleteChanges(&pronunciation)
		if err := s.createAuditLogForEntity(ctx, model.EntityPronunciation, pronunciationID, model.ActionDelete, pronunciationChanges); err != nil {
			return fmt.Errorf("create audit log for pronunciation: %w", err)
		}

		if err := s.createAuditLog(ctx, pronunciation.EntryID, model.ActionUpdate, changes); err != nil {
			return fmt.Errorf("create audit log: %w", err)
		}

		return nil
	})
}
