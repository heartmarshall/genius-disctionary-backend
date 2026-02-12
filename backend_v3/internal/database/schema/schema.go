// Package schema содержит определения таблиц и колонок БД.
// Используется для type-safe построения SQL запросов через squirrel.
package schema

import (
	"strings"

	"github.com/Masterminds/squirrel"
)

// Table описывает таблицу базы данных.
type Table string

func (t Table) String() string { return string(t) }

// Column описывает колонку с опциональным префиксом таблицы.
type Column string

func (c Column) String() string { return string(c) }

// Qualified возвращает полное имя колонки "table.column".
func (c Column) Qualified() string { return string(c) }

// Bare возвращает "голое" имя колонки без таблицы.
func (c Column) Bare() string {
	parts := strings.Split(string(c), ".")
	if len(parts) > 1 {
		return parts[1]
	}
	return string(c)
}

// --- Fluent API для условий WHERE ---

func (c Column) Eq(val any) squirrel.Eq          { return squirrel.Eq{string(c): val} }
func (c Column) NotEq(val any) squirrel.NotEq    { return squirrel.NotEq{string(c): val} }
func (c Column) Lt(val any) squirrel.Lt          { return squirrel.Lt{string(c): val} }
func (c Column) LtOrEq(val any) squirrel.LtOrEq  { return squirrel.LtOrEq{string(c): val} }
func (c Column) Gt(val any) squirrel.Gt          { return squirrel.Gt{string(c): val} }
func (c Column) GtOrEq(val any) squirrel.GtOrEq  { return squirrel.GtOrEq{string(c): val} }
func (c Column) In(vals any) squirrel.Eq         { return squirrel.Eq{string(c): vals} }
func (c Column) NotIn(vals any) squirrel.NotEq   { return squirrel.NotEq{string(c): vals} }
func (c Column) Like(pat string) squirrel.Like   { return squirrel.Like{string(c): pat} }
func (c Column) ILike(pat string) squirrel.ILike { return squirrel.ILike{string(c): pat} }
func (c Column) IsNull() squirrel.Eq             { return squirrel.Eq{string(c): nil} }
func (c Column) IsNotNull() squirrel.NotEq       { return squirrel.NotEq{string(c): nil} }

// --- Fluent API для ORDER BY ---

func (c Column) Desc() string { return string(c) + " DESC" }
func (c Column) Asc() string  { return string(c) + " ASC" }

// ============================================================================
// TABLE DEFINITION INTERFACE
// ============================================================================

// TableDef описывает контракт для определения таблицы БД.
type TableDef interface {
	Columns() []string
	InsertColumns() []string
}

// Compile-time проверки, что все таблицы реализуют TableDef.
var (
	_ TableDef = DictionaryEntriesTable{}
	_ TableDef = SensesTable{}
	_ TableDef = TranslationsTable{}
	_ TableDef = ExamplesTable{}
	_ TableDef = ImagesTable{}
	_ TableDef = PronunciationsTable{}
	_ TableDef = CardsTable{}
	_ TableDef = ReviewLogsTable{}
	_ TableDef = HintsTable{}
	_ TableDef = InboxItemsTable{}
	_ TableDef = AuditRecordsTable{}
	_ TableDef = TopicsTable{}
	_ TableDef = DictionaryEntryTopicsTable{}
)

// ============================================================================
// DICTIONARY ENTRIES
// ============================================================================

type DictionaryEntriesTable struct {
	Name           Table
	ID             Column
	Text           Column
	TextNormalized Column
	Notes          Column
	CreatedAt      Column
	UpdatedAt      Column
}

var DictionaryEntries = DictionaryEntriesTable{
	Name:           "dictionary_entries",
	ID:             "dictionary_entries.id",
	Text:           "dictionary_entries.text",
	TextNormalized: "dictionary_entries.text_normalized",
	Notes:          "dictionary_entries.notes",
	CreatedAt:      "dictionary_entries.created_at",
	UpdatedAt:      "dictionary_entries.updated_at",
}

func (t DictionaryEntriesTable) Columns() []string {
	return []string{
		string(t.ID), string(t.Text), string(t.TextNormalized),
		string(t.Notes), string(t.CreatedAt), string(t.UpdatedAt),
	}
}

func (t DictionaryEntriesTable) InsertColumns() []string {
	return []string{"text", "text_normalized", "notes"}
}

// ============================================================================
// SENSES
// ============================================================================

type SensesTable struct {
	Name         Table
	ID           Column
	EntryID      Column
	Definition   Column
	PartOfSpeech Column
	SourceSlug   Column
	CefrLevel    Column
	CreatedAt    Column
}

var Senses = SensesTable{
	Name:         "senses",
	ID:           "senses.id",
	EntryID:      "senses.entry_id",
	Definition:   "senses.definition",
	PartOfSpeech: "senses.part_of_speech",
	SourceSlug:   "senses.source_slug",
	CefrLevel:    "senses.cefr_level",
	CreatedAt:    "senses.created_at",
}

