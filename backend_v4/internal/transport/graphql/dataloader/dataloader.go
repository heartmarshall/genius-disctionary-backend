// Package dataloader provides per-request DataLoaders for batching GraphQL
// resolver queries into single SQL calls. DataLoaders call repositories
// directly, bypassing the service layer. Authorization is ensured via SQL
// (WHERE user_id filters in repo queries).
package dataloader

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/graph-gophers/dataloader/v7"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/card"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/example"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/image"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/pronunciation"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/reviewlog"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/topic"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/translation"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

const (
	maxBatch = 100
	wait     = 2 * time.Millisecond
)

// ---------------------------------------------------------------------------
// Repository interfaces (consumer-defined)
// ---------------------------------------------------------------------------

type senseRepo interface {
	GetByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]domain.Sense, error)
}

type translationRepo interface {
	GetBySenseIDs(ctx context.Context, senseIDs []uuid.UUID) ([]translation.TranslationWithSenseID, error)
}

type exampleRepo interface {
	GetBySenseIDs(ctx context.Context, senseIDs []uuid.UUID) ([]example.ExampleWithSenseID, error)
}

type pronunciationRepo interface {
	GetByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]pronunciation.PronunciationWithEntryID, error)
}

type imageRepo interface {
	GetCatalogByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]image.CatalogImageWithEntryID, error)
	GetUserByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]image.UserImageWithEntryID, error)
}

type cardRepo interface {
	GetByEntryIDs(ctx context.Context, userID uuid.UUID, entryIDs []uuid.UUID) ([]card.CardWithEntryID, error)
}

type topicRepo interface {
	GetByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]topic.TopicWithEntryID, error)
}

type reviewLogRepo interface {
	GetByCardIDs(ctx context.Context, cardIDs []uuid.UUID) ([]reviewlog.ReviewLogWithCardID, error)
}

// ---------------------------------------------------------------------------
// Repos aggregates all repositories needed by DataLoaders.
// ---------------------------------------------------------------------------

// Repos holds all repositories required by DataLoaders.
type Repos struct {
	Sense         senseRepo
	Translation   translationRepo
	Example       exampleRepo
	Pronunciation pronunciationRepo
	Image         imageRepo
	Card          cardRepo
	Topic         topicRepo
	ReviewLog     reviewLogRepo
}

// ---------------------------------------------------------------------------
// Loaders holds all per-request DataLoader instances.
// ---------------------------------------------------------------------------

// Loaders contains all 9 DataLoaders. Created per-request via NewLoaders.
type Loaders struct {
	SensesByEntryID         *dataloader.Loader[uuid.UUID, []domain.Sense]
	TranslationsBySenseID   *dataloader.Loader[uuid.UUID, []domain.Translation]
	ExamplesBySenseID       *dataloader.Loader[uuid.UUID, []domain.Example]
	PronunciationsByEntryID *dataloader.Loader[uuid.UUID, []domain.RefPronunciation]
	CatalogImagesByEntryID  *dataloader.Loader[uuid.UUID, []domain.RefImage]
	UserImagesByEntryID     *dataloader.Loader[uuid.UUID, []domain.UserImage]
	CardByEntryID           *dataloader.Loader[uuid.UUID, *domain.Card]
	TopicsByEntryID         *dataloader.Loader[uuid.UUID, []domain.Topic]
	ReviewLogsByCardID      *dataloader.Loader[uuid.UUID, []domain.ReviewLog]
}

// NewLoaders creates a new set of DataLoaders backed by the given repositories.
// Must be called per-request (loaders cache results within a single request).
func NewLoaders(repos *Repos) *Loaders {
	return &Loaders{
		SensesByEntryID:         newLoader(newSensesBatchFn(repos.Sense)),
		TranslationsBySenseID:   newLoader(newTranslationsBatchFn(repos.Translation)),
		ExamplesBySenseID:       newLoader(newExamplesBatchFn(repos.Example)),
		PronunciationsByEntryID: newLoader(newPronunciationsBatchFn(repos.Pronunciation)),
		CatalogImagesByEntryID:  newLoader(newCatalogImagesBatchFn(repos.Image)),
		UserImagesByEntryID:     newLoader(newUserImagesBatchFn(repos.Image)),
		CardByEntryID:           newLoader(newCardBatchFn(repos.Card)),
		TopicsByEntryID:         newLoader(newTopicsBatchFn(repos.Topic)),
		ReviewLogsByCardID:      newLoader(newReviewLogsBatchFn(repos.ReviewLog)),
	}
}

// newLoader creates a dataloader.Loader with standard batch parameters.
func newLoader[V any](batchFn dataloader.BatchFunc[uuid.UUID, V]) *dataloader.Loader[uuid.UUID, V] {
	return dataloader.NewBatchedLoader(
		batchFn,
		dataloader.WithWait[uuid.UUID, V](wait),
		dataloader.WithBatchCapacity[uuid.UUID, V](maxBatch),
	)
}

// ---------------------------------------------------------------------------
// Context helpers
// ---------------------------------------------------------------------------

type contextKey string

const loadersKey contextKey = "dataloaders"

// WithLoaders stores Loaders in the context.
func WithLoaders(ctx context.Context, l *Loaders) context.Context {
	return context.WithValue(ctx, loadersKey, l)
}

// FromContext retrieves Loaders from the context.
// Panics if loaders are not present (indicates middleware misconfiguration).
func FromContext(ctx context.Context) *Loaders {
	l, ok := ctx.Value(loadersKey).(*Loaders)
	if !ok || l == nil {
		panic("dataloader: loaders not found in context — is middleware configured?")
	}
	return l
}
