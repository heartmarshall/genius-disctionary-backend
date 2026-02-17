package dataloader

import (
	"context"

	"github.com/google/uuid"
	"github.com/graph-gophers/dataloader/v7"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// ---------------------------------------------------------------------------
// Senses by EntryID
// ---------------------------------------------------------------------------

func newSensesBatchFn(repo senseRepo) dataloader.BatchFunc[uuid.UUID, []domain.Sense] {
	return func(ctx context.Context, keys []uuid.UUID) []*dataloader.Result[[]domain.Sense] {
		senses, err := repo.GetByEntryIDs(ctx, keys)
		if err != nil {
			return errorResults[[]domain.Sense](len(keys), err)
		}

		grouped := make(map[uuid.UUID][]domain.Sense, len(keys))
		for _, s := range senses {
			grouped[s.EntryID] = append(grouped[s.EntryID], s)
		}

		return mapResults(keys, grouped, emptySlice[domain.Sense])
	}
}

// ---------------------------------------------------------------------------
// Translations by SenseID
// ---------------------------------------------------------------------------

func newTranslationsBatchFn(repo translationRepo) dataloader.BatchFunc[uuid.UUID, []domain.Translation] {
	return func(ctx context.Context, keys []uuid.UUID) []*dataloader.Result[[]domain.Translation] {
		translations, err := repo.GetBySenseIDs(ctx, keys)
		if err != nil {
			return errorResults[[]domain.Translation](len(keys), err)
		}

		grouped := make(map[uuid.UUID][]domain.Translation, len(keys))
		for _, tr := range translations {
			grouped[tr.SenseID] = append(grouped[tr.SenseID], tr)
		}

		return mapResults(keys, grouped, emptySlice[domain.Translation])
	}
}

// ---------------------------------------------------------------------------
// Examples by SenseID
// ---------------------------------------------------------------------------

func newExamplesBatchFn(repo exampleRepo) dataloader.BatchFunc[uuid.UUID, []domain.Example] {
	return func(ctx context.Context, keys []uuid.UUID) []*dataloader.Result[[]domain.Example] {
		examples, err := repo.GetBySenseIDs(ctx, keys)
		if err != nil {
			return errorResults[[]domain.Example](len(keys), err)
		}

		grouped := make(map[uuid.UUID][]domain.Example, len(keys))
		for _, ex := range examples {
			grouped[ex.SenseID] = append(grouped[ex.SenseID], ex)
		}

		return mapResults(keys, grouped, emptySlice[domain.Example])
	}
}

// ---------------------------------------------------------------------------
// Pronunciations by EntryID
// ---------------------------------------------------------------------------

func newPronunciationsBatchFn(repo pronunciationRepo) dataloader.BatchFunc[uuid.UUID, []domain.RefPronunciation] {
	return func(ctx context.Context, keys []uuid.UUID) []*dataloader.Result[[]domain.RefPronunciation] {
		rows, err := repo.GetByEntryIDs(ctx, keys)
		if err != nil {
			return errorResults[[]domain.RefPronunciation](len(keys), err)
		}

		grouped := make(map[uuid.UUID][]domain.RefPronunciation, len(keys))
		for _, r := range rows {
			grouped[r.EntryID] = append(grouped[r.EntryID], r.RefPronunciation)
		}

		return mapResults(keys, grouped, emptySlice[domain.RefPronunciation])
	}
}

// ---------------------------------------------------------------------------
// Catalog Images by EntryID
// ---------------------------------------------------------------------------

func newCatalogImagesBatchFn(repo imageRepo) dataloader.BatchFunc[uuid.UUID, []domain.RefImage] {
	return func(ctx context.Context, keys []uuid.UUID) []*dataloader.Result[[]domain.RefImage] {
		rows, err := repo.GetCatalogByEntryIDs(ctx, keys)
		if err != nil {
			return errorResults[[]domain.RefImage](len(keys), err)
		}

		grouped := make(map[uuid.UUID][]domain.RefImage, len(keys))
		for _, r := range rows {
			grouped[r.EntryID] = append(grouped[r.EntryID], r.RefImage)
		}

		return mapResults(keys, grouped, emptySlice[domain.RefImage])
	}
}

// ---------------------------------------------------------------------------
// User Images by EntryID
// ---------------------------------------------------------------------------

func newUserImagesBatchFn(repo imageRepo) dataloader.BatchFunc[uuid.UUID, []domain.UserImage] {
	return func(ctx context.Context, keys []uuid.UUID) []*dataloader.Result[[]domain.UserImage] {
		rows, err := repo.GetUserByEntryIDs(ctx, keys)
		if err != nil {
			return errorResults[[]domain.UserImage](len(keys), err)
		}

		grouped := make(map[uuid.UUID][]domain.UserImage, len(keys))
		for _, r := range rows {
			grouped[r.EntryID] = append(grouped[r.EntryID], r.UserImage)
		}

		return mapResults(keys, grouped, emptySlice[domain.UserImage])
	}
}

// ---------------------------------------------------------------------------
// Card by EntryID (1:1 nullable)
// ---------------------------------------------------------------------------

func newCardBatchFn(repo cardRepo) dataloader.BatchFunc[uuid.UUID, *domain.Card] {
	return func(ctx context.Context, keys []uuid.UUID) []*dataloader.Result[*domain.Card] {
		rows, err := repo.GetByEntryIDs(ctx, keys)
		if err != nil {
			return errorResults[*domain.Card](len(keys), err)
		}

		byEntry := make(map[uuid.UUID]*domain.Card, len(rows))
		for i := range rows {
			c := rows[i] // copy to avoid aliasing
			byEntry[c.EntryID] = &c
		}

		results := make([]*dataloader.Result[*domain.Card], len(keys))
		for i, key := range keys {
			results[i] = &dataloader.Result[*domain.Card]{Data: byEntry[key]}
		}
		return results
	}
}

// ---------------------------------------------------------------------------
// Topics by EntryID
// ---------------------------------------------------------------------------

func newTopicsBatchFn(repo topicRepo) dataloader.BatchFunc[uuid.UUID, []domain.Topic] {
	return func(ctx context.Context, keys []uuid.UUID) []*dataloader.Result[[]domain.Topic] {
		rows, err := repo.GetByEntryIDs(ctx, keys)
		if err != nil {
			return errorResults[[]domain.Topic](len(keys), err)
		}

		grouped := make(map[uuid.UUID][]domain.Topic, len(keys))
		for _, r := range rows {
			grouped[r.EntryID] = append(grouped[r.EntryID], r.Topic)
		}

		return mapResults(keys, grouped, emptySlice[domain.Topic])
	}
}

// ---------------------------------------------------------------------------
// ReviewLogs by CardID
// ---------------------------------------------------------------------------

func newReviewLogsBatchFn(repo reviewLogRepo) dataloader.BatchFunc[uuid.UUID, []domain.ReviewLog] {
	return func(ctx context.Context, keys []uuid.UUID) []*dataloader.Result[[]domain.ReviewLog] {
		rows, err := repo.GetByCardIDs(ctx, keys)
		if err != nil {
			return errorResults[[]domain.ReviewLog](len(keys), err)
		}

		grouped := make(map[uuid.UUID][]domain.ReviewLog, len(keys))
		for _, r := range rows {
			grouped[r.CardID] = append(grouped[r.CardID], r.ReviewLog)
		}

		return mapResults(keys, grouped, emptySlice[domain.ReviewLog])
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// errorResults returns a slice of error results for all keys.
func errorResults[V any](n int, err error) []*dataloader.Result[V] {
	results := make([]*dataloader.Result[V], n)
	for i := range results {
		results[i] = &dataloader.Result[V]{Error: err}
	}
	return results
}

// mapResults maps grouped results back to key order, using defaultFn for missing keys.
func mapResults[V any](keys []uuid.UUID, grouped map[uuid.UUID]V, defaultFn func() V) []*dataloader.Result[V] {
	results := make([]*dataloader.Result[V], len(keys))
	for i, key := range keys {
		if v, ok := grouped[key]; ok {
			results[i] = &dataloader.Result[V]{Data: v}
		} else {
			results[i] = &dataloader.Result[V]{Data: defaultFn()}
		}
	}
	return results
}

// emptySlice returns a non-nil empty slice.
func emptySlice[T any]() []T {
	return []T{}
}
