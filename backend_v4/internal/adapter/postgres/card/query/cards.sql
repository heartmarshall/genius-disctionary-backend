-- ---------------------------------------------------------------------------
-- cards
-- ---------------------------------------------------------------------------

-- name: GetCardByID :one
SELECT id, user_id, entry_id, status, learning_step,
       next_review_at, interval_days, ease_factor, created_at, updated_at
FROM cards
WHERE id = @id AND user_id = @user_id;

-- name: GetCardByEntryID :one
SELECT id, user_id, entry_id, status, learning_step,
       next_review_at, interval_days, ease_factor, created_at, updated_at
FROM cards
WHERE entry_id = @entry_id AND user_id = @user_id;

-- name: CreateCard :one
INSERT INTO cards (id, user_id, entry_id, status, ease_factor, created_at, updated_at)
VALUES (@id, @user_id, @entry_id, @status, @ease_factor, @created_at, @updated_at)
RETURNING id, user_id, entry_id, status, learning_step,
          next_review_at, interval_days, ease_factor, created_at, updated_at;

-- name: UpdateCardSRS :execrows
UPDATE cards
SET status = @status,
    next_review_at = @next_review_at,
    interval_days = @interval_days,
    ease_factor = @ease_factor,
    learning_step = @learning_step,
    updated_at = now()
WHERE id = @id AND user_id = @user_id;

-- name: DeleteCard :execrows
DELETE FROM cards
WHERE id = @id AND user_id = @user_id;
