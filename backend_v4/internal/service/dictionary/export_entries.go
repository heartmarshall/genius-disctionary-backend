package dictionary

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// ---------------------------------------------------------------------------
// 13. ExportEntries
// ---------------------------------------------------------------------------

// ExportEntries exports all dictionary entries for the user.
func (s *Service) ExportEntries(ctx context.Context) (*ExportResult, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	// Find all entries.
	entries, _, err := s.entries.Find(ctx, userID, domain.EntryFilter{
		SortBy:    "created_at",
		SortOrder: "ASC",
		Limit:     s.cfg.ExportMaxEntries,
	})
	if err != nil {
		return nil, fmt.Errorf("find entries for export: %w", err)
	}

	if len(entries) == 0 {
		return &ExportResult{
			Items:      []ExportItem{},
			ExportedAt: time.Now(),
		}, nil
	}

	// Collect entry IDs.
	entryIDs := make([]uuid.UUID, len(entries))
	for i, e := range entries {
		entryIDs[i] = e.ID
	}

	// Batch load senses.
	senses, err := s.senses.GetByEntryIDs(ctx, entryIDs)
	if err != nil {
		return nil, fmt.Errorf("get senses: %w", err)
	}

	// Collect sense IDs and build sense-by-entry map.
	sensesByEntry := make(map[uuid.UUID][]domain.Sense)
	var senseIDs []uuid.UUID
	for _, sense := range senses {
		sensesByEntry[sense.EntryID] = append(sensesByEntry[sense.EntryID], sense)
		senseIDs = append(senseIDs, sense.ID)
	}

	// Batch load translations and examples.
	var translations []domain.Translation
	var examples []domain.Example
	if len(senseIDs) > 0 {
		translations, err = s.translations.GetBySenseIDs(ctx, senseIDs)
		if err != nil {
			return nil, fmt.Errorf("get translations: %w", err)
		}
		examples, err = s.examples.GetBySenseIDs(ctx, senseIDs)
		if err != nil {
			return nil, fmt.Errorf("get examples: %w", err)
		}
	}

	translationsBySense := make(map[uuid.UUID][]domain.Translation)
	for _, tr := range translations {
		translationsBySense[tr.SenseID] = append(translationsBySense[tr.SenseID], tr)
	}

	examplesBySense := make(map[uuid.UUID][]domain.Example)
	for _, ex := range examples {
		examplesBySense[ex.SenseID] = append(examplesBySense[ex.SenseID], ex)
	}

	// Batch load cards.
	cards, err := s.cards.GetByEntryIDs(ctx, userID, entryIDs)
	if err != nil {
		return nil, fmt.Errorf("get cards: %w", err)
	}
	cardByEntry := make(map[uuid.UUID]domain.Card)
	for _, c := range cards {
		cardByEntry[c.EntryID] = c
	}

	// Build export items.
	items := make([]ExportItem, 0, len(entries))
	for _, entry := range entries {
		item := ExportItem{
			Text:      entry.Text,
			Notes:     entry.Notes,
			CreatedAt: entry.CreatedAt,
		}

		// Card status.
		if card, found := cardByEntry[entry.ID]; found {
			status := card.State
			item.CardStatus = &status
		}

		// Senses.
		for _, sense := range sensesByEntry[entry.ID] {
			exportSense := ExportSense{
				Definition:   sense.Definition,
				PartOfSpeech: sense.PartOfSpeech,
			}

			// Translations.
			for _, tr := range translationsBySense[sense.ID] {
				if tr.Text != nil {
					exportSense.Translations = append(exportSense.Translations, *tr.Text)
				}
			}

			// Examples.
			for _, ex := range examplesBySense[sense.ID] {
				exportEx := ExportExample{
					Translation: ex.Translation,
				}
				if ex.Sentence != nil {
					exportEx.Sentence = *ex.Sentence
				}
				exportSense.Examples = append(exportSense.Examples, exportEx)
			}

			item.Senses = append(item.Senses, exportSense)
		}

		items = append(items, item)
	}

	return &ExportResult{
		Items:      items,
		ExportedAt: time.Now(),
	}, nil
}
