package entry

import (
	"github.com/google/uuid"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// Filter defines parameters for searching and paginating user entries.
type Filter struct {
	// Search performs ILIKE '%...%' on text_normalized (uses GIN trgm index).
	// nil or empty string means no text filter.
	Search *string

	// HasCard filters entries that have (true) or don't have (false) an associated card.
	HasCard *bool

	// PartOfSpeech filters entries that have at least one sense with the given POS.
	// Uses COALESCE(s.part_of_speech, rs.part_of_speech) to check both user and ref senses.
	PartOfSpeech *domain.PartOfSpeech

	// TopicID filters entries that belong to the given topic.
	TopicID *uuid.UUID

	// Status filters entries that have a card with the given learning status.
	Status *domain.LearningStatus

	// SortBy determines the sort column: "text", "created_at", "updated_at".
	// Default: "created_at".
	SortBy string

	// SortOrder: "ASC" or "DESC". Default: "DESC".
	SortOrder string

	// Limit is the maximum number of entries to return. Default: 50, max: 200.
	Limit int

	// Offset is the number of entries to skip (offset-based pagination).
	Offset int

	// Cursor is an opaque cursor for keyset pagination.
	// Format: base64(sort_value + "|" + entry_id).
	// When set, offset is ignored and keyset pagination is used.
	Cursor *string
}

const (
	defaultLimit = 50
	maxLimit     = 200

	sortByText      = "text"
	sortByCreatedAt = "created_at"
	sortByUpdatedAt = "updated_at"

	sortOrderASC  = "ASC"
	sortOrderDESC = "DESC"
)

// normalize applies defaults and clamps values.
func (f *Filter) normalize() {
	// Sort column.
	switch f.SortBy {
	case sortByText, sortByCreatedAt, sortByUpdatedAt:
		// valid
	default:
		f.SortBy = sortByCreatedAt
	}

	// Sort order.
	switch f.SortOrder {
	case sortOrderASC, sortOrderDESC:
		// valid
	default:
		f.SortOrder = sortOrderDESC
	}

	// Limit.
	if f.Limit <= 0 {
		f.Limit = defaultLimit
	}
	if f.Limit > maxLimit {
		f.Limit = maxLimit
	}

	// Offset cannot be negative.
	if f.Offset < 0 {
		f.Offset = 0
	}
}

// isCursor returns true if cursor-based pagination is requested.
func (f *Filter) isCursor() bool {
	return f.Cursor != nil
}

// sortColumn returns the SQL column name for the current SortBy value.
func (f *Filter) sortColumn() string {
	switch f.SortBy {
	case sortByText:
		return "text_normalized"
	case sortByUpdatedAt:
		return "updated_at"
	default:
		return "created_at"
	}
}
