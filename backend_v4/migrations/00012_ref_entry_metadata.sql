-- +goose Up
ALTER TABLE ref_entries ADD COLUMN frequency_rank INT;
ALTER TABLE ref_entries ADD COLUMN cefr_level TEXT;
ALTER TABLE ref_entries ADD COLUMN is_core_lexicon BOOLEAN DEFAULT false;

ALTER TABLE ref_entries ADD CONSTRAINT chk_ref_entries_cefr_level
    CHECK (cefr_level IN ('A1', 'A2', 'B1', 'B2', 'C1', 'C2'));

CREATE INDEX ix_ref_entries_cefr_level ON ref_entries(cefr_level) WHERE cefr_level IS NOT NULL;
CREATE INDEX ix_ref_entries_frequency_rank ON ref_entries(frequency_rank) WHERE frequency_rank IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS ix_ref_entries_frequency_rank;
DROP INDEX IF EXISTS ix_ref_entries_cefr_level;
ALTER TABLE ref_entries DROP CONSTRAINT IF EXISTS chk_ref_entries_cefr_level;
ALTER TABLE ref_entries DROP COLUMN IF EXISTS is_core_lexicon;
ALTER TABLE ref_entries DROP COLUMN IF EXISTS cefr_level;
ALTER TABLE ref_entries DROP COLUMN IF EXISTS frequency_rank;