func (t SensesTable) Columns() []string {
	return []string{
		string(t.ID), string(t.EntryID), string(t.Definition),
		string(t.PartOfSpeech), string(t.SourceSlug), string(t.CefrLevel),
		string(t.CreatedAt),
	}
}

func (t SensesTable) InsertColumns() []string {
	return []string{"entry_id", "definition", "part_of_speech", "source_slug", "cefr_level"}
}

// ============================================================================
// TRANSLATIONS
// ============================================================================

type TranslationsTable struct {
	Name       Table
	ID         Column
	SenseID    Column
	Text       Column
	SourceSlug Column
}

var Translations = TranslationsTable{
	Name:       "translations",
	ID:         "translations.id",
	SenseID:    "translations.sense_id",
	Text:       "translations.text",
	SourceSlug: "translations.source_slug",
}

func (t TranslationsTable) Columns() []string {
	return []string{
		string(t.ID), string(t.SenseID), string(t.Text), string(t.SourceSlug),
	}
}

func (t TranslationsTable) InsertColumns() []string {
	return []string{"sense_id", "text", "source_slug"}
}

// ============================================================================
// EXAMPLES
// ============================================================================

type ExamplesTable struct {
	Name        Table
	ID          Column
	SenseID     Column
	Sentence    Column
	Translation Column
	SourceSlug  Column
	CreatedAt   Column
}

var Examples = ExamplesTable{
	Name:        "examples",
	ID:          "examples.id",
	SenseID:     "examples.sense_id",
	Sentence:    "examples.sentence",
	Translation: "examples.translation",
	SourceSlug:  "examples.source_slug",
	CreatedAt:   "examples.created_at",
}

func (t ExamplesTable) Columns() []string {
	return []string{
		string(t.ID), string(t.SenseID), string(t.Sentence),
		string(t.Translation), string(t.SourceSlug), string(t.CreatedAt),
	}
}

func (t ExamplesTable) InsertColumns() []string {
	return []string{"sense_id", "sentence", "translation", "source_slug"}
}

// ============================================================================
// IMAGES
// ============================================================================

type ImagesTable struct {
	Name       Table
	ID         Column
	EntryID    Column
	URL        Column
	Caption    Column
	SourceSlug Column
}

var Images = ImagesTable{
	Name:       "images",
	ID:         "images.id",
	EntryID:    "images.entry_id",
	URL:        "images.url",
	Caption:    "images.caption",
	SourceSlug: "images.source_slug",
}

func (t ImagesTable) Columns() []string {
	return []string{
		string(t.ID), string(t.EntryID), string(t.URL),
		string(t.Caption), string(t.SourceSlug),
	}
}

func (t ImagesTable) InsertColumns() []string {
	return []string{"entry_id", "url", "caption", "source_slug"}
}

// ============================================================================
// PRONUNCIATIONS
// ============================================================================

type PronunciationsTable struct {
	Name          Table
	ID            Column
	EntryID       Column
	AudioURL      Column
	Transcription Column
	Region        Column
	SourceSlug    Column
}

var Pronunciations = PronunciationsTable{
	Name:          "pronunciations",
	ID:            "pronunciations.id",
	EntryID:       "pronunciations.entry_id",
	AudioURL:      "pronunciations.audio_url",
	Transcription: "pronunciations.transcription",
	Region:        "pronunciations.region",
	SourceSlug:    "pronunciations.source_slug",
}

func (t PronunciationsTable) Columns() []string {
	return []string{
		string(t.ID), string(t.EntryID), string(t.AudioURL),
		string(t.Transcription), string(t.Region), string(t.SourceSlug),
	}
}

func (t PronunciationsTable) InsertColumns() []string {
	return []string{"entry_id", "audio_url", "transcription", "region", "source_slug"}
}

// ============================================================================
// CARDS
// ============================================================================

type CardsTable struct {
	Name         Table
	ID           Column
	EntryID      Column
	Status       Column
	NextReviewAt Column
	IntervalDays Column
	EaseFactor   Column
	CreatedAt    Column
	UpdatedAt    Column
}

var Cards = CardsTable{
	Name:         "cards",
	ID:           "cards.id",
	EntryID:      "cards.entry_id",
	Status:       "cards.status",
	NextReviewAt: "cards.next_review_at",
	IntervalDays: "cards.interval_days",
	EaseFactor:   "cards.ease_factor",
	CreatedAt:    "cards.created_at",
	UpdatedAt:    "cards.updated_at",
}

func (t CardsTable) Columns() []string {
	return []string{
		string(t.ID), string(t.EntryID), string(t.Status),
		string(t.NextReviewAt), string(t.IntervalDays), string(t.EaseFactor),
		string(t.CreatedAt), string(t.UpdatedAt),
	}
}

func (t CardsTable) InsertColumns() []string {
	return []string{"entry_id", "status", "next_review_at", "interval_days", "ease_factor"}
}

