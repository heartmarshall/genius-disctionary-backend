-- +goose Up
CREATE TABLE topics (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_topics_user_name ON topics(user_id, name);

CREATE TABLE entry_topics (
    entry_id UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    topic_id UUID NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    PRIMARY KEY (entry_id, topic_id)
);
CREATE INDEX ix_entry_topics_topic ON entry_topics(topic_id);

CREATE TABLE inbox_items (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    text       TEXT NOT NULL,
    context    TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_inbox_items_user ON inbox_items(user_id, created_at DESC);

CREATE TABLE audit_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entity_type entity_type NOT NULL,
    entity_id   UUID,
    action      audit_action NOT NULL,
    changes     JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_audit_log_user ON audit_log(user_id, created_at DESC);
CREATE INDEX ix_audit_log_entity ON audit_log(entity_type, entity_id) WHERE entity_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS inbox_items;
DROP TABLE IF EXISTS entry_topics;
DROP TABLE IF EXISTS topics;
