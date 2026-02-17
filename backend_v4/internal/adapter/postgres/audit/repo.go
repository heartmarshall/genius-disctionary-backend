// Package audit implements the Audit repository using PostgreSQL.
// It provides append-only operations for audit log records.
package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	postgres "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/audit/sqlc"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// Repo provides audit log persistence backed by PostgreSQL.
type Repo struct {
	pool *pgxpool.Pool
}

// New creates a new audit repository.
func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// ---------------------------------------------------------------------------
// Write operations
// ---------------------------------------------------------------------------

// Create inserts a new audit record and returns the persisted domain.AuditRecord.
func (r *Repo) Create(ctx context.Context, record domain.AuditRecord) (domain.AuditRecord, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	changesJSON, err := json.Marshal(record.Changes)
	if err != nil {
		return domain.AuditRecord{}, fmt.Errorf("audit_record marshal changes: %w", err)
	}

	row, err := q.CreateAuditRecord(ctx, sqlc.CreateAuditRecordParams{
		ID:         record.ID,
		UserID:     record.UserID,
		EntityType: sqlc.EntityType(record.EntityType),
		EntityID:   uuidPtrToPgUUID(record.EntityID),
		Action:     sqlc.AuditAction(record.Action),
		Changes:    changesJSON,
		CreatedAt:  record.CreatedAt,
	})
	if err != nil {
		return domain.AuditRecord{}, mapError(err, "audit_record", record.ID)
	}

	return toDomainAuditRecord(row)
}

// Log creates an audit record without returning it (fire-and-forget).
// Satisfies study.auditLogger, topic.auditLogger, and content.auditRepo.
func (r *Repo) Log(ctx context.Context, record domain.AuditRecord) error {
	_, err := r.Create(ctx, record)
	return err
}

// ---------------------------------------------------------------------------
// Read operations
// ---------------------------------------------------------------------------

// GetByEntity returns the change history for a specific entity, ordered by
// created_at DESC, limited to `limit` records.
func (r *Repo) GetByEntity(ctx context.Context, entityType domain.EntityType, entityID uuid.UUID, limit int) ([]domain.AuditRecord, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	rows, err := q.GetByEntity(ctx, sqlc.GetByEntityParams{
		EntityType: sqlc.EntityType(entityType),
		EntityID:   pgtype.UUID{Bytes: entityID, Valid: true},
		Lim:        int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("get audit_records by entity: %w", err)
	}

	records := make([]domain.AuditRecord, len(rows))
	for i, row := range rows {
		rec, err := toDomainAuditRecord(row)
		if err != nil {
			return nil, err
		}
		records[i] = rec
	}

	return records, nil
}

// GetByUser returns audit log records for a user, ordered by created_at DESC
// with pagination.
func (r *Repo) GetByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.AuditRecord, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	rows, err := q.GetByUser(ctx, sqlc.GetByUserParams{
		UserID: userID,
		Lim:    int32(limit),
		Off:    int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("get audit_records by user: %w", err)
	}

	records := make([]domain.AuditRecord, len(rows))
	for i, row := range rows {
		rec, err := toDomainAuditRecord(row)
		if err != nil {
			return nil, err
		}
		records[i] = rec
	}

	return records, nil
}

// ---------------------------------------------------------------------------
// Error mapping
// ---------------------------------------------------------------------------

// mapError converts pgx/pgconn errors into domain errors.
func mapError(err error, entity string, id uuid.UUID) error {
	if err == nil {
		return nil
	}

	// context errors pass through as-is
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s %s: %w", entity, id, err)
	}

	// pgx.ErrNoRows -> domain.ErrNotFound
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s %s: %w", entity, id, domain.ErrNotFound)
	}

	// PgError codes
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return fmt.Errorf("%s %s: %w", entity, id, domain.ErrAlreadyExists)
		case "23503": // foreign_key_violation
			return fmt.Errorf("%s %s: %w", entity, id, domain.ErrNotFound)
		case "23514": // check_violation
			return fmt.Errorf("%s %s: %w", entity, id, domain.ErrValidation)
		}
	}

	// Everything else: wrap with context
	return fmt.Errorf("%s %s: %w", entity, id, err)
}

// ---------------------------------------------------------------------------
// Mapping helpers: sqlc -> domain
// ---------------------------------------------------------------------------

// toDomainAuditRecord converts a sqlc.AuditLog row into a domain.AuditRecord.
func toDomainAuditRecord(row sqlc.AuditLog) (domain.AuditRecord, error) {
	record := domain.AuditRecord{
		ID:         row.ID,
		UserID:     row.UserID,
		EntityType: domain.EntityType(row.EntityType),
		Action:     domain.AuditAction(row.Action),
		CreatedAt:  row.CreatedAt,
	}

	// entity_id: nullable UUID
	if row.EntityID.Valid {
		id := uuid.UUID(row.EntityID.Bytes)
		record.EntityID = &id
	}

	// changes: JSONB -> map[string]any
	if len(row.Changes) > 0 {
		changes := make(map[string]any)
		if err := json.Unmarshal(row.Changes, &changes); err != nil {
			return domain.AuditRecord{}, fmt.Errorf("audit_record %s unmarshal changes: %w", row.ID, err)
		}
		record.Changes = changes
	}

	return record, nil
}

// ---------------------------------------------------------------------------
// pgtype helpers
// ---------------------------------------------------------------------------

// uuidPtrToPgUUID converts a *uuid.UUID to pgtype.UUID (nil -> NULL).
func uuidPtrToPgUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}
