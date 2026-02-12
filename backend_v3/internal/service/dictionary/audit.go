package dictionary

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/model"
	"github.com/heartmarshall/my-english/internal/service/types"
)

// createAuditLog создает запись аудита для операции над записью словаря.
func (s *Service) createAuditLog(ctx context.Context, entityID uuid.UUID, action model.AuditAction, changes model.JSON) error {
	audit := &model.AuditRecord{
		EntityType: model.EntityEntry,
		EntityID:   &entityID,
		Action:     action,
		Changes:    changes,
	}
	_, err := s.repos.Audit.Create(ctx, audit)
	if err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}
	return nil
}

// createAuditLogForEntity создает запись аудита для любой сущности.
func (s *Service) createAuditLogForEntity(ctx context.Context, entityType model.EntityType, entityID uuid.UUID, action model.AuditAction, changes model.JSON) error {
	audit := &model.AuditRecord{
		EntityType: entityType,
		EntityID:   &entityID,
		Action:     action,
		Changes:    changes,
	}
	_, err := s.repos.Audit.Create(ctx, audit)
	if err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}
	return nil
}

// diffDictionaryEntry сравнивает две записи словаря и возвращает изменения полей.
func diffDictionaryEntry(old, new *model.DictionaryEntry) model.JSON {
	changes := make(model.JSON)

	if old.Text != new.Text {
		changes[types.AuditFieldText] = map[string]any{
			types.AuditFieldOld: old.Text,
			types.AuditFieldNew: new.Text,
		}
	}

	if old.TextNormalized != new.TextNormalized {
		changes[types.AuditFieldTextNormalized] = map[string]any{
			types.AuditFieldOld: old.TextNormalized,
			types.AuditFieldNew: new.TextNormalized,
		}
	}

	return changes
}

// diffSense сравнивает два смысла и возвращает изменения полей.
func diffSense(old, new *model.Sense) model.JSON {
	changes := make(model.JSON)

	if !equalStringPtr(old.Definition, new.Definition) {
		changes[types.AuditFieldDefinition] = map[string]any{
			types.AuditFieldOld: old.Definition,
			types.AuditFieldNew: new.Definition,
		}
	}

	if !equalPartOfSpeechPtr(old.PartOfSpeech, new.PartOfSpeech) {
		changes[types.AuditFieldPartOfSpeech] = map[string]any{
			types.AuditFieldOld: old.PartOfSpeech,
			types.AuditFieldNew: new.PartOfSpeech,
		}
	}

	if old.SourceSlug != new.SourceSlug {
		changes[types.AuditFieldSourceSlug] = map[string]any{
			types.AuditFieldOld: old.SourceSlug,
			types.AuditFieldNew: new.SourceSlug,
		}
	}

	if !equalStringPtr(old.CefrLevel, new.CefrLevel) {
		changes[types.AuditFieldCefrLevel] = map[string]any{
			types.AuditFieldOld: old.CefrLevel,
			types.AuditFieldNew: new.CefrLevel,
		}
	}

	return changes
}

// diffCard сравнивает две карточки и возвращает изменения полей.
func diffCard(old, new *model.Card) model.JSON {
	changes := make(model.JSON)

	if old.Status != new.Status {
		changes[types.AuditFieldStatus] = map[string]any{
			types.AuditFieldOld: old.Status,
			types.AuditFieldNew: new.Status,
		}
	}

	if !equalTimePtr(old.NextReviewAt, new.NextReviewAt) {
		changes[types.AuditFieldNextReviewAt] = map[string]any{
			types.AuditFieldOld: formatTimePtr(old.NextReviewAt),
			types.AuditFieldNew: formatTimePtr(new.NextReviewAt),
		}
	}

	if old.IntervalDays != new.IntervalDays {
		changes[types.AuditFieldIntervalDays] = map[string]any{
			types.AuditFieldOld: old.IntervalDays,
			types.AuditFieldNew: new.IntervalDays,
		}
	}

	if old.EaseFactor != new.EaseFactor {
		changes[types.AuditFieldEaseFactor] = map[string]any{
			types.AuditFieldOld: old.EaseFactor,
			types.AuditFieldNew: new.EaseFactor,
		}
	}

	return changes
}

