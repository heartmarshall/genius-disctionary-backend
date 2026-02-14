-- ---------------------------------------------------------------------------
-- inbox_items
-- ---------------------------------------------------------------------------

-- name: GetInboxItemByID :one
SELECT id, user_id, text, context, created_at
FROM inbox_items
WHERE id = @id AND user_id = @user_id;

-- name: ListInboxItemsByUser :many
SELECT id, user_id, text, context, created_at
FROM inbox_items
WHERE user_id = @user_id
ORDER BY created_at DESC
LIMIT @lim::int OFFSET @off::int;

-- name: CountInboxItemsByUser :one
SELECT count(*) FROM inbox_items
WHERE user_id = @user_id;

-- name: CreateInboxItem :one
INSERT INTO inbox_items (id, user_id, text, context, created_at)
VALUES (@id, @user_id, @text, @context, @created_at)
RETURNING id, user_id, text, context, created_at;

-- name: DeleteInboxItem :execrows
DELETE FROM inbox_items
WHERE id = @id AND user_id = @user_id;

-- name: DeleteAllInboxItems :exec
DELETE FROM inbox_items
WHERE user_id = @user_id;
