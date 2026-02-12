package dataloader

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database/repository"
	"github.com/heartmarshall/my-english/internal/model"
)

// ============================================================================
// 1:N FETCHERS (Slice Fetchers)
// Общий паттерн:
// 1. Получить плоский список из БД по списку ID.
// 2. Сгруппировать по ID родителя в map.
// 3. Расставить по порядку входных ключей.
// ============================================================================

func newSensesByEntryIDFetcher(repo repository.SenseRepository, logger *slog.Logger) func(context.Context, []uuid.UUID) ([]([]model.Sense), []error) {
	return func(ctx context.Context, keys []uuid.UUID) ([]([]model.Sense), []error) {
		// 1. Запрос в БД
		items, err := repo.ListByEntryIDs(ctx, keys)
		if err != nil {
			// Логируем ошибку, если logger доступен
			if logger != nil {
				logger.Error("failed to fetch senses",
					slog.Int("count", len(keys)),
					slog.Any("error", err),
				)
			}
			// Если ошибка БД, возвращаем её для каждого ключа
			errors := make([]error, len(keys))
			for i := range errors {
				errors[i] = fmt.Errorf("fetch senses: %w", err)
			}
			return nil, errors
		}

		// 2. Группировка
		grouped := make(map[uuid.UUID][]model.Sense, len(keys))
		for _, item := range items {
			grouped[item.EntryID] = append(grouped[item.EntryID], item)
		}

		// 3. Сортировка по порядку ключей
		result := make([]([]model.Sense), len(keys))
		for i, key := range keys {
			// Если данных нет, вернется nil (пустой слайс), что корректно
			result[i] = grouped[key]
		}

		return result, nil
	}
}

func newImagesByEntryIDFetcher(repo repository.ImageRepository, logger *slog.Logger) func(context.Context, []uuid.UUID) ([]([]model.Image), []error) {
	return func(ctx context.Context, keys []uuid.UUID) ([]([]model.Image), []error) {
		items, err := repo.ListByEntryIDs(ctx, keys)
		if err != nil {
			if logger != nil {
				logger.Error("failed to fetch images",
					slog.Int("count", len(keys)),
					slog.Any("error", err),
				)
			}
			errors := make([]error, len(keys))
			for i := range errors {
				errors[i] = fmt.Errorf("fetch images: %w", err)
			}
			return nil, errors
		}

		grouped := make(map[uuid.UUID][]model.Image, len(keys))
		for _, item := range items {
			grouped[item.EntryID] = append(grouped[item.EntryID], item)
		}

		result := make([]([]model.Image), len(keys))
		for i, key := range keys {
			result[i] = grouped[key]
		}

		return result, nil
	}
}

func newPronunciationsByEntryIDFetcher(repo repository.PronunciationRepository, logger *slog.Logger) func(context.Context, []uuid.UUID) ([]([]model.Pronunciation), []error) {
	return func(ctx context.Context, keys []uuid.UUID) ([]([]model.Pronunciation), []error) {
		items, err := repo.ListByEntryIDs(ctx, keys)
		if err != nil {
			if logger != nil {
				logger.Error("failed to fetch pronunciations",
					slog.Int("count", len(keys)),
					slog.Any("error", err),
				)
			}
			errors := make([]error, len(keys))
			for i := range errors {
				errors[i] = fmt.Errorf("fetch pronunciations: %w", err)
			}
			return nil, errors
		}

		grouped := make(map[uuid.UUID][]model.Pronunciation, len(keys))
		for _, item := range items {
			grouped[item.EntryID] = append(grouped[item.EntryID], item)
		}

		result := make([]([]model.Pronunciation), len(keys))
		for i, key := range keys {
			result[i] = grouped[key]
		}

		return result, nil
	}
}

