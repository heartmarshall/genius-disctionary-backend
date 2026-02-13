package domain

import (
	"time"

	"github.com/google/uuid"
)

// Topic is a user-defined category for grouping dictionary entries.
type Topic struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Name        string
	Description *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	EntryCount  int // computed field, not stored in DB
}

// InboxItem is a quick note saved for later processing.
type InboxItem struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Text      string
	Context   *string
	CreatedAt time.Time
}

// AuditRecord logs a mutation event on a domain entity.
type AuditRecord struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	EntityType EntityType
	EntityID   *uuid.UUID
	Action     AuditAction
	Changes    map[string]any
	CreatedAt  time.Time
}