// diffTranslation сравнивает два перевода и возвращает изменения полей.
func diffTranslation(old, new *model.Translation) model.JSON {
	changes := make(model.JSON)

	if old.Text != new.Text {
		changes[types.AuditFieldText] = map[string]any{
			types.AuditFieldOld: old.Text,
			types.AuditFieldNew: new.Text,
		}
	}

	if old.SourceSlug != new.SourceSlug {
		changes[types.AuditFieldSourceSlug] = map[string]any{
			types.AuditFieldOld: old.SourceSlug,
			types.AuditFieldNew: new.SourceSlug,
		}
	}

	return changes
}

// diffExample сравнивает два примера и возвращает изменения полей.
func diffExample(old, new *model.Example) model.JSON {
	changes := make(model.JSON)

	if old.Sentence != new.Sentence {
		changes[types.AuditFieldSentence] = map[string]any{
			types.AuditFieldOld: old.Sentence,
			types.AuditFieldNew: new.Sentence,
		}
	}

	if !equalStringPtr(old.Translation, new.Translation) {
		changes[types.AuditFieldTranslation] = map[string]any{
			types.AuditFieldOld: old.Translation,
			types.AuditFieldNew: new.Translation,
		}
	}

	if old.SourceSlug != new.SourceSlug {
		changes[types.AuditFieldSourceSlug] = map[string]any{
			types.AuditFieldOld: old.SourceSlug,
			types.AuditFieldNew: new.SourceSlug,
		}
	}

	return changes
}

// diffImage сравнивает два изображения и возвращает изменения полей.
func diffImage(old, new *model.Image) model.JSON {
	changes := make(model.JSON)

	if old.URL != new.URL {
		changes[types.AuditFieldURL] = map[string]any{
			types.AuditFieldOld: old.URL,
			types.AuditFieldNew: new.URL,
		}
	}

	if !equalStringPtr(old.Caption, new.Caption) {
		changes[types.AuditFieldCaption] = map[string]any{
			types.AuditFieldOld: old.Caption,
			types.AuditFieldNew: new.Caption,
		}
	}

	if old.SourceSlug != new.SourceSlug {
		changes[types.AuditFieldSourceSlug] = map[string]any{
			types.AuditFieldOld: old.SourceSlug,
			types.AuditFieldNew: new.SourceSlug,
		}
	}

	return changes
}

// diffPronunciation сравнивает два произношения и возвращает изменения полей.
func diffPronunciation(old, new *model.Pronunciation) model.JSON {
	changes := make(model.JSON)

	if !equalStringPtr(old.AudioURL, new.AudioURL) {
		changes[types.AuditFieldAudioURL] = map[string]any{
			types.AuditFieldOld: old.AudioURL,
			types.AuditFieldNew: new.AudioURL,
		}
	}

	if old.Transcription != new.Transcription {
		changes[types.AuditFieldTranscription] = map[string]any{
			types.AuditFieldOld: old.Transcription,
			types.AuditFieldNew: new.Transcription,
		}
	}

	if !equalStringPtr(old.Region, new.Region) {
		changes[types.AuditFieldRegion] = map[string]any{
			types.AuditFieldOld: old.Region,
			types.AuditFieldNew: new.Region,
		}
	}

	if old.SourceSlug != new.SourceSlug {
		changes[types.AuditFieldSourceSlug] = map[string]any{
			types.AuditFieldOld: old.SourceSlug,
			types.AuditFieldNew: new.SourceSlug,
		}
	}

	return changes
}

