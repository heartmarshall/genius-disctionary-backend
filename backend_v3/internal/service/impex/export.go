package impex

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database/repository/topics"
	"github.com/heartmarshall/my-english/internal/model"
	"golang.org/x/sync/errgroup"
)

// Export экспортирует весь словарь в формате JSON.
// Метод получает все записи словаря и связанные данные (senses, cards, images и т.д.)
// и возвращает JSON bytes.
func (s *Service) Export(ctx context.Context) ([]byte, error) {
	// Получаем все записи словаря
	entries, err := s.repos.Dictionary.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all entries: %w", err)
	}

	if len(entries) == 0 {
		// Возвращаем пустой массив
		return []byte("[]"), nil
	}

	// Собираем ID всех слов
	entryIDs := make([]uuid.UUID, len(entries))
	for i, entry := range entries {
		entryIDs[i] = entry.ID
	}

	// Загружаем связанные данные параллельно через errgroup
	var (
		senses            []model.Sense
		cards             []model.Card
		images            []model.Image
		pronunciations    []model.Pronunciation
		topicsWithEntryID []topics.TopicWithEntryID
	)

	g, gctx := errgroup.WithContext(ctx)

	// Загружаем senses
	g.Go(func() error {
		var err error
		senses, err = s.repos.Senses.ListByEntryIDs(gctx, entryIDs)
		if err != nil {
			return fmt.Errorf("list senses: %w", err)
		}
		return nil
	})

	// Загружаем cards
	g.Go(func() error {
		var err error
		cards, err = s.repos.Cards.ListByEntryIDs(gctx, entryIDs)
		if err != nil {
			return fmt.Errorf("list cards: %w", err)
		}
		return nil
	})

	// Загружаем images
	g.Go(func() error {
		var err error
		images, err = s.repos.Images.ListByEntryIDs(gctx, entryIDs)
		if err != nil {
			return fmt.Errorf("list images: %w", err)
		}
		return nil
	})

	// Загружаем pronunciations
	g.Go(func() error {
		var err error
		pronunciations, err = s.repos.Pronunciations.ListByEntryIDs(gctx, entryIDs)
		if err != nil {
			return fmt.Errorf("list pronunciations: %w", err)
		}
		return nil
	})

	// Загружаем topics (через ListByEntryIDs, который возвращает TopicWithEntryID)
	g.Go(func() error {
		var err error
		topicsWithEntryID, err = s.repos.Topics.ListByEntryIDs(gctx, entryIDs)
		if err != nil {
			return fmt.Errorf("list topics: %w", err)
		}
		return nil
	})

	// Ждем завершения всех горутин
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Группируем данные по entryID для быстрого доступа
	sensesByEntryID := make(map[uuid.UUID][]model.Sense)
	for _, sense := range senses {
		sensesByEntryID[sense.EntryID] = append(sensesByEntryID[sense.EntryID], sense)
	}

	cardsByEntryID := make(map[uuid.UUID]*model.Card)
	for i := range cards {
		cardsByEntryID[cards[i].EntryID] = &cards[i]
	}

	imagesByEntryID := make(map[uuid.UUID][]model.Image)
	for _, img := range images {
		imagesByEntryID[img.EntryID] = append(imagesByEntryID[img.EntryID], img)
	}

	pronunciationsByEntryID := make(map[uuid.UUID][]model.Pronunciation)
	for _, pron := range pronunciations {
		pronunciationsByEntryID[pron.EntryID] = append(pronunciationsByEntryID[pron.EntryID], pron)
	}

	// Группируем topics по entryID
	topicsByEntryID := make(map[uuid.UUID][]string) // Map entryID -> []topicName
	for _, tw := range topicsWithEntryID {
		topicsByEntryID[tw.EntryID] = append(topicsByEntryID[tw.EntryID], tw.Name)
	}

	// Загружаем translations и examples для всех senses
	senseIDs := make([]uuid.UUID, 0, len(senses))
	for _, sense := range senses {
		senseIDs = append(senseIDs, sense.ID)
	}

	var (
		translations []model.Translation
		examples     []model.Example
	)

	g2, gctx2 := errgroup.WithContext(ctx)

	g2.Go(func() error {
		var err error
		translations, err = s.repos.Translations.ListBySenseIDs(gctx2, senseIDs)
		if err != nil {
			return fmt.Errorf("list translations: %w", err)
		}
		return nil
	})

	g2.Go(func() error {
		var err error
		examples, err = s.repos.Examples.ListBySenseIDs(gctx2, senseIDs)
		if err != nil {
			return fmt.Errorf("list examples: %w", err)
		}
		return nil
	})

	if err := g2.Wait(); err != nil {
		return nil, err
	}

	// Группируем translations и examples по senseID
	translationsBySenseID := make(map[uuid.UUID][]model.Translation)
	for _, tr := range translations {
		translationsBySenseID[tr.SenseID] = append(translationsBySenseID[tr.SenseID], tr)
	}

	examplesBySenseID := make(map[uuid.UUID][]model.Example)
	for _, ex := range examples {
		examplesBySenseID[ex.SenseID] = append(examplesBySenseID[ex.SenseID], ex)
	}

	// Собираем дерево объектов ExportEntry
	exportEntries := make([]ExportEntry, 0, len(entries))
	for _, entry := range entries {
		exportEntry := ExportEntry{
			Entry: &entry,
		}

		// Добавляем senses с переводами и примерами
		if entrySenses, ok := sensesByEntryID[entry.ID]; ok {
			exportSenses := make([]ExportSense, 0, len(entrySenses))
			for _, sense := range entrySenses {
				exportSense := ExportSense{
					Sense: &sense,
				}
				if trs, ok := translationsBySenseID[sense.ID]; ok {
					exportSense.Translations = trs
				}
				if exs, ok := examplesBySenseID[sense.ID]; ok {
					exportSense.Examples = exs
				}
				exportSenses = append(exportSenses, exportSense)
			}
			exportEntry.Senses = exportSenses
		}

		// Добавляем images
		if imgs, ok := imagesByEntryID[entry.ID]; ok {
			exportEntry.Images = imgs
		}

		// Добавляем pronunciations
		if prons, ok := pronunciationsByEntryID[entry.ID]; ok {
			exportEntry.Pronunciations = prons
		}

		// Добавляем card (если есть)
		if card, ok := cardsByEntryID[entry.ID]; ok {
			exportEntry.Card = &ExportCard{Card: card}
		}

		// Добавляем topic names
		if topicNames, ok := topicsByEntryID[entry.ID]; ok {
			exportEntry.TopicNames = topicNames
		}

		exportEntries = append(exportEntries, exportEntry)
	}

	// Сериализуем в JSON
	jsonBytes, err := json.Marshal(exportEntries)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}

	return jsonBytes, nil
}
