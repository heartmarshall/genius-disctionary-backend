-- +goose Up

-- Step 1: Add username column to users (nullable first, will be populated).
ALTER TABLE users ADD COLUMN username TEXT;

-- Step 2: Create auth_methods table.
CREATE TABLE auth_methods (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    method        TEXT NOT NULL,
    provider_id   TEXT,
    password_hash TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One row per (method, provider_id) for OAuth methods.
CREATE UNIQUE INDEX ux_auth_methods_oauth
    ON auth_methods(method, provider_id)
    WHERE method <> 'password' AND provider_id IS NOT NULL;

-- One password method per user.
CREATE UNIQUE INDEX ux_auth_methods_password
    ON auth_methods(user_id)
    WHERE method = 'password';

CREATE INDEX ix_auth_methods_user ON auth_methods(user_id);

-- Step 3: Migrate existing OAuth data from users to auth_methods.
INSERT INTO auth_methods (user_id, method, provider_id, created_at, updated_at)
SELECT id, oauth_provider, oauth_id, created_at, updated_at
FROM users
WHERE oauth_provider IS NOT NULL AND oauth_provider <> '';

-- Step 4: Populate username from existing name (or email prefix as fallback).
UPDATE users
SET username = COALESCE(
    NULLIF(TRIM(name), ''),
    SPLIT_PART(email, '@', 1)
);

-- Step 5: Make username NOT NULL and add unique index.
ALTER TABLE users ALTER COLUMN username SET NOT NULL;
CREATE UNIQUE INDEX ux_users_username ON users(username);

-- Step 6: Drop old OAuth columns from users.
DROP INDEX IF EXISTS ux_users_oauth;
ALTER TABLE users DROP COLUMN IF EXISTS oauth_provider;
ALTER TABLE users DROP COLUMN IF EXISTS oauth_id;

-- +goose Down
ALTER TABLE users ADD COLUMN oauth_provider TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN oauth_id       TEXT NOT NULL DEFAULT '';

UPDATE users u
SET oauth_provider = am.method,
    oauth_id       = COALESCE(am.provider_id, '')
FROM auth_methods am
WHERE am.user_id = u.id
  AND am.method <> 'password';

CREATE UNIQUE INDEX ux_users_oauth ON users(oauth_provider, oauth_id);

DROP INDEX IF EXISTS ux_users_username;
ALTER TABLE users DROP COLUMN IF EXISTS username;
DROP TABLE IF EXISTS auth_methods;
