-- ---------------------------------------------------------------------------
-- cards (FSRS-5)
-- ---------------------------------------------------------------------------

-- name: GetCardByID :one
SELECT id, user_id, entry_id, state, step, stability, difficulty,
       due, last_review, reps, lapses, scheduled_days, elapsed_days,
       created_at, updated_at
FROM cards
WHERE id = @id AND user_id = @user_id;

-- name: GetCardByEntryID :one
SELECT id, user_id, entry_id, state, step, stability, difficulty,
       due, last_review, reps, lapses, scheduled_days, elapsed_days,
       created_at, updated_at
FROM cards
WHERE entry_id = @entry_id AND user_id = @user_id;

-- name: CreateCard :one
INSERT INTO cards (id, user_id, entry_id, state, due, created_at, updated_at)
VALUES (@id, @user_id, @entry_id, 'NEW', now(), @created_at, @updated_at)
RETURNING id, user_id, entry_id, state, step, stability, difficulty,
          due, last_review, reps, lapses, scheduled_days, elapsed_days,
          created_at, updated_at;

-- name: UpdateCardSRS :one
UPDATE cards
SET state = @state,
    step = @step,
    stability = @stability,
    difficulty = @difficulty,
    due = @due,
    last_review = @last_review,
    reps = @reps,
    lapses = @lapses,
    scheduled_days = @scheduled_days,
    elapsed_days = @elapsed_days,
    updated_at = now()
WHERE id = @id AND user_id = @user_id
RETURNING id, user_id, entry_id, state, step, stability, difficulty,
          due, last_review, reps, lapses, scheduled_days, elapsed_days,
          created_at, updated_at;

-- name: DeleteCard :execrows
DELETE FROM cards
WHERE id = @id AND user_id = @user_id;
