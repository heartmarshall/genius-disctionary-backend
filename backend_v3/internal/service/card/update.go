package card

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database"
	"github.com/heartmarshall/my-english/internal/model"
	"github.com/heartmarshall/my-english/internal/service/types"
)

// updateCardTx выполняет логику обновления карточки внутри транзакции.
// Обновляет только те поля, которые были указаны в input.
func (s *Service) updateCardTx(ctx context.Context, input UpdateCardInput, cardID uuid.UUID) (*model.Card, error) {
	var updatedCard *model.Card

	err := s.tx.RunInTx(ctx, func(ctx context.Context, _ database.Querier) error {
		// Получаем существующую карточку
		existingCard, err := s.repos.Cards.GetByID(ctx, cardID)
		if err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("get card by ID: %w", err)
		}

		// Определяем, что нужно обновить
		status := existingCard.Status
		if input.Status != nil {
			status = *input.Status
		}

		nextReviewAt := existingCard.NextReviewAt
		// Если NextReviewAt указан в input, обновляем его
		// nil в input означает, что поле не передано, поэтому не трогаем существующее значение
		// Для сброса NextReviewAt нужно передать специальное значение (например, пустую строку в JSON)
		// или использовать отдельный флаг, но для простоты пока оставляем как есть
		// В будущем можно добавить ResetNextReviewAt bool
		if input.NextReviewAt != nil {
			nextReviewAt = input.NextReviewAt
		}

		intervalDays := existingCard.IntervalDays
		if input.IntervalDays != nil {
			intervalDays = *input.IntervalDays
		}

		easeFactor := existingCard.EaseFactor
		if input.EaseFactor != nil {
			easeFactor = *input.EaseFactor
		}

		// Проверяем, были ли изменения
		hasChanges := status != existingCard.Status ||
			!equalTimePtr(nextReviewAt, existingCard.NextReviewAt) ||
			intervalDays != existingCard.IntervalDays ||
			easeFactor != existingCard.EaseFactor

		if !hasChanges {
			// Нет изменений, возвращаем существующую карточку
			updatedCard = existingCard
			return nil
		}

		// Обновляем карточку
		card := &model.Card{
			ID:           existingCard.ID,
			EntryID:      existingCard.EntryID,
			Status:       status,
			NextReviewAt: nextReviewAt,
			IntervalDays: intervalDays,
			EaseFactor:   easeFactor,
		}

		updatedCard, err = s.repos.Cards.Update(ctx, cardID, card)
		if err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("update card: %w", err)
		}

		// Создаем аудит-лог с детальными изменениями полей
		changes := diffCard(existingCard, updatedCard)
		if len(changes) > 0 {
			if err := s.createAuditLog(ctx, cardID, model.ActionUpdate, changes); err != nil {
				return fmt.Errorf("create audit log: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return updatedCard, nil
}
