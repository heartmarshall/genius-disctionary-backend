-- ---------------------------------------------------------------------------
-- review_logs
-- ---------------------------------------------------------------------------

-- name: CreateReviewLog :one
INSERT INTO review_logs (id, card_id, user_id, grade, prev_state, duration_ms, reviewed_at)
VALUES (@id, @card_id, @user_id, @grade, @prev_state, @duration_ms, @reviewed_at)
RETURNING id, card_id, user_id, grade, prev_state, duration_ms, reviewed_at;

-- name: GetByCardID :many
SELECT id, card_id, user_id, grade, prev_state, duration_ms, reviewed_at
FROM review_logs
WHERE card_id = @card_id
ORDER BY reviewed_at DESC
LIMIT @lim::int OFFSET @off::int;

-- name: GetLastByCardID :one
SELECT id, card_id, user_id, grade, prev_state, duration_ms, reviewed_at
FROM review_logs
WHERE card_id = @card_id
ORDER BY reviewed_at DESC
LIMIT 1;

-- name: DeleteReviewLog :execrows
DELETE FROM review_logs
WHERE id = @id;