func newExamplesBySenseIDFetcher(repo repository.ExampleRepository, logger *slog.Logger) func(context.Context, []uuid.UUID) ([]([]model.Example), []error) {
	return func(ctx context.Context, keys []uuid.UUID) ([]([]model.Example), []error) {
		items, err := repo.ListBySenseIDs(ctx, keys)
		if err != nil {
			if logger != nil {
				logger.Error("failed to fetch examples",
					slog.Int("count", len(keys)),
					slog.Any("error", err),
				)
			}
			errors := make([]error, len(keys))
			for i := range errors {
				errors[i] = fmt.Errorf("fetch examples: %w", err)
			}
			return nil, errors
		}

		grouped := make(map[uuid.UUID][]model.Example, len(keys))
		for _, item := range items {
			grouped[item.SenseID] = append(grouped[item.SenseID], item)
		}

		result := make([]([]model.Example), len(keys))
		for i, key := range keys {
			result[i] = grouped[key]
		}

		return result, nil
	}
}

func newTranslationsBySenseIDFetcher(repo repository.TranslationRepository, logger *slog.Logger) func(context.Context, []uuid.UUID) ([]([]model.Translation), []error) {
	return func(ctx context.Context, keys []uuid.UUID) ([]([]model.Translation), []error) {
		items, err := repo.ListBySenseIDs(ctx, keys)
		if err != nil {
			if logger != nil {
				logger.Error("failed to fetch translations",
					slog.Int("count", len(keys)),
					slog.Any("error", err),
				)
			}
			errors := make([]error, len(keys))
			for i := range errors {
				errors[i] = fmt.Errorf("fetch translations: %w", err)
			}
			return nil, errors
		}

		grouped := make(map[uuid.UUID][]model.Translation, len(keys))
		for _, item := range items {
			grouped[item.SenseID] = append(grouped[item.SenseID], item)
		}

		result := make([]([]model.Translation), len(keys))
		for i, key := range keys {
			result[i] = grouped[key]
		}

		return result, nil
	}
}

// ============================================================================
// 1:1 FETCHERS (Pointer Fetchers)
// Паттерн для nullable связей (Card может не быть у слова).
// ============================================================================

func newCardByEntryIDFetcher(repo repository.CardRepository, logger *slog.Logger) func(context.Context, []uuid.UUID) ([]*model.Card, []error) {
	return func(ctx context.Context, keys []uuid.UUID) ([]*model.Card, []error) {
		items, err := repo.ListByEntryIDs(ctx, keys)
		if err != nil {
			if logger != nil {
				logger.Error("failed to fetch cards",
					slog.Int("count", len(keys)),
					slog.Any("error", err),
				)
			}
			errors := make([]error, len(keys))
			for i := range errors {
				errors[i] = fmt.Errorf("fetch cards: %w", err)
			}
			return nil, errors
		}

		// Маппим по EntryID
		mapped := make(map[uuid.UUID]*model.Card, len(items))
		for i := range items {
			// Берем указатель на элемент массива, так как items живет в рамках функции
			mapped[items[i].EntryID] = &items[i]
		}

		result := make([]*model.Card, len(keys))
		for i, key := range keys {
			// Если карты нет, mapped[key] вернет nil, что и нужно
			result[i] = mapped[key]
		}

		return result, nil
	}
}

func newTopicsByEntryIDFetcher(repo repository.TopicRepository, logger *slog.Logger) func(context.Context, []uuid.UUID) ([]([]model.Topic), []error) {
	return func(ctx context.Context, keys []uuid.UUID) ([]([]model.Topic), []error) {
		items, err := repo.ListByEntryIDs(ctx, keys)
		if err != nil {
			if logger != nil {
				logger.Error("failed to fetch topics",
					slog.Int("count", len(keys)),
					slog.Any("error", err),
				)
			}
			errors := make([]error, len(keys))
			for i := range errors {
				errors[i] = fmt.Errorf("fetch topics: %w", err)
			}
			return nil, errors
		}

		// Группировка по EntryID
		grouped := make(map[uuid.UUID][]model.Topic, len(keys))
		for _, item := range items {
			// item имеет тип TopicWithEntryID, который содержит Topic и EntryID
			grouped[item.EntryID] = append(grouped[item.EntryID], item.Topic)
		}

		// Сортировка по порядку ключей
		result := make([]([]model.Topic), len(keys))
		for i, key := range keys {
			// Если данных нет, вернется nil (пустой слайс), что корректно
			result[i] = grouped[key]
		}

		return result, nil
	}
}
