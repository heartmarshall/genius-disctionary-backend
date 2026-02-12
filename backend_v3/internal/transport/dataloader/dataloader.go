package dataloader

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database/repository"
	"github.com/heartmarshall/my-english/internal/model"
	"github.com/vikstrous/dataloadgen"
)

type ctxKey string

const loadersKey ctxKey = "dataloaders"

const (
	// DefaultDataLoaderWaitTime is the default wait time before batching requests (2ms)
	DefaultDataLoaderWaitTime = time.Millisecond * 2
	// DefaultDataLoaderMaxBatchSize is the default maximum batch size for DataLoaders
	DefaultDataLoaderMaxBatchSize = 100
)

// LoaderConfig содержит конфигурацию для DataLoaders.
type LoaderConfig struct {
	// WaitTime определяет время ожидания перед батчингом запросов.
	WaitTime time.Duration
	// MaxBatchSize определяет максимальный размер батча.
	MaxBatchSize int
	// Logger для логирования (опционально).
	Logger *slog.Logger
}

// DefaultLoaderConfig возвращает конфигурацию по умолчанию.
func DefaultLoaderConfig() LoaderConfig {
	return LoaderConfig{
		WaitTime:     DefaultDataLoaderWaitTime,
		MaxBatchSize: DefaultDataLoaderMaxBatchSize,
		Logger:       nil,
	}
}

// Loaders содержит все доступные загрузчики данных.
type Loaders struct {
	// 1:N Loaders (Одно слово -> Много смыслов/картинок/произношений/топиков)
	SensesByEntryID         *dataloadgen.Loader[uuid.UUID, []model.Sense]
	ImagesByEntryID         *dataloadgen.Loader[uuid.UUID, []model.Image]
	PronunciationsByEntryID *dataloadgen.Loader[uuid.UUID, []model.Pronunciation]
	TopicsByEntryID         *dataloadgen.Loader[uuid.UUID, []model.Topic]

	// 1:N Loaders (Один смысл -> Много примеров/переводов)
	ExamplesBySenseID     *dataloadgen.Loader[uuid.UUID, []model.Example]
	TranslationsBySenseID *dataloadgen.Loader[uuid.UUID, []model.Translation]

	// 1:1 Loaders (Одно слово -> Одна карточка)
	CardByEntryID *dataloadgen.Loader[uuid.UUID, *model.Card]

	// Конфигурация
	config LoaderConfig
}

// NewLoaders создает экземпляры всех загрузчиков.
func NewLoaders(repos *repository.Registry, config LoaderConfig) *Loaders {
	return &Loaders{
		SensesByEntryID: dataloadgen.NewLoader(
			newSensesByEntryIDFetcher(repos.Senses, config.Logger),
			dataloadgen.WithWait(config.WaitTime),
			dataloadgen.WithBatchCapacity(config.MaxBatchSize),
		),
		ImagesByEntryID: dataloadgen.NewLoader(
			newImagesByEntryIDFetcher(repos.Images, config.Logger),
			dataloadgen.WithWait(config.WaitTime),
			dataloadgen.WithBatchCapacity(config.MaxBatchSize),
		),
		PronunciationsByEntryID: dataloadgen.NewLoader(
			newPronunciationsByEntryIDFetcher(repos.Pronunciations, config.Logger),
			dataloadgen.WithWait(config.WaitTime),
			dataloadgen.WithBatchCapacity(config.MaxBatchSize),
		),
		ExamplesBySenseID: dataloadgen.NewLoader(
			newExamplesBySenseIDFetcher(repos.Examples, config.Logger),
			dataloadgen.WithWait(config.WaitTime),
			dataloadgen.WithBatchCapacity(config.MaxBatchSize),
		),
		TranslationsBySenseID: dataloadgen.NewLoader(
			newTranslationsBySenseIDFetcher(repos.Translations, config.Logger),
			dataloadgen.WithWait(config.WaitTime),
			dataloadgen.WithBatchCapacity(config.MaxBatchSize),
		),
		CardByEntryID: dataloadgen.NewLoader(
			newCardByEntryIDFetcher(repos.Cards, config.Logger),
			dataloadgen.WithWait(config.WaitTime),
			dataloadgen.WithBatchCapacity(config.MaxBatchSize),
		),
		TopicsByEntryID: dataloadgen.NewLoader(
			newTopicsByEntryIDFetcher(repos.Topics, config.Logger),
			dataloadgen.WithWait(config.WaitTime),
			dataloadgen.WithBatchCapacity(config.MaxBatchSize),
		),
		config: config,
	}
}

// Middleware внедряет Loaders в контекст запроса.
// Loaders создаются заново для каждого запроса (Request-scoped),
// чтобы кэширование жило только в рамках одного запроса.
func Middleware(repos *repository.Registry) func(http.Handler) http.Handler {
	return MiddlewareWithConfig(repos, DefaultLoaderConfig())
}

// MiddlewareWithConfig внедряет Loaders с кастомной конфигурацией.
func MiddlewareWithConfig(repos *repository.Registry, config LoaderConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			loaders := NewLoaders(repos, config)
			ctx := context.WithValue(r.Context(), loadersKey, loaders)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

// For извлекает Loaders из контекста.
// Возвращает nil, если Loaders не найдены в контексте.
func For(ctx context.Context) *Loaders {
	if ctx == nil {
		return nil
	}
	if loaders, ok := ctx.Value(loadersKey).(*Loaders); ok {
		return loaders
	}
	return nil
}
