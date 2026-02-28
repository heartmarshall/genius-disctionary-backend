package domain

import "github.com/google/uuid"

// EntryFilter contains filtering/pagination parameters for entry searches.
type EntryFilter struct {
	Search       *string
	HasCard      *bool
	PartOfSpeech *PartOfSpeech
	TopicID      *uuid.UUID
	Status       *CardState
	SortBy       string
	SortOrder    string
	Limit        int
	Cursor       *string
	Offset       *int
}

// ReorderItem represents an item to reorder with its new position.
type ReorderItem struct {
	ID       uuid.UUID
	Position int
}
