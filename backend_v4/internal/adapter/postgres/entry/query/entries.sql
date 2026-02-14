-- name: GetEntryByID :one
SELECT id, user_id, ref_entry_id, text, text_normalized, notes,
       created_at, updated_at, deleted_at
FROM entries
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;
