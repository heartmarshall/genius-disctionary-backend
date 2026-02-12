// Package repository определяет интерфейсы и реализации репозиториев.
// Интерфейсы позволяют использовать моки в тестах и упрощают dependency injection.
package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database/repository/cards"
	"github.com/heartmarshall/my-english/internal/database/repository/dictionary"
	"github.com/heartmarshall/my-english/internal/database/repository/topics"
	"github.com/heartmarshall/my-english/internal/model"
)

// ============================================================================
// DICTIONARY
// ============================================================================

// DictionaryRepository определяет контракт для работы со словарными записями.
type DictionaryRepository interface {
	// Читающие операции
	GetByID(ctx context.Context, id uuid.UUID) (*model.DictionaryEntry, error)
	FindByNormalizedText(ctx context.Context, text string) (*model.DictionaryEntry, error)
	Find(ctx context.Context, filter dictionary.DictionaryFilter) ([]model.DictionaryEntry, error)
	CountTotal(ctx context.Context, filter dictionary.DictionaryFilter) (int64, error)
	ExistsByNormalizedText(ctx context.Context, text string) (bool, error)
	ListByIDs(ctx context.Context, ids []uuid.UUID) ([]model.DictionaryEntry, error)
	ListAll(ctx context.Context) ([]model.DictionaryEntry, error)

	// Пишущие операции
	Create(ctx context.Context, entry *model.DictionaryEntry) (*model.DictionaryEntry, error)
	CreateOrGet(ctx context.Context, entry *model.DictionaryEntry) (*model.DictionaryEntry, error)
	Update(ctx context.Context, id uuid.UUID, entry *model.DictionaryEntry) (*model.DictionaryEntry, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// ============================================================================
// CARDS
// ============================================================================

// CardRepository определяет контракт для работы с карточками.
type CardRepository interface {
	// Читающие операции
	GetByID(ctx context.Context, id uuid.UUID) (*model.Card, error)
	GetByEntryID(ctx context.Context, entryID uuid.UUID) (*model.Card, error)
	GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*model.Card, error)
	GetDueCards(ctx context.Context, now time.Time, limit int) ([]model.Card, error)
	GetDashboardStats(ctx context.Context) (*cards.DashboardStats, error)
	ListByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]model.Card, error)

	// Пишущие операции
	Create(ctx context.Context, card *model.Card) (*model.Card, error)
	Update(ctx context.Context, id uuid.UUID, card *model.Card) (*model.Card, error)
	UpdateSRSFields(ctx context.Context, id uuid.UUID, status model.LearningStatus, nextReviewAt *time.Time, intervalDays int, easeFactor float64) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ReviewLogRepository определяет контракт для работы с логами ревью.
type ReviewLogRepository interface {
	Create(ctx context.Context, log *model.ReviewLog) (*model.ReviewLog, error)
	ListByCardID(ctx context.Context, cardID uuid.UUID, limit int) ([]model.ReviewLog, error)
}

// ============================================================================
// CONTENT (Senses, Examples, Translations, Images, Pronunciations)
// ============================================================================

// SenseRepository определяет контракт для работы со смыслами слов.
type SenseRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Sense, error)
	ListByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]model.Sense, error)
	Create(ctx context.Context, sense *model.Sense) (*model.Sense, error)
	BatchCreate(ctx context.Context, senses []model.Sense) ([]model.Sense, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// ExampleRepository определяет контракт для работы с примерами.
type ExampleRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Example, error)
	ListBySenseIDs(ctx context.Context, senseIDs []uuid.UUID) ([]model.Example, error)
	BatchCreate(ctx context.Context, examples []model.Example) ([]model.Example, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// TranslationRepository определяет контракт для работы с переводами.
type TranslationRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Translation, error)
	ListBySenseIDs(ctx context.Context, senseIDs []uuid.UUID) ([]model.Translation, error)
	BatchCreate(ctx context.Context, translations []model.Translation) ([]model.Translation, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// ImageRepository определяет контракт для работы с изображениями.
type ImageRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Image, error)
	ListByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]model.Image, error)
	BatchCreate(ctx context.Context, images []model.Image) ([]model.Image, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// PronunciationRepository определяет контракт для работы с произношениями.
type PronunciationRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Pronunciation, error)
	ListByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]model.Pronunciation, error)
	BatchCreate(ctx context.Context, pronunciations []model.Pronunciation) ([]model.Pronunciation, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// ============================================================================
// INBOX & AUDIT
// ============================================================================

// InboxRepository определяет контракт для работы с inbox.
type InboxRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.InboxItem, error)
	ListAll(ctx context.Context) ([]model.InboxItem, error)
	ListPaginated(ctx context.Context, limit, offset int) ([]model.InboxItem, error)
	Count(ctx context.Context) (int64, error)
	Create(ctx context.Context, item *model.InboxItem) (*model.InboxItem, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteAll(ctx context.Context) (int64, error)
}

// AuditRepository определяет контракт для работы с аудит логами.
type AuditRepository interface {
	Create(ctx context.Context, audit *model.AuditRecord) (*model.AuditRecord, error)
}

// ============================================================================
// TYPE ALIASES (for convenience)
// ============================================================================

// DashboardStats is an alias for cards.DashboardStats.
type DashboardStats = cards.DashboardStats

// TopicRepository определяет контракт для работы с топиками.
type TopicRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Topic, error)
	ListAll(ctx context.Context) ([]model.Topic, error)
	ListByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]topics.TopicWithEntryID, error)

	Create(ctx context.Context, topic *model.Topic) (*model.Topic, error)
	Update(ctx context.Context, id uuid.UUID, topic *model.Topic) (*model.Topic, error)
	Delete(ctx context.Context, id uuid.UUID) error

	BindToEntry(ctx context.Context, entryID, topicID uuid.UUID) error
	UnbindFromEntry(ctx context.Context, entryID, topicID uuid.UUID) error
	UnbindAllFromEntry(ctx context.Context, entryID uuid.UUID) error
}
