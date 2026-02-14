package dictionary

import (
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// FindResult contains the result of a paginated entry search.
type FindResult struct {
	Entries     []domain.Entry
	TotalCount  int
	HasNextPage bool
	PageInfo    *PageInfo
}

// PageInfo contains cursor information for cursor-based pagination.
type PageInfo struct {
	StartCursor *string
	EndCursor   *string
}

// BatchResult contains the result of a batch delete operation.
type BatchResult struct {
	Deleted int
	Errors  []BatchError
}

// BatchError describes a single failure in a batch operation.
type BatchError struct {
	EntryID uuid.UUID
	Error   string
}

// ImportResult contains the result of an import operation.
type ImportResult struct {
	Imported int
	Skipped  int
	Errors   []ImportError
}

// ImportError describes a single failure during import.
type ImportError struct {
	LineNumber int
	Text       string
	Reason     string
}

// ExportResult contains the exported dictionary data.
type ExportResult struct {
	Items      []ExportItem
	ExportedAt time.Time
}

// ExportItem represents a single exported entry with its related data.
type ExportItem struct {
	Text       string
	Notes      *string
	Senses     []ExportSense
	CardStatus *domain.LearningStatus
	CreatedAt  time.Time
}

// ExportSense represents a sense in an exported entry.
type ExportSense struct {
	Definition   *string
	PartOfSpeech *domain.PartOfSpeech
	Translations []string
	Examples     []ExportExample
}

// ExportExample represents an example in an exported sense.
type ExportExample struct {
	Sentence    string
	Translation *string
}
