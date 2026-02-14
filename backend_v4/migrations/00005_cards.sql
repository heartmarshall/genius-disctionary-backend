-- +goose Up
CREATE TABLE cards (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entry_id       UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    status         learning_status NOT NULL DEFAULT 'NEW',
    learning_step  INT NOT NULL DEFAULT 0,
    next_review_at TIMESTAMPTZ,
    interval_days  INT NOT NULL DEFAULT 0,
    ease_factor    FLOAT NOT NULL DEFAULT 2.5,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_cards_entry ON cards(entry_id);
CREATE INDEX ix_cards_user_due ON cards(user_id, status, next_review_at) WHERE status != 'MASTERED';
CREATE INDEX ix_cards_user ON cards(user_id);

CREATE TABLE review_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    card_id     UUID NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    grade       review_grade NOT NULL,
    prev_state  JSONB,
    duration_ms INT,
    reviewed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_review_logs_card ON review_logs(card_id, reviewed_at DESC);

CREATE TABLE study_sessions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at   TIMESTAMPTZ,
    cards_studied INT NOT NULL DEFAULT 0,
    abandoned_at  TIMESTAMPTZ
);
CREATE INDEX ix_study_sessions_user ON study_sessions(user_id, started_at DESC);

-- +goose Down
DROP TABLE IF EXISTS study_sessions;
DROP TABLE IF EXISTS review_logs;
DROP TABLE IF EXISTS cards;
