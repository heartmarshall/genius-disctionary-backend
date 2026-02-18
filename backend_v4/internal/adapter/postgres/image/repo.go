// Package image implements the Image repository using PostgreSQL.
// It handles two types of images: catalog (M2M via entry_images) and
// user-uploaded (CRUD via user_images).
// Write queries use sqlc-generated code. Read queries use raw SQL with JOIN.
package image

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	postgres "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/image/sqlc"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// CatalogImageWithEntryID is the batch result type for GetCatalogByEntryIDs.
// It embeds domain.RefImage and adds EntryID for grouping by the caller.
type CatalogImageWithEntryID struct {
	EntryID uuid.UUID
	domain.RefImage
}

// UserImageWithEntryID is the batch result type for GetUserByEntryIDs.
// It embeds domain.UserImage and adds EntryID for grouping by the caller.
type UserImageWithEntryID struct {
	EntryID uuid.UUID
	domain.UserImage
}

// Repo provides image persistence backed by PostgreSQL.
type Repo struct {
	pool *pgxpool.Pool
}

// New creates a new image repository.
func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// ---------------------------------------------------------------------------
// Raw SQL for JOIN read queries (catalog images)
// ---------------------------------------------------------------------------

const getCatalogByEntryIDSQL = `
SELECT
    ri.id, ri.ref_entry_id, ri.url, ri.caption, ri.source_slug
FROM entry_images ei
JOIN ref_images ri ON ei.ref_image_id = ri.id
WHERE ei.entry_id = $1`

const getCatalogByEntryIDsSQL = `
SELECT
    ei.entry_id,
    ri.id, ri.ref_entry_id, ri.url, ri.caption, ri.source_slug
FROM entry_images ei
JOIN ref_images ri ON ei.ref_image_id = ri.id
WHERE ei.entry_id = ANY($1::uuid[])
ORDER BY ei.entry_id`

// ---------------------------------------------------------------------------
// Raw SQL for read queries (user images)
// ---------------------------------------------------------------------------

const getUserByEntryIDSQL = `
SELECT id, entry_id, url, caption, created_at
FROM user_images
WHERE entry_id = $1
ORDER BY created_at`

const getUserByEntryIDsSQL = `
SELECT id, entry_id, url, caption, created_at
FROM user_images
WHERE entry_id = ANY($1::uuid[])
ORDER BY entry_id, created_at`

// ---------------------------------------------------------------------------
// Catalog image read operations
// ---------------------------------------------------------------------------

// GetCatalogByEntryID returns all ref_images linked to an entry via the M2M table.
// Returns an empty slice (not nil) when no images are linked.
func (r *Repo) GetCatalogByEntryID(ctx context.Context, entryID uuid.UUID) ([]domain.RefImage, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	rows, err := querier.Query(ctx, getCatalogByEntryIDSQL, entryID)
	if err != nil {
		return nil, fmt.Errorf("get catalog images by entry_id: %w", err)
	}
	defer rows.Close()

	result, err := scanCatalogImages(rows)
	if err != nil {
		return nil, fmt.Errorf("get catalog images by entry_id: %w", err)
	}

	return result, nil
}

