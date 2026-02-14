-- +goose Up
CREATE TABLE entries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ref_entry_id    UUID REFERENCES ref_entries(id) ON DELETE SET NULL,
    text            TEXT NOT NULL,
    text_normalized TEXT NOT NULL,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);
CREATE UNIQUE INDEX ux_entries_user_text ON entries(user_id, text_normalized) WHERE deleted_at IS NULL;
CREATE INDEX ix_entries_user_alive ON entries(user_id) WHERE deleted_at IS NULL;
CREATE INDEX ix_entries_user_ref ON entries(user_id, ref_entry_id) WHERE ref_entry_id IS NOT NULL;

CREATE TABLE senses (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id       UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    ref_sense_id   UUID REFERENCES ref_senses(id) ON DELETE SET NULL,
    definition     TEXT,
    part_of_speech part_of_speech,
    cefr_level     TEXT,
    source_slug    TEXT NOT NULL,
    position       INT NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_senses_entry ON senses(entry_id);
CREATE INDEX ix_senses_ref ON senses(ref_sense_id) WHERE ref_sense_id IS NOT NULL;

CREATE TABLE translations (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sense_id           UUID NOT NULL REFERENCES senses(id) ON DELETE CASCADE,
    ref_translation_id UUID REFERENCES ref_translations(id) ON DELETE SET NULL,
    text               TEXT,
    source_slug        TEXT NOT NULL,
    position           INT NOT NULL DEFAULT 0
);
CREATE INDEX ix_translations_sense ON translations(sense_id);

CREATE TABLE examples (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sense_id        UUID NOT NULL REFERENCES senses(id) ON DELETE CASCADE,
    ref_example_id  UUID REFERENCES ref_examples(id) ON DELETE SET NULL,
    sentence        TEXT,
    translation     TEXT,
    source_slug     TEXT NOT NULL,
    position        INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_examples_sense ON examples(sense_id);

CREATE TABLE entry_pronunciations (
    entry_id            UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    ref_pronunciation_id UUID NOT NULL REFERENCES ref_pronunciations(id) ON DELETE CASCADE,
    PRIMARY KEY (entry_id, ref_pronunciation_id)
);

CREATE TABLE entry_images (
    entry_id     UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    ref_image_id UUID NOT NULL REFERENCES ref_images(id) ON DELETE CASCADE,
    PRIMARY KEY (entry_id, ref_image_id)
);

CREATE TABLE user_images (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id   UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    url        TEXT NOT NULL,
    caption    TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_user_images_entry ON user_images(entry_id);

-- +goose Down
DROP TABLE IF EXISTS user_images;
DROP TABLE IF EXISTS entry_images;
DROP TABLE IF EXISTS entry_pronunciations;
DROP TABLE IF EXISTS examples;
DROP TABLE IF EXISTS translations;
DROP TABLE IF EXISTS senses;
DROP TABLE IF EXISTS entries;
