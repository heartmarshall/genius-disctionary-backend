-- +goose Up
CREATE TABLE ref_entries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    text            TEXT NOT NULL,
    text_normalized TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_ref_entries_text_norm ON ref_entries(text_normalized);

CREATE TABLE ref_senses (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ref_entry_id  UUID NOT NULL REFERENCES ref_entries(id) ON DELETE CASCADE,
    definition    TEXT,
    part_of_speech part_of_speech,
    cefr_level    TEXT,
    source_slug   TEXT NOT NULL,
    position      INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_ref_senses_entry ON ref_senses(ref_entry_id);

CREATE TABLE ref_translations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ref_sense_id  UUID NOT NULL REFERENCES ref_senses(id) ON DELETE CASCADE,
    text          TEXT NOT NULL,
    source_slug   TEXT NOT NULL,
    position      INT NOT NULL DEFAULT 0
);
CREATE INDEX ix_ref_translations_sense ON ref_translations(ref_sense_id);

CREATE TABLE ref_examples (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ref_sense_id  UUID NOT NULL REFERENCES ref_senses(id) ON DELETE CASCADE,
    sentence      TEXT NOT NULL,
    translation   TEXT,
    source_slug   TEXT NOT NULL,
    position      INT NOT NULL DEFAULT 0
);
CREATE INDEX ix_ref_examples_sense ON ref_examples(ref_sense_id);

CREATE TABLE ref_pronunciations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ref_entry_id  UUID NOT NULL REFERENCES ref_entries(id) ON DELETE CASCADE,
    transcription TEXT NOT NULL,
    audio_url     TEXT,
    region        TEXT,
    source_slug   TEXT NOT NULL
);
CREATE INDEX ix_ref_pronunciations_entry ON ref_pronunciations(ref_entry_id);

CREATE TABLE ref_images (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ref_entry_id  UUID NOT NULL REFERENCES ref_entries(id) ON DELETE CASCADE,
    url           TEXT NOT NULL,
    caption       TEXT,
    source_slug   TEXT NOT NULL
);
CREATE INDEX ix_ref_images_entry ON ref_images(ref_entry_id);

-- +goose Down
DROP TABLE IF EXISTS ref_images;
DROP TABLE IF EXISTS ref_pronunciations;
DROP TABLE IF EXISTS ref_examples;
DROP TABLE IF EXISTS ref_translations;
DROP TABLE IF EXISTS ref_senses;
DROP TABLE IF EXISTS ref_entries;