// GetCatalogByEntryIDs returns catalog images for multiple entries (batch for DataLoader).
// Results include EntryID for grouping by the caller.
func (r *Repo) GetCatalogByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]CatalogImageWithEntryID, error) {
	if len(entryIDs) == 0 {
		return []CatalogImageWithEntryID{}, nil
	}

	querier := postgres.QuerierFromCtx(ctx, r.pool)

	rows, err := querier.Query(ctx, getCatalogByEntryIDsSQL, entryIDs)
	if err != nil {
		return nil, fmt.Errorf("get catalog images by entry_ids: %w", err)
	}
	defer rows.Close()

	result, err := scanCatalogImagesWithEntryID(rows)
	if err != nil {
		return nil, fmt.Errorf("get catalog images by entry_ids: %w", err)
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// Catalog image write operations
// ---------------------------------------------------------------------------

// LinkCatalog creates an M2M link between an entry and a ref_image.
// Idempotent: linking the same pair twice is NOT an error (ON CONFLICT DO NOTHING).
func (r *Repo) LinkCatalog(ctx context.Context, entryID, refImageID uuid.UUID) error {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	err := q.LinkCatalog(ctx, sqlc.LinkCatalogParams{
		EntryID:    entryID,
		RefImageID: refImageID,
	})
	if err != nil {
		return mapError(err, "entry_image", entryID)
	}

	return nil
}

// UnlinkCatalog removes the M2M link between an entry and a ref_image.
// Not an error if the link does not exist (0 rows affected is OK).
func (r *Repo) UnlinkCatalog(ctx context.Context, entryID, refImageID uuid.UUID) error {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	err := q.UnlinkCatalog(ctx, sqlc.UnlinkCatalogParams{
		EntryID:    entryID,
		RefImageID: refImageID,
	})
	if err != nil {
		return mapError(err, "entry_image", entryID)
	}

	return nil
}

// ---------------------------------------------------------------------------
// User image read operations
// ---------------------------------------------------------------------------

const getUserByIDSQL = `
SELECT id, entry_id, url, caption, created_at
FROM user_images
WHERE id = $1`

const getUserByIDForUserSQL = `
SELECT ui.id, ui.entry_id, ui.url, ui.caption, ui.created_at
FROM user_images ui
JOIN entries e ON e.id = ui.entry_id
WHERE ui.id = $1 AND e.user_id = $2 AND e.deleted_at IS NULL`

const countUserByEntrySQL = `
SELECT COUNT(*) FROM user_images WHERE entry_id = $1`

// GetUserByIDForUser returns a single user-uploaded image by its ID,
// verifying that the parent entry belongs to the given user (single JOIN query).
// Returns domain.ErrNotFound if the image does not exist or the entry is not owned.
func (r *Repo) GetUserByIDForUser(ctx context.Context, userID, imageID uuid.UUID) (*domain.UserImage, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	var (
		id        uuid.UUID
		entryID   uuid.UUID
		url       string
		caption   pgtype.Text
		createdAt time.Time
	)

	err := querier.QueryRow(ctx, getUserByIDForUserSQL, imageID, userID).Scan(&id, &entryID, &url, &caption, &createdAt)
	if err != nil {
		return nil, mapError(err, "user_image", imageID)
	}

	img := buildDomainUserImage(id, entryID, url, caption, createdAt)
	return &img, nil
}

// CountUserByEntry returns the number of user-uploaded images for an entry.
func (r *Repo) CountUserByEntry(ctx context.Context, entryID uuid.UUID) (int, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	var count int64
	err := querier.QueryRow(ctx, countUserByEntrySQL, entryID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count user images: %w", err)
	}

	return int(count), nil
}

// GetUserByID returns a single user-uploaded image by its ID.
// Returns domain.ErrNotFound if the image does not exist.
func (r *Repo) GetUserByID(ctx context.Context, imageID uuid.UUID) (*domain.UserImage, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	var (
		id        uuid.UUID
		entryID   uuid.UUID
		url       string
		caption   pgtype.Text
		createdAt time.Time
	)

	err := querier.QueryRow(ctx, getUserByIDSQL, imageID).Scan(&id, &entryID, &url, &caption, &createdAt)
	if err != nil {
		return nil, mapError(err, "user_image", imageID)
	}

	img := buildDomainUserImage(id, entryID, url, caption, createdAt)
	return &img, nil
}

// GetUserByEntryID returns all user-uploaded images for an entry.
// Returns an empty slice (not nil) when no images exist.
func (r *Repo) GetUserByEntryID(ctx context.Context, entryID uuid.UUID) ([]domain.UserImage, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	rows, err := querier.Query(ctx, getUserByEntryIDSQL, entryID)
	if err != nil {
		return nil, fmt.Errorf("get user images by entry_id: %w", err)
	}
	defer rows.Close()

	result, err := scanUserImages(rows)
	if err != nil {
		return nil, fmt.Errorf("get user images by entry_id: %w", err)
	}

	return result, nil
}

// GetUserByEntryIDs returns user images for multiple entries (batch for DataLoader).
// Results include EntryID for grouping by the caller.
func (r *Repo) GetUserByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]UserImageWithEntryID, error) {
	if len(entryIDs) == 0 {
		return []UserImageWithEntryID{}, nil
	}

	querier := postgres.QuerierFromCtx(ctx, r.pool)

	rows, err := querier.Query(ctx, getUserByEntryIDsSQL, entryIDs)
	if err != nil {
		return nil, fmt.Errorf("get user images by entry_ids: %w", err)
	}
	defer rows.Close()

	result, err := scanUserImagesWithEntryID(rows)
	if err != nil {
		return nil, fmt.Errorf("get user images by entry_ids: %w", err)
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// User image write operations
// ---------------------------------------------------------------------------

const updateUserCaptionSQL = `
UPDATE user_images SET caption = $1
WHERE id = $2
RETURNING id, entry_id, url, caption, created_at`

// UpdateUser updates the caption of a user image.
// Returns domain.ErrNotFound if the image does not exist.
func (r *Repo) UpdateUser(ctx context.Context, imageID uuid.UUID, caption *string) (*domain.UserImage, error) {
	querier := postgres.QuerierFromCtx(ctx, r.pool)

	var (
		id        uuid.UUID
		entryID   uuid.UUID
		url       string
		cap       pgtype.Text
		createdAt time.Time
	)

	err := querier.QueryRow(ctx, updateUserCaptionSQL, ptrStringToPgText(caption), imageID).
		Scan(&id, &entryID, &url, &cap, &createdAt)
	if err != nil {
		return nil, mapError(err, "user_image", imageID)
	}

	img := buildDomainUserImage(id, entryID, url, cap, createdAt)
	return &img, nil
}

// CreateUser creates a user-uploaded image and returns the persisted domain.UserImage.
func (r *Repo) CreateUser(ctx context.Context, entryID uuid.UUID, url string, caption *string) (*domain.UserImage, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	now := time.Now().UTC().Truncate(time.Microsecond)
	row, err := q.CreateUserImage(ctx, sqlc.CreateUserImageParams{
		ID:        uuid.New(),
		EntryID:   entryID,
		Url:       url,
		Caption:   ptrStringToPgText(caption),
		CreatedAt: now,
	})
	if err != nil {
		return nil, mapError(err, "user_image", uuid.Nil)
	}

	img := toDomainUserImage(row)
	return &img, nil
}

// DeleteUser removes a user image by ID. Returns domain.ErrNotFound if the image
// does not exist.
func (r *Repo) DeleteUser(ctx context.Context, imageID uuid.UUID) error {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	rowsAffected, err := q.DeleteUserImage(ctx, imageID)
	if err != nil {
		return mapError(err, "user_image", imageID)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user_image %s: %w", imageID, domain.ErrNotFound)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Row scanning helpers (catalog images)
// ---------------------------------------------------------------------------

// scanCatalogImages scans multiple rows from GetCatalogByEntryID into domain.RefImage slices.
func scanCatalogImages(rows pgx.Rows) ([]domain.RefImage, error) {
	var result []domain.RefImage
	for rows.Next() {
		img, err := scanCatalogImageFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, img)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if result == nil {
		result = []domain.RefImage{}
	}

	return result, nil
}

// scanCatalogImageFromRows scans a single row from pgx.Rows into a domain.RefImage.
func scanCatalogImageFromRows(rows pgx.Rows) (domain.RefImage, error) {
	var (
		id         uuid.UUID
		refEntryID uuid.UUID
		url        string
		caption    pgtype.Text
		sourceSlug string
	)

	if err := rows.Scan(&id, &refEntryID, &url, &caption, &sourceSlug); err != nil {
		return domain.RefImage{}, err
	}

	return buildDomainRefImage(id, refEntryID, url, caption, sourceSlug), nil
}

// scanCatalogImagesWithEntryID scans multiple rows from GetCatalogByEntryIDs into CatalogImageWithEntryID slices.
func scanCatalogImagesWithEntryID(rows pgx.Rows) ([]CatalogImageWithEntryID, error) {
	var result []CatalogImageWithEntryID
	for rows.Next() {
		var (
			entryID    uuid.UUID
			id         uuid.UUID
			refEntryID uuid.UUID
			url        string
			caption    pgtype.Text
			sourceSlug string
		)

		if err := rows.Scan(&entryID, &id, &refEntryID, &url, &caption, &sourceSlug); err != nil {
			return nil, err
		}

		result = append(result, CatalogImageWithEntryID{
			EntryID:  entryID,
			RefImage: buildDomainRefImage(id, refEntryID, url, caption, sourceSlug),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if result == nil {
		result = []CatalogImageWithEntryID{}
	}

	return result, nil
}

// buildDomainRefImage constructs a domain.RefImage from scanned values.
func buildDomainRefImage(id, refEntryID uuid.UUID, url string, caption pgtype.Text, sourceSlug string) domain.RefImage {
	img := domain.RefImage{
		ID:         id,
		RefEntryID: refEntryID,
		URL:        url,
		SourceSlug: sourceSlug,
	}

	if caption.Valid {
		img.Caption = &caption.String
	}

	return img
}

// ---------------------------------------------------------------------------
// Row scanning helpers (user images)
// ---------------------------------------------------------------------------

// scanUserImages scans multiple rows from GetUserByEntryID into domain.UserImage slices.
func scanUserImages(rows pgx.Rows) ([]domain.UserImage, error) {
	var result []domain.UserImage
	for rows.Next() {
		img, err := scanUserImageFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, img)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if result == nil {
		result = []domain.UserImage{}
	}

	return result, nil
}

// scanUserImageFromRows scans a single row from pgx.Rows into a domain.UserImage.
func scanUserImageFromRows(rows pgx.Rows) (domain.UserImage, error) {
	var (
		id        uuid.UUID
		entryID   uuid.UUID
		url       string
		caption   pgtype.Text
		createdAt time.Time
	)

	if err := rows.Scan(&id, &entryID, &url, &caption, &createdAt); err != nil {
		return domain.UserImage{}, err
	}

	return buildDomainUserImage(id, entryID, url, caption, createdAt), nil
}

// scanUserImagesWithEntryID scans multiple rows from GetUserByEntryIDs into UserImageWithEntryID slices.
func scanUserImagesWithEntryID(rows pgx.Rows) ([]UserImageWithEntryID, error) {
	var result []UserImageWithEntryID
	for rows.Next() {
		var (
			id        uuid.UUID
			entryID   uuid.UUID
			url       string
			caption   pgtype.Text
			createdAt time.Time
		)

		if err := rows.Scan(&id, &entryID, &url, &caption, &createdAt); err != nil {
			return nil, err
		}

		result = append(result, UserImageWithEntryID{
			EntryID:   entryID,
			UserImage: buildDomainUserImage(id, entryID, url, caption, createdAt),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if result == nil {
		result = []UserImageWithEntryID{}
	}

	return result, nil
}

// buildDomainUserImage constructs a domain.UserImage from scanned values.
func buildDomainUserImage(id, entryID uuid.UUID, url string, caption pgtype.Text, createdAt time.Time) domain.UserImage {
	img := domain.UserImage{
		ID:        id,
		EntryID:   entryID,
		URL:       url,
		CreatedAt: createdAt,
	}

	if caption.Valid {
		img.Caption = &caption.String
	}

	return img
}

// ---------------------------------------------------------------------------
// Mapping helpers: sqlc -> domain (for write query results)
// ---------------------------------------------------------------------------

// toDomainUserImage converts a sqlc.UserImage row into a domain.UserImage.
func toDomainUserImage(row sqlc.UserImage) domain.UserImage {
	img := domain.UserImage{
		ID:        row.ID,
		EntryID:   row.EntryID,
		URL:       row.Url,
		CreatedAt: row.CreatedAt,
	}

	if row.Caption.Valid {
		img.Caption = &row.Caption.String
	}

	return img
}

// ---------------------------------------------------------------------------
// Error mapping
// ---------------------------------------------------------------------------

// mapError converts pgx/pgconn errors into domain errors.
func mapError(err error, entity string, id uuid.UUID) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s %s: %w", entity, id, err)
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s %s: %w", entity, id, domain.ErrNotFound)
	}

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

	return fmt.Errorf("%s %s: %w", entity, id, err)
}

// ---------------------------------------------------------------------------
// pgtype helpers
// ---------------------------------------------------------------------------

// ptrStringToPgText converts a *string to pgtype.Text (nil -> NULL).
func ptrStringToPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}
