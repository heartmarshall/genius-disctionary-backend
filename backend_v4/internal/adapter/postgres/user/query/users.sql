-- name: GetUserByID :one
SELECT id, email, username, name, avatar_url, role, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, username, name, avatar_url, role, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByUsername :one
SELECT id, email, username, name, avatar_url, role, created_at, updated_at
FROM users
WHERE username = $1;

-- name: CreateUser :one
INSERT INTO users (id, email, username, name, avatar_url, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, email, username, name, avatar_url, role, created_at, updated_at;

-- name: UpdateUser :one
UPDATE users
SET name = $2, avatar_url = $3, updated_at = now()
WHERE id = $1
RETURNING id, email, username, name, avatar_url, role, created_at, updated_at;

-- name: GetUserSettings :one
SELECT user_id, new_cards_per_day, reviews_per_day, max_interval_days, timezone, updated_at
FROM user_settings
WHERE user_id = $1;

-- name: CreateUserSettings :one
INSERT INTO user_settings (user_id, new_cards_per_day, reviews_per_day, max_interval_days, timezone, updated_at)
VALUES ($1, $2, $3, $4, $5, now())
RETURNING user_id, new_cards_per_day, reviews_per_day, max_interval_days, timezone, updated_at;

-- name: UpdateUserSettings :one
UPDATE user_settings
SET new_cards_per_day = $2, reviews_per_day = $3, max_interval_days = $4, timezone = $5, updated_at = now()
WHERE user_id = $1
RETURNING user_id, new_cards_per_day, reviews_per_day, max_interval_days, timezone, updated_at;

-- name: UpdateUserRole :one
UPDATE users
SET role = $2, updated_at = now()
WHERE id = $1
RETURNING id, email, username, name, avatar_url, role, created_at, updated_at;

-- name: ListUsers :many
SELECT id, email, username, name, avatar_url, role, created_at, updated_at
FROM users
ORDER BY created_at
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT count(*) FROM users;
