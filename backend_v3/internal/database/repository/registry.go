// Package repository предоставляет репозитории для работы с базой данных.
package repository

import (
	"github.com/heartmarshall/my-english/internal/database"
	"github.com/heartmarshall/my-english/internal/database/repository/audit"
	"github.com/heartmarshall/my-english/internal/database/repository/cards"
	"github.com/heartmarshall/my-english/internal/database/repository/content"
	"github.com/heartmarshall/my-english/internal/database/repository/dictionary"
	"github.com/heartmarshall/my-english/internal/database/repository/inbox"
	"github.com/heartmarshall/my-english/internal/database/repository/topics"
)

// ============================================================================
// REGISTRY
// ============================================================================

// Registry объединяет все репозитории приложения.
// Используйте для dependency injection в сервисах.
//
// Пример использования:
//
//	registry := repository.NewRegistry(pool)
//	service := dictionary.NewService(registry.Dictionary, registry.Senses)
type Registry struct {
	// Словарь
	Dictionary DictionaryRepository

	// Контент словаря
	Senses         SenseRepository
	Translations   TranslationRepository
	Examples       ExampleRepository
	Images         ImageRepository
	Pronunciations PronunciationRepository

	// Карточки и SRS
	Cards      CardRepository
	ReviewLogs ReviewLogRepository

	// Inbox
	Inbox InboxRepository

	// Аудит
	Audit AuditRepository

	// Топики
	Topics TopicRepository
}

// NewRegistry создает все репозитории, используя переданный Querier.
//
// Параметры:
//   - q: Querier для выполнения запросов (обычно *pgxpool.Pool)
//
// Для транзакций создавайте новый Registry с tx вместо pool:
//
//	database.WithTx(ctx, pool, func(ctx context.Context, tx database.Querier) error {
//	    txRegistry := repository.NewRegistry(tx)
//	    // использовать txRegistry...
//	})
func NewRegistry(q database.Querier) *Registry {
	return &Registry{
		Dictionary:     dictionary.NewDictionaryRepository(q),
		Senses:         content.NewSenseRepository(q),
		Translations:   content.NewTranslationRepository(q),
		Examples:       content.NewExampleRepository(q),
		Images:         content.NewImageRepository(q),
		Pronunciations: content.NewPronunciationRepository(q),
		Cards:          cards.NewCardRepository(q),
		ReviewLogs:     cards.NewReviewLogRepository(q),
		Inbox:          inbox.NewInboxRepository(q),
		Audit:          audit.NewAuditRepository(q),
		Topics:         topics.NewTopicRepository(q),
	}
}

// ============================================================================
// REGISTRY WITH CUSTOM REPOS (for testing)
// ============================================================================

// RegistryConfig позволяет создать Registry с кастомными реализациями.
// Используется для тестирования с моками.
type RegistryConfig struct {
	Dictionary     DictionaryRepository
	Senses         SenseRepository
	Translations   TranslationRepository
	Examples       ExampleRepository
	Images         ImageRepository
	Pronunciations PronunciationRepository
	Cards          CardRepository
	ReviewLogs     ReviewLogRepository
	Inbox          InboxRepository
	Audit          AuditRepository
	Topics         TopicRepository
}

// NewRegistryWithConfig создает Registry с кастомными реализациями.
func NewRegistryWithConfig(cfg RegistryConfig) *Registry {
	return &Registry{
		Dictionary:     cfg.Dictionary,
		Senses:         cfg.Senses,
		Translations:   cfg.Translations,
		Examples:       cfg.Examples,
		Images:         cfg.Images,
		Pronunciations: cfg.Pronunciations,
		Cards:          cfg.Cards,
		ReviewLogs:     cfg.ReviewLogs,
		Inbox:          cfg.Inbox,
		Audit:          cfg.Audit,
		Topics:         cfg.Topics,
	}
}

// ============================================================================
// TX REGISTRY FACTORY
// ============================================================================

// TxRegistryFactory создаёт Registry для транзакций.
type TxRegistryFactory struct {
	pool database.Querier
}

// NewTxRegistryFactory создаёт фабрику.
func NewTxRegistryFactory(pool database.Querier) *TxRegistryFactory {
	return &TxRegistryFactory{pool: pool}
}

// ForTx создаёт Registry для транзакции.
func (f *TxRegistryFactory) ForTx(tx database.Querier) *Registry {
	return NewRegistry(tx)
}

// Default возвращает Registry для обычных операций (без транзакции).
func (f *TxRegistryFactory) Default() *Registry {
	return NewRegistry(f.pool)
}
