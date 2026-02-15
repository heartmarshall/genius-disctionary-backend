package study

import "github.com/google/uuid"

// BatchCreateResult holds the outcome of a batch card creation.
type BatchCreateResult struct {
	Created         int
	SkippedExisting int
	SkippedNoSenses int
	Errors          []BatchCreateError
}

// BatchCreateError describes an error for a specific entry during batch creation.
type BatchCreateError struct {
	EntryID uuid.UUID
	Reason  string
}