// buildCreateChanges создает структуру изменений для операции CREATE.
// Записывает все поля созданной сущности.
func buildCreateChanges(entity any) model.JSON {
	changes := make(model.JSON)

	switch v := entity.(type) {
	case *model.DictionaryEntry:
		changes[types.AuditFieldText] = v.Text
		changes[types.AuditFieldTextNormalized] = v.TextNormalized
	case *model.Sense:
		if v.Definition != nil {
			changes[types.AuditFieldDefinition] = *v.Definition
		}
		if v.PartOfSpeech != nil {
			changes[types.AuditFieldPartOfSpeech] = *v.PartOfSpeech
		}
		changes[types.AuditFieldSourceSlug] = v.SourceSlug
		if v.CefrLevel != nil {
			changes[types.AuditFieldCefrLevel] = *v.CefrLevel
		}
	case *model.Translation:
		changes[types.AuditFieldText] = v.Text
		changes[types.AuditFieldSourceSlug] = v.SourceSlug
	case *model.Example:
		changes[types.AuditFieldSentence] = v.Sentence
		if v.Translation != nil {
			changes[types.AuditFieldTranslation] = *v.Translation
		}
		changes[types.AuditFieldSourceSlug] = v.SourceSlug
	case *model.Image:
		changes[types.AuditFieldURL] = v.URL
		if v.Caption != nil {
			changes[types.AuditFieldCaption] = *v.Caption
		}
		changes[types.AuditFieldSourceSlug] = v.SourceSlug
	case *model.Pronunciation:
		if v.AudioURL != nil {
			changes[types.AuditFieldAudioURL] = *v.AudioURL
		}
		changes[types.AuditFieldTranscription] = v.Transcription
		if v.Region != nil {
			changes[types.AuditFieldRegion] = *v.Region
		}
		changes[types.AuditFieldSourceSlug] = v.SourceSlug
	case *model.Card:
		changes[types.AuditFieldEntryID] = v.EntryID.String()
		changes[types.AuditFieldStatus] = v.Status
		if v.NextReviewAt != nil {
			changes[types.AuditFieldNextReviewAt] = v.NextReviewAt.Format(time.RFC3339)
		}
		changes[types.AuditFieldIntervalDays] = v.IntervalDays
		changes[types.AuditFieldEaseFactor] = v.EaseFactor
	}

	return changes
}

// buildDeleteChanges создает структуру изменений для операции DELETE.
// Записывает все поля удаленной сущности для истории.
func buildDeleteChanges(entity any) model.JSON {
	changes := make(model.JSON)
	changes[types.AuditFieldDeleted] = true

	switch v := entity.(type) {
	case *model.DictionaryEntry:
		changes[types.AuditFieldText] = v.Text
		changes[types.AuditFieldTextNormalized] = v.TextNormalized
	case *model.Sense:
		changes[types.AuditFieldSenseID] = v.ID.String()
		if v.Definition != nil {
			changes[types.AuditFieldDefinition] = *v.Definition
		}
		if v.PartOfSpeech != nil {
			changes[types.AuditFieldPartOfSpeech] = *v.PartOfSpeech
		}
		changes[types.AuditFieldSourceSlug] = v.SourceSlug
	case *model.Translation:
		changes[types.AuditFieldTranslationID] = v.ID.String()
		changes[types.AuditFieldText] = v.Text
		changes[types.AuditFieldSourceSlug] = v.SourceSlug
	case *model.Example:
		changes[types.AuditFieldExampleID] = v.ID.String()
		changes[types.AuditFieldSentence] = v.Sentence
		if v.Translation != nil {
			changes[types.AuditFieldTranslation] = *v.Translation
		}
		changes[types.AuditFieldSourceSlug] = v.SourceSlug
	case *model.Image:
		changes[types.AuditFieldImageID] = v.ID.String()
		changes[types.AuditFieldURL] = v.URL
		if v.Caption != nil {
			changes[types.AuditFieldCaption] = *v.Caption
		}
		changes[types.AuditFieldSourceSlug] = v.SourceSlug
	case *model.Pronunciation:
		changes[types.AuditFieldPronunciationID] = v.ID.String()
		if v.AudioURL != nil {
			changes[types.AuditFieldAudioURL] = *v.AudioURL
		}
		changes[types.AuditFieldTranscription] = v.Transcription
		if v.Region != nil {
			changes[types.AuditFieldRegion] = *v.Region
		}
		changes[types.AuditFieldSourceSlug] = v.SourceSlug
	}

	return changes
}

