package types

// AuditField содержит константы для названий полей в аудит-логах.
// Использование констант вместо магических строк предотвращает опечатки
// и упрощает рефакторинг.

// ============================================================================
// COMMON FIELDS
// ============================================================================

const (
	// AuditFieldOld значение "old" для diff полей
	AuditFieldOld = "old"
	// AuditFieldNew значение "new" для diff полей
	AuditFieldNew = "new"
	// AuditFieldDeleted флаг удаления
	AuditFieldDeleted = "deleted"
	// AuditFieldAction тип действия (sense_added, examples_added и т.д.)
	AuditFieldAction = "action"
)

// ============================================================================
// DICTIONARY ENTRY FIELDS
// ============================================================================

const (
	AuditFieldText           = "text"
	AuditFieldTextNormalized = "text_normalized"
)

// ============================================================================
// SENSE FIELDS
// ============================================================================

const (
	AuditFieldDefinition   = "definition"
	AuditFieldPartOfSpeech = "part_of_speech"
	AuditFieldCefrLevel    = "cefr_level"
	AuditFieldSenseID      = "sense_id"
)

// ============================================================================
// TRANSLATION FIELDS
// ============================================================================

const (
	AuditFieldTranslationID = "translation_id"
	AuditFieldTranslations  = "translations"
)

// ============================================================================
// EXAMPLE FIELDS
// ============================================================================

const (
	AuditFieldExampleID = "example_id"
	AuditFieldSentence  = "sentence"
	AuditFieldTranslation = "translation"
)

// ============================================================================
// IMAGE FIELDS
// ============================================================================

const (
	AuditFieldImageID = "image_id"
	AuditFieldURL     = "url"
	AuditFieldCaption = "caption"
)

// ============================================================================
// PRONUNCIATION FIELDS
// ============================================================================

const (
	AuditFieldPronunciationID = "pronunciation_id"
	AuditFieldAudioURL       = "audio_url"
	AuditFieldTranscription  = "transcription"
	AuditFieldRegion         = "region"
)

// ============================================================================
// CARD FIELDS
// ============================================================================

const (
	AuditFieldEntryID      = "entry_id"
	AuditFieldStatus       = "status"
	AuditFieldNextReviewAt = "next_review_at"
	AuditFieldIntervalDays = "interval_days"
	AuditFieldEaseFactor   = "ease_factor"
)

// ============================================================================
// COMMON ENTITY FIELDS
// ============================================================================

const (
	AuditFieldSourceSlug = "source_slug"
)

// ============================================================================
// COUNT FIELDS
// ============================================================================

const (
	AuditFieldSensesCount         = "senses_count"
	AuditFieldTranslationsCount   = "translations_count"
	AuditFieldExamplesCount       = "examples_count"
	AuditFieldImagesCount         = "images_count"
	AuditFieldPronunciationsCount = "pronunciations_count"
	AuditFieldTopicsCount         = "topics_count"
)

// ============================================================================
// ACTION TYPES
// ============================================================================

const (
	AuditActionSenseAdded         = "sense_added"
	AuditActionExamplesAdded      = "examples_added"
	AuditActionTranslationsAdded  = "translations_added"
	AuditActionImagesAdded        = "images_added"
	AuditActionPronunciationsAdded = "pronunciations_added"
	AuditActionTopicsAdded        = "topics_added"
	AuditActionTopicsRemoved      = "topics_removed"
	AuditActionSenseDeleted       = "sense_deleted"
	AuditActionExampleDeleted     = "example_deleted"
	AuditActionTranslationDeleted = "translation_deleted"
	AuditActionImageDeleted       = "image_deleted"
	AuditActionPronunciationDeleted = "pronunciation_deleted"
)

// ============================================================================
// RECREATION FIELDS
// ============================================================================

const (
	AuditFieldSensesRecreated         = "senses_recreated"
	AuditFieldImagesRecreated         = "images_recreated"
	AuditFieldPronunciationsRecreated = "pronunciations_recreated"
	AuditFieldTopicsRecreated         = "topics_recreated"
	AuditFieldSensesOldCount          = "senses_old_count"
	AuditFieldSensesNewCount          = "senses_new_count"
	AuditFieldImagesOldCount          = "images_old_count"
	AuditFieldImagesNewCount          = "images_new_count"
	AuditFieldPronunciationsOldCount  = "pronunciations_old_count"
	AuditFieldPronunciationsNewCount  = "pronunciations_new_count"
	AuditFieldTopicsOldCount          = "topics_old_count"
	AuditFieldTopicsNewCount          = "topics_new_count"
)

// ============================================================================
// TOPIC FIELDS
// ============================================================================

const (
	AuditFieldTopicID = "topic_id"
	AuditFieldTopics  = "topics"
)

// ============================================================================
// CARD CREATION FLAG
// ============================================================================

const (
	AuditFieldCardCreated = "card_created"
)

// ============================================================================
// REVIEW FIELDS (for Study Service)
// ============================================================================

const (
	AuditFieldReviewGrade      = "review_grade"
	AuditFieldReviewDurationMs = "review_duration_ms"
	AuditFieldReviewedAt       = "reviewed_at"
	AuditFieldReviewLogID      = "review_log_id"
)

