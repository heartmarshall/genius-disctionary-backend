-- ---------------------------------------------------------------------------
-- audit_log
-- ---------------------------------------------------------------------------

-- name: CreateAuditRecord :one
INSERT INTO audit_log (id, user_id, entity_type, entity_id, action, changes, created_at)
VALUES (@id, @user_id, @entity_type, @entity_id, @action, @changes, @created_at)
RETURNING id, user_id, entity_type, entity_id, action, changes, created_at;

-- name: GetByEntity :many
SELECT id, user_id, entity_type, entity_id, action, changes, created_at
FROM audit_log
WHERE entity_type = @entity_type AND entity_id = @entity_id
ORDER BY created_at DESC
LIMIT @lim::int;

-- name: GetByUser :many
SELECT id, user_id, entity_type, entity_id, action, changes, created_at
FROM audit_log
WHERE user_id = @user_id
ORDER BY created_at DESC
LIMIT @lim::int OFFSET @off::int;
