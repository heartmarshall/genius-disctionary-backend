package dictionary

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database"
	"github.com/heartmarshall/my-english/internal/model"
	"github.com/heartmarshall/my-english/internal/service/types"
)

// createWordTx выполняет логику создания слова внутри транзакции.
func (s *Service) createWordTx(ctx context.Context, input CreateWordInput, textRaw, textNorm string) (*model.DictionaryEntry, error) {
	var createdEntry *model.DictionaryEntry

	err := s.tx.RunInTx(ctx, func(ctx context.Context, _ database.Querier) error {
		// Проверяем дубликат
		exists, err := s.repos.Dictionary.ExistsByNormalizedText(ctx, textNorm)
		if err != nil {
			return fmt.Errorf("check duplicate: %w", err)
		}
		if exists {
			return types.ErrAlreadyExists
		}

		// Создаем основную запись (Entry)
		entry := buildDictionaryEntryWithNotes(textRaw, textNorm, input.Notes)
		createdEntry, err = s.repos.Dictionary.Create(ctx, entry)
		if err != nil {
			if database.IsDuplicateError(err) {
				return types.ErrAlreadyExists
			}
			return fmt.Errorf("create entry: %w", err)
		}

		// Создаем связанные сущности
		if err := s.createSenses(ctx, createdEntry.ID, input.Senses); err != nil {
			return fmt.Errorf("create senses: %w", err)
		}

		if err := s.createImages(ctx, createdEntry.ID, input.Images); err != nil {
			return fmt.Errorf("create images: %w", err)
		}

		if err := s.createPronunciations(ctx, createdEntry.ID, input.Pronunciations); err != nil {
			return fmt.Errorf("create pronunciations: %w", err)
		}

		if err := s.createCardIfNeeded(ctx, createdEntry.ID, input.CreateCard, input.CardOptions); err != nil {
			return fmt.Errorf("create card: %w", err)
		}

		if len(input.TopicIDs) > 0 {
			for _, tIDStr := range input.TopicIDs {
				topicID, err := uuid.Parse(tIDStr)
				if err != nil {
					return types.NewValidationError("topicIDs", fmt.Sprintf("invalid topic id: %s", tIDStr))
				}
				if err := s.repos.Topics.BindToEntry(ctx, createdEntry.ID, topicID); err != nil {
					return fmt.Errorf("bind topic %s: %w", topicID, err)
				}
			}
		}

		// Создаем аудит-лог с полной информацией о созданной сущности
		changes := buildCreateChanges(createdEntry)
		// Добавляем информацию о связанных сущностях
		if len(input.Senses) > 0 {
			changes[types.AuditFieldSensesCount] = len(input.Senses)
		}
		if len(input.Images) > 0 {
			changes[types.AuditFieldImagesCount] = len(input.Images)
		}
		if len(input.Pronunciations) > 0 {
			changes[types.AuditFieldPronunciationsCount] = len(input.Pronunciations)
		}
		if len(input.TopicIDs) > 0 {
			changes[types.AuditFieldTopicsCount] = len(input.TopicIDs)
		}
		if input.CreateCard {
			changes[types.AuditFieldCardCreated] = true
		}

		if err := s.createAuditLog(ctx, createdEntry.ID, model.ActionCreate, changes); err != nil {
			return fmt.Errorf("create audit log: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return createdEntry, nil
}