// ============================================================================
// REVIEW LOGS
// ============================================================================

type ReviewLogsTable struct {
	Name       Table
	ID         Column
	CardID     Column
	Grade      Column
	DurationMs Column
	ReviewedAt Column
}

var ReviewLogs = ReviewLogsTable{
	Name:       "review_logs",
	ID:         "review_logs.id",
	CardID:     "review_logs.card_id",
	Grade:      "review_logs.grade",
	DurationMs: "review_logs.duration_ms",
	ReviewedAt: "review_logs.reviewed_at",
}

func (t ReviewLogsTable) Columns() []string {
	return []string{
		string(t.ID), string(t.CardID), string(t.Grade),
		string(t.DurationMs), string(t.ReviewedAt),
	}
}

func (t ReviewLogsTable) InsertColumns() []string {
	return []string{"card_id", "grade", "duration_ms"}
}

// ============================================================================
// HINTS
// ============================================================================

type HintsTable struct {
	Name      Table
	ID        Column
	CardID    Column
	Text      Column
	CreatedAt Column
	UpdatedAt Column
}

var Hints = HintsTable{
	Name:      "hints",
	ID:        "hints.id",
	CardID:    "hints.card_id",
	Text:      "hints.text",
	CreatedAt: "hints.created_at",
	UpdatedAt: "hints.updated_at",
}

func (t HintsTable) Columns() []string {
	return []string{
		string(t.ID), string(t.CardID), string(t.Text),
		string(t.CreatedAt), string(t.UpdatedAt),
	}
}

func (t HintsTable) InsertColumns() []string {
	return []string{"card_id", "text"}
}

// ============================================================================
// INBOX ITEMS
// ============================================================================

type InboxItemsTable struct {
	Name      Table
	ID        Column
	Text      Column
	Context   Column
	CreatedAt Column
}

var InboxItems = InboxItemsTable{
	Name:      "inbox_items",
	ID:        "inbox_items.id",
	Text:      "inbox_items.text",
	Context:   "inbox_items.context",
	CreatedAt: "inbox_items.created_at",
}

func (t InboxItemsTable) Columns() []string {
	return []string{
		string(t.ID), string(t.Text), string(t.Context), string(t.CreatedAt),
	}
}

func (t InboxItemsTable) InsertColumns() []string {
	return []string{"text", "context"}
}

// ============================================================================
// AUDIT RECORDS
// ============================================================================

type AuditRecordsTable struct {
	Name       Table
	ID         Column
	EntityType Column
	EntityID   Column
	Action     Column
	Changes    Column
	CreatedAt  Column
}

var AuditRecords = AuditRecordsTable{
	Name:       "audit_records",
	ID:         "audit_records.id",
	EntityType: "audit_records.entity_type",
	EntityID:   "audit_records.entity_id",
	Action:     "audit_records.action",
	Changes:    "audit_records.changes",
	CreatedAt:  "audit_records.created_at",
}

func (t AuditRecordsTable) Columns() []string {
	return []string{
		string(t.ID), string(t.EntityType), string(t.EntityID),
		string(t.Action), string(t.Changes), string(t.CreatedAt),
	}
}

func (t AuditRecordsTable) InsertColumns() []string {
	return []string{"entity_type", "entity_id", "action", "changes"}
}

// ============================================================================
// TOPICS
// ============================================================================

type TopicsTable struct {
	Name        Table
	ID          Column
	NameColumn  Column // Переименовали, чтобы не конфликтовать с именем таблицы в DSL, если вдруг
	Description Column
	CreatedAt   Column
	UpdatedAt   Column
}

var Topics = TopicsTable{
	Name:        "topics",
	ID:          "topics.id",
	NameColumn:  "topics.name",
	Description: "topics.description",
	CreatedAt:   "topics.created_at",
	UpdatedAt:   "topics.updated_at",
}

func (t TopicsTable) Columns() []string {
	return []string{
		string(t.ID), string(t.NameColumn), string(t.Description),
		string(t.CreatedAt), string(t.UpdatedAt),
	}
}

func (t TopicsTable) InsertColumns() []string {
	return []string{"name", "description"}
}

// ============================================================================
// DICTIONARY ENTRY TOPICS (Junction)
// ============================================================================

type DictionaryEntryTopicsTable struct {
	Name    Table
	EntryID Column
	TopicID Column
}

var DictionaryEntryTopics = DictionaryEntryTopicsTable{
	Name:    "dictionary_entry_topics",
	EntryID: "dictionary_entry_topics.entry_id",
	TopicID: "dictionary_entry_topics.topic_id",
}

func (t DictionaryEntryTopicsTable) Columns() []string {
	return []string{
		string(t.EntryID), string(t.TopicID),
	}
}

func (t DictionaryEntryTopicsTable) InsertColumns() []string {
	return []string{"entry_id", "topic_id"}
}
