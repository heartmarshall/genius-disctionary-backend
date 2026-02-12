-- +goose Up

CREATE TYPE learning_status AS ENUM (
'NEW',
'LEARNING',
'REVIEW',
'MASTERED'
);


CREATE TYPE part_of_speech AS ENUM (
'NOUN',
'VERB',
'ADJECTIVE',
'ADVERB',
'PRONOUN',
'PREPOSITION',
'CONJUNCTION',
'INTERJECTION',
'PHRASE',
'IDIOM',
'OTHER'
);


CREATE TYPE entity_type AS ENUM (
'ENTRY',
'SENSE',
'EXAMPLE',
'IMAGE',
'PRONUNCIATION',
'CARD'
);


CREATE TYPE audit_action AS ENUM (
'CREATE',
'UPDATE',
'DELETE'
);


CREATE TYPE review_grade AS ENUM (
'AGAIN',
'HARD',
'GOOD',
'EASY'
);


-- ============================================================================
-- DICTIONARY ENTRIES
-- ============================================================================
CREATE TABLE dictionary_entries (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),


text TEXT NOT NULL,
text_normalized TEXT NOT NULL,


created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);


CREATE UNIQUE INDEX ux_dictionary_entries_text_norm
ON dictionary_entries (text_normalized);


-- ============================================================================
-- SENSES
-- ============================================================================
CREATE TABLE senses (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
entry_id UUID NOT NULL REFERENCES dictionary_entries(id) ON DELETE CASCADE,


definition TEXT,
part_of_speech part_of_speech,
source_slug TEXT NOT NULL,
cefr_level TEXT,


created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);


CREATE INDEX ix_senses_entry_id ON senses(entry_id);
CREATE INDEX ix_senses_pos ON senses(part_of_speech);


-- ============================================================================
-- TRANSLATIONS
-- ============================================================================
CREATE TABLE translations (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
sense_id UUID NOT NULL REFERENCES senses(id) ON DELETE CASCADE,


text TEXT NOT NULL,
source_slug TEXT NOT NULL
);


CREATE INDEX ix_translations_sense_id ON translations(sense_id);


-- ============================================================================
-- EXAMPLES
-- ============================================================================
CREATE TABLE examples (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
sense_id UUID NOT NULL REFERENCES senses(id) ON DELETE CASCADE,


sentence TEXT NOT NULL,
translation TEXT,
source_slug TEXT NOT NULL,


created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);


CREATE INDEX ix_examples_sense_id ON examples(sense_id);


-- ============================================================================
-- IMAGES
-- ============================================================================
CREATE TABLE images (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
entry_id UUID NOT NULL REFERENCES dictionary_entries(id) ON DELETE CASCADE,


url TEXT NOT NULL,
caption TEXT,
source_slug TEXT NOT NULL
);


CREATE INDEX ix_images_entry_id ON images(entry_id);

-- ============================================================================
-- PRONUNCIATIONS
-- ============================================================================
CREATE TABLE pronunciations (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
entry_id UUID NOT NULL REFERENCES dictionary_entries(id) ON DELETE CASCADE,


audio_url TEXT NOT NULL,
transcription TEXT,
region TEXT,
source_slug TEXT NOT NULL
);


CREATE INDEX ix_pronunciations_entry_id ON pronunciations(entry_id);


-- ============================================================================
-- CARDS
-- ============================================================================
CREATE TABLE cards (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
entry_id UUID NOT NULL UNIQUE REFERENCES dictionary_entries(id) ON DELETE CASCADE,


status learning_status NOT NULL,
next_review_at TIMESTAMPTZ,
interval_days INTEGER NOT NULL DEFAULT 0,
ease_factor REAL NOT NULL DEFAULT 2.5,


created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);


CREATE INDEX ix_cards_next_review ON cards(next_review_at);
CREATE INDEX ix_cards_status_next_review ON cards(status, next_review_at);


-- ============================================================================
-- REVIEW LOGS
-- ============================================================================
CREATE TABLE review_logs (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
card_id UUID NOT NULL REFERENCES cards(id) ON DELETE CASCADE,


grade review_grade NOT NULL,
duration_ms INTEGER,


reviewed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);


CREATE INDEX ix_review_logs_card_id ON review_logs(card_id);
CREATE INDEX ix_review_logs_reviewed_at ON review_logs(reviewed_at);


-- ============================================================================
-- HINTS
-- ============================================================================
CREATE TABLE hints (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
card_id UUID NOT NULL REFERENCES cards(id) ON DELETE CASCADE,


text TEXT NOT NULL,


created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);


CREATE INDEX ix_hints_card_id ON hints(card_id);


-- ============================================================================
-- INBOX ITEMS
-- ============================================================================
CREATE TABLE inbox_items (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),


text TEXT NOT NULL,
context TEXT,


created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);


-- ============================================================================
-- AUDIT RECORDS
-- ============================================================================
CREATE TABLE audit_records (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),


entity_type entity_type NOT NULL,
entity_id UUID,


action audit_action NOT NULL,
changes JSONB NOT NULL,


created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);


CREATE INDEX ix_audit_records_entity ON audit_records(entity_type, entity_id);
CREATE INDEX ix_audit_records_created_at ON audit_records(created_at);

-- ============================================================================
-- TRIGGERS
-- ============================================================================
CREATE OR REPLACE FUNCTION touch_updated_at() RETURNS trigger LANGUAGE plpgsql AS 'BEGIN NEW.updated_at = now(); RETURN NEW; END;';

CREATE TRIGGER trg_dictionary_entries_updated
BEFORE UPDATE ON dictionary_entries
FOR EACH ROW EXECUTE FUNCTION touch_updated_at();


CREATE TRIGGER trg_cards_updated
BEFORE UPDATE ON cards
FOR EACH ROW EXECUTE FUNCTION touch_updated_at();


CREATE TRIGGER trg_hints_updated
BEFORE UPDATE ON hints
FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- +goose Down
-- ============================================================================
-- ROLLBACK
-- ============================================================================
DROP TRIGGER IF EXISTS trg_dictionary_entries_updated ON dictionary_entries;
DROP TRIGGER IF EXISTS trg_cards_updated ON cards;
DROP TRIGGER IF EXISTS trg_hints_updated ON hints;

DROP FUNCTION IF EXISTS touch_updated_at();

DROP TABLE IF EXISTS audit_records;
DROP TABLE IF EXISTS inbox_items;
DROP TABLE IF EXISTS hints;
DROP TABLE IF EXISTS review_logs;
DROP TABLE IF EXISTS cards; 