// Helper functions for comparison

func equalStringPtr(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func equalPartOfSpeechPtr(a, b *model.PartOfSpeech) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func equalTimePtr(a, b *time.Time) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Equal(*b)
}

func formatTimePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}

// buildBulkChanges создает структуру изменений для массовых операций.
// Используется когда создается/удаляется несколько сущностей одновременно.
func buildBulkChanges(entityType string, created []any, deleted []any) model.JSON {
	changes := make(model.JSON)

	if len(created) > 0 {
		createdData := make([]model.JSON, 0, len(created))
		for _, entity := range created {
			createdData = append(createdData, buildCreateChanges(entity))
		}
		changes["created_"+entityType] = createdData
		changes["created_"+entityType+"_count"] = len(created)
	}

	if len(deleted) > 0 {
		deletedData := make([]model.JSON, 0, len(deleted))
		for _, entity := range deleted {
			deletedData = append(deletedData, buildDeleteChanges(entity))
		}
		changes["deleted_"+entityType] = deletedData
		changes["deleted_"+entityType+"_count"] = len(deleted)
	}

	return changes
}

// reflectDiff использует рефлексию для сравнения двух структур и возвращает изменения.
// Это универсальная функция, которая может сравнивать любые структуры.
// Используется как fallback, если нет специализированной функции diff.
func reflectDiff(old, new any) model.JSON {
	changes := make(model.JSON)

	oldVal := reflect.ValueOf(old)
	newVal := reflect.ValueOf(new)

	// Проверяем, что это указатели на структуры
	if oldVal.Kind() != reflect.Ptr || newVal.Kind() != reflect.Ptr {
		return changes
	}

	oldVal = oldVal.Elem()
	newVal = newVal.Elem()

	if oldVal.Type() != newVal.Type() {
		return changes
	}

	// Сравниваем каждое поле
	for i := 0; i < oldVal.NumField(); i++ {
		field := oldVal.Type().Field(i)
		oldField := oldVal.Field(i)
		newField := newVal.Field(i)

		// Пропускаем приватные поля и служебные поля (ID, CreatedAt, UpdatedAt)
		if !field.IsExported() {
			continue
		}

		fieldName := field.Tag.Get("json")
		if fieldName == "" {
			fieldName = field.Name
		}
		if fieldName == "-" {
			continue
		}

		// Пропускаем служебные поля
		if fieldName == "id" || fieldName == "created_at" || fieldName == "updated_at" {
			continue
		}

		// Сравниваем значения
		if !reflect.DeepEqual(oldField.Interface(), newField.Interface()) {
			oldValue := formatReflectValue(oldField)
			newValue := formatReflectValue(newField)

			changes[fieldName] = map[string]any{
				"old": oldValue,
				"new": newValue,
			}
		}
	}

	return changes
}

// formatReflectValue форматирует значение из рефлексии в JSON-совместимый тип.
func formatReflectValue(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}

	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return nil
		}
		return formatReflectValue(v.Elem())
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint()
	case reflect.Float32, reflect.Float64:
		return v.Float()
	case reflect.Bool:
		return v.Bool()
	case reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return v.Interface()
	default:
		// Для сложных типов (структуры, слайсы) возвращаем строковое представление
		return fmt.Sprintf("%v", v.Interface())
	}
}
