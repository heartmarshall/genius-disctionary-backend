package card

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database"
	"github.com/heartmarshall/my-english/internal/model"
	"github.com/heartmarshall/my-english/internal/service/types"
)

const (
	// DefaultEaseFactor — значение ease factor по умолчанию для новых карточек.
	DefaultEaseFactor = 2.5
)

// createCardTx выполняет логику создания карточки внутри транзакции.
func (s *Service) createCardTx(ctx context.Context, input CreateCardInput, entryID uuid.UUID) (*model.Card, error) {
	var createdCard *model.Card

	err := s.tx.RunInTx(ctx, func(ctx context.Context, _ database.Querier) error {
		// Проверяем существование записи словаря
		_, err := s.repos.Dictionary.GetByID(ctx, entryID)
		if err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("get entry by ID: %w", err)
		}

		// Проверяем, не существует ли уже карточка для этой записи
		existingCard, err := s.repos.Cards.GetByEntryID(ctx, entryID)
		if err != nil && !database.IsNotFoundError(err) {
			return fmt.Errorf("check existing card: %w", err)
		}
		if existingCard != nil {
			return types.ErrAlreadyExists
		}

		// Подготавливаем данные для создания карточки
		status := model.StatusNew
		if input.Status != nil {
			status = *input.Status
		}

		intervalDays := 0
		if input.IntervalDays != nil {
			intervalDays = *input.IntervalDays
		}

		easeFactor := DefaultEaseFactor
		if input.EaseFactor != nil {
			easeFactor = *input.EaseFactor
		}

		card := &model.Card{
			EntryID:      entryID,
			Status:       status,
			NextReviewAt: input.NextReviewAt,
			IntervalDays: intervalDays,
			EaseFactor:   easeFactor,
		}

		// Создаем карточку (репозиторий применит дефолтные значения если нужно)
		// TODO: дефолтные значения - это БИЗНЕС ЛОГИКА. НЕ НАДО перекладывать это на репозитории
		createdCard, err = s.repos.Cards.Create(ctx, card)
		if err != nil {
			if database.IsDuplicateError(err) {
				return types.ErrAlreadyExists
			}
			return fmt.Errorf("create card: %w", err)
		}

		// Создаем аудит-лог с полной информацией о созданной карточке
		changes := buildCreateChanges(createdCard)
		if err := s.createAuditLog(ctx, createdCard.ID, model.ActionCreate, changes); err != nil {
			return fmt.Errorf("create audit log: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return createdCard, nil
}
