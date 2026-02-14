-- ---------------------------------------------------------------------------
-- topics
-- ---------------------------------------------------------------------------

-- name: GetTopicByID :one
SELECT id, user_id, name, description, created_at, updated_at
FROM topics
WHERE id = @id AND user_id = @user_id;

-- name: ListTopicsByUser :many
SELECT id, user_id, name, description, created_at, updated_at
FROM topics
WHERE user_id = @user_id
ORDER BY name;

-- name: CreateTopic :one
INSERT INTO topics (user_id, name, description)
VALUES (@user_id, @name, @description)
RETURNING id, user_id, name, description, created_at, updated_at;

-- name: UpdateTopic :execrows
UPDATE topics
SET name = @name, description = @description, updated_at = now()
WHERE id = @id AND user_id = @user_id;

-- name: DeleteTopic :execrows
DELETE FROM topics
WHERE id = @id AND user_id = @user_id;

-- ---------------------------------------------------------------------------
-- entry_topics (M2M)
-- ---------------------------------------------------------------------------

-- name: LinkEntry :exec
INSERT INTO entry_topics (entry_id, topic_id)
VALUES (@entry_id, @topic_id)
ON CONFLICT DO NOTHING;

-- name: UnlinkEntry :exec
DELETE FROM entry_topics
WHERE entry_id = @entry_id AND topic_id = @topic_id;

-- name: GetEntryIDsByTopicID :many
SELECT entry_id FROM entry_topics
WHERE topic_id = @topic_id;
