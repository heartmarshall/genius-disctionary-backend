package model

import (
	"database/sql/driver"
	"fmt"
)

// PartOfSpeech corresponds to the Postgres ENUM part_of_speech
type PartOfSpeech string

const (
	PosNoun         PartOfSpeech = "NOUN"
	PosVerb         PartOfSpeech = "VERB"
	PosAdjective    PartOfSpeech = "ADJECTIVE"
	PosAdverb       PartOfSpeech = "ADVERB"
	PosPronoun      PartOfSpeech = "PRONOUN"
	PosPreposition  PartOfSpeech = "PREPOSITION"
	PosConjunction  PartOfSpeech = "CONJUNCTION"
	PosInterjection PartOfSpeech = "INTERJECTION"
	PosPhrase       PartOfSpeech = "PHRASE"
	PosIdiom        PartOfSpeech = "IDIOM"
	PosOther        PartOfSpeech = "OTHER"
)

// LearningStatus corresponds to the Postgres ENUM learning_status
type LearningStatus string

const (
	StatusNew      LearningStatus = "NEW"
	StatusLearning LearningStatus = "LEARNING"
	StatusReview   LearningStatus = "REVIEW"
	StatusMastered LearningStatus = "MASTERED"
)

// ReviewGrade corresponds to the Postgres ENUM review_grade
type ReviewGrade string

const (
	GradeAgain ReviewGrade = "AGAIN"
	GradeHard  ReviewGrade = "HARD"
	GradeGood  ReviewGrade = "GOOD"
	GradeEasy  ReviewGrade = "EASY"
)

// EntityType corresponds to the Postgres ENUM entity_type
type EntityType string

const (
	EntityEntry         EntityType = "ENTRY"
	EntitySense         EntityType = "SENSE"
	EntityExample       EntityType = "EXAMPLE"
	EntityImage         EntityType = "IMAGE"
	EntityPronunciation EntityType = "PRONUNCIATION"
	EntityCard          EntityType = "CARD"
)

// AuditAction corresponds to the Postgres ENUM audit_action
type AuditAction string

const (
	ActionCreate AuditAction = "CREATE"
	ActionUpdate AuditAction = "UPDATE"
	ActionDelete AuditAction = "DELETE"
)

// Validate checks if the Enum value is valid (useful for transport->service mapping)
func (s LearningStatus) IsValid() bool {
	switch s {
	case StatusNew, StatusLearning, StatusReview, StatusMastered:
		return true
	}
	return false
}

// Value implements driver.Valuer
func (s LearningStatus) Value() (driver.Value, error) {
	return string(s), nil
}

// Scan implements sql.Scanner
func (s *LearningStatus) Scan(value any) error {
	if value == nil {
		return nil
	}

	var str string
	switch v := value.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		return fmt.Errorf("failed to scan LearningStatus: %v (type %T)", value, value)
	}

	*s = LearningStatus(str)
	return nil
}

type WordSortField string

const (
	SortFieldCreatedAt WordSortField = "CREATED_AT"
	SortFieldText      WordSortField = "TEXT"
	SortFieldUpdatedAt WordSortField = "UPDATED_AT"
)

// SortDirection corresponds to GraphQL SortDirection
type SortDirection string

const (
	SortDirAsc  SortDirection = "ASC"
	SortDirDesc SortDirection = "DESC"
)
