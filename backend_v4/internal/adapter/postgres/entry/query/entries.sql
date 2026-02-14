-- name: GetEntryByID :one
SELECT id, user_id, ref_entry_id, text, text_normalized, notes,
       created_at, updated_at, deleted_at
FROM entries
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: GetEntryByText :one
SELECT id, user_id, ref_entry_id, text, text_normalized, notes,
       created_at, updated_at, deleted_at
FROM entries
WHERE user_id = $1 AND text_normalized = $2 AND deleted_at IS NULL;

-- name: GetEntriesByIDs :many
SELECT id, user_id, ref_entry_id, text, text_normalized, notes,
       created_at, updated_at, deleted_at
FROM entries
WHERE user_id = $1 AND id = ANY(@ids::uuid[]) AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: CountEntriesByUser :one
SELECT count(*) FROM entries
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: CreateEntry :one
INSERT INTO entries (id, user_id, ref_entry_id, text, text_normalized, notes, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, user_id, ref_entry_id, text, text_normalized, notes, created_at, updated_at, deleted_at;

-- name: UpdateEntryNotes :one
UPDATE entries
SET notes = $3, updated_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
RETURNING id, user_id, ref_entry_id, text, text_normalized, notes, created_at, updated_at, deleted_at;

-- name: SoftDeleteEntry :exec
UPDATE entries
SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: RestoreEntry :one
UPDATE entries
SET deleted_at = NULL, updated_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NOT NULL
RETURNING id, user_id, ref_entry_id, text, text_normalized, notes, created_at, updated_at, deleted_at;

-- name: HardDeleteOldEntries :execrows
DELETE FROM entries
WHERE id IN (
    SELECT e.id FROM entries e WHERE e.deleted_at < $1 LIMIT 100
);
