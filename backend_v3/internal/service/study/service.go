package study

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database"
	"github.com/heartmarshall/my-english/internal/database/repository"
	"github.com/heartmarshall/my-english/internal/database/repository/cards"
	"github.com/heartmarshall/my-english/internal/model"
	"github.com/heartmarshall/my-english/internal/service/types"
)

// Service реализует бизнес-логику для работы с изучением карточек.
type Service struct {
	repos *repository.Registry
	tx    *database.TxManager
}

// NewService создает новый экземпляр сервиса изучения.
// Возвращает ошибку, если repos или tx равны nil.
func NewService(repos *repository.Registry, tx *database.TxManager) (*Service, error) {
	if repos == nil {
		return nil, fmt.Errorf("repos cannot be nil")
	}
	if tx == nil {
		return nil, fmt.Errorf("tx cannot be nil")
	}

	return &Service{
		repos: repos,
		tx:    tx,
	}, nil
}

// ReviewCardInput содержит данные для ответа на карточку.
type ReviewCardInput struct {
	CardID     uuid.UUID         // ID карточки для повторения
	Grade      model.ReviewGrade // Оценка пользователя (Again, Hard, Good, Easy)
	DurationMs *int              // Время, потраченное на ответ в миллисекундах (опционально)
	ReviewedAt time.Time         // Время повторения (обычно Now, но полезно для тестов или оффлайн-синхронизации)
}

// ReviewResult содержит результат успешного повторения карточки.
type ReviewResult struct {
	Card         model.Card      // Обновленная карточка с новыми SRS параметрами
	ReviewLog    model.ReviewLog // Запись о повторении
	NextReviewAt time.Time       // Время следующего повторения
}

// ReviewCard обрабатывает ответ пользователя на карточку.
// Метод выполняет атомарное обновление карточки с блокировкой строки (FOR UPDATE),
// что предотвращает race conditions при параллельных запросах.
// Использует алгоритм SRS (Spaced Repetition System) для расчета следующего повторения.
func (s *Service) ReviewCard(ctx context.Context, input ReviewCardInput) (*ReviewResult, error) {
	// Валидация входных данных
	if input.CardID == uuid.Nil {
		return nil, types.NewValidationError("cardID", "cannot be nil")
	}

	if input.ReviewedAt.IsZero() {
		input.ReviewedAt = time.Now()
	}

	var result ReviewResult

	err := s.tx.RunInTx(ctx, func(ctx context.Context, _ database.Querier) error {
		// Получаем карточку с блокировкой FOR UPDATE
		// Это предотвращает race conditions при параллельных запросах
		card, err := s.repos.Cards.GetByIDForUpdate(ctx, input.CardID)
		if err != nil {
			if database.IsNotFoundError(err) {
				return types.ErrNotFound
			}
			return fmt.Errorf("get card by ID for update: %w", err)
		}

		// Рассчитываем новые параметры SRS (чистая функция)
		srsCalc := calculateSRS(card, input.Grade, input.ReviewedAt)

		// Обновляем карточку
		// Используем UpdateSRSFields для оптимизации (обновляем только нужные поля)
		err = s.repos.Cards.UpdateSRSFields(
			ctx,
			card.ID,
			srsCalc.Status,
			&srsCalc.NextReviewAt,
			srsCalc.IntervalDays,
			srsCalc.EaseFactor,
		)
		if err != nil {
			return fmt.Errorf("update card SRS fields: %w", err)
		}

		// Записываем лог повторения
		logEntry := &model.ReviewLog{
			CardID:     card.ID,
			Grade:      input.Grade,
			DurationMs: input.DurationMs,
			ReviewedAt: input.ReviewedAt,
		}
		createdLog, err := s.repos.ReviewLogs.Create(ctx, logEntry)
		if err != nil {
			return fmt.Errorf("create review log: %w", err)
		}

		// Подготавливаем результат (обновляем поля в объекте card для возврата)
		card.Status = srsCalc.Status
		card.NextReviewAt = &srsCalc.NextReviewAt
		card.IntervalDays = srsCalc.IntervalDays
		card.EaseFactor = srsCalc.EaseFactor
		card.UpdatedAt = time.Now()

		result = ReviewResult{
			Card:         *card,
			ReviewLog:    *createdLog,
			NextReviewAt: srsCalc.NextReviewAt,
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("review card transaction: %w", err)
	}

	return &result, nil
}

// GetStudyQueue возвращает очередь карточек для изучения.
// Метод возвращает слова, которые пора повторять, отсортированные в порядке приоритета.
// Логика выборки инкапсулирована в репозитории.
func (s *Service) GetStudyQueue(ctx context.Context, limit int) ([]model.DictionaryEntry, error) {
	if limit <= 0 {
		return nil, types.NewValidationError("limit", "must be greater than 0")
	}

	// Получаем карточки, которые пора повторять
	cards, err := s.repos.Cards.GetDueCards(ctx, time.Now(), limit)
	if err != nil {
		return nil, fmt.Errorf("get due cards: %w", err)
	}

	if len(cards) == 0 {
		return []model.DictionaryEntry{}, nil
	}

	// Собираем ID слов
	entryIDs := make([]uuid.UUID, len(cards))
	for i, c := range cards {
		entryIDs[i] = c.EntryID
	}

	// Загружаем слова
	// Порядок в IN (...) не гарантирован, поэтому нужно отсортировать в соответствии с очередью cards
	entries, err := s.repos.Dictionary.ListByIDs(ctx, entryIDs)
	if err != nil {
		return nil, fmt.Errorf("list entries by IDs: %w", err)
	}

	// Восстанавливаем порядок из исходной очереди карточек
	entriesMap := make(map[uuid.UUID]model.DictionaryEntry, len(entries))
	for _, e := range entries {
		entriesMap[e.ID] = e
	}

	orderedEntries := make([]model.DictionaryEntry, 0, len(cards))
	for _, c := range cards {
		if e, ok := entriesMap[c.EntryID]; ok {
			orderedEntries = append(orderedEntries, e)
		}
	}

	return orderedEntries, nil
}

// GetDashboardStats возвращает статистику для дашборда изучения.
// Включает количество карточек по статусам, карточек к повторению и другую аналитику.
func (s *Service) GetDashboardStats(ctx context.Context) (*cards.DashboardStats, error) {
	stats, err := s.repos.Cards.GetDashboardStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("get dashboard stats: %w", err)
	}

	return stats, nil
}

// GetCardHistory возвращает историю повторений карточки.
func (s *Service) GetCardHistory(ctx context.Context, cardID uuid.UUID, limit int) ([]model.ReviewLog, error) {
	logs, err := s.repos.ReviewLogs.ListByCardID(ctx, cardID, limit)
	if err != nil {
		return nil, fmt.Errorf("list review logs: %w", err)
	}
	return logs, nil
}
