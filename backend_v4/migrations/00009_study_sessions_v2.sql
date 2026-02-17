-- +goose Up
ALTER TABLE study_sessions DROP COLUMN IF EXISTS cards_studied;
ALTER TABLE study_sessions DROP COLUMN IF EXISTS abandoned_at;

-- Session status: ACTIVE, FINISHED, ABANDONED
-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE session_status AS ENUM ('ACTIVE', 'FINISHED', 'ABANDONED');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

ALTER TABLE study_sessions ADD COLUMN status session_status NOT NULL DEFAULT 'ACTIVE';
ALTER TABLE study_sessions ADD COLUMN result JSONB;
ALTER TABLE study_sessions ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- Only one active session per user
CREATE UNIQUE INDEX IF NOT EXISTS ux_study_sessions_active ON study_sessions(user_id) WHERE status = 'ACTIVE';

-- +goose Down
DROP INDEX IF EXISTS ux_study_sessions_active;
ALTER TABLE study_sessions DROP COLUMN IF EXISTS created_at;
ALTER TABLE study_sessions DROP COLUMN IF EXISTS result;
ALTER TABLE study_sessions DROP COLUMN IF EXISTS status;
DROP TYPE IF EXISTS session_status;
ALTER TABLE study_sessions ADD COLUMN abandoned_at TIMESTAMPTZ;
ALTER TABLE study_sessions ADD COLUMN cards_studied INT NOT NULL DEFAULT 0;
