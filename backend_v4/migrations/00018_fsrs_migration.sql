-- +goose Up

-- Create new card_state enum for FSRS-5
CREATE TYPE card_state AS ENUM ('NEW', 'LEARNING', 'REVIEW', 'RELEARNING');

-- Add FSRS-5 columns to cards table
ALTER TABLE cards
  ADD COLUMN state card_state NOT NULL DEFAULT 'NEW',
  ADD COLUMN step INT NOT NULL DEFAULT 0,
  ADD COLUMN stability FLOAT NOT NULL DEFAULT 0,
  ADD COLUMN difficulty FLOAT NOT NULL DEFAULT 0,
  ADD COLUMN due TIMESTAMPTZ NOT NULL DEFAULT now(),
  ADD COLUMN last_review TIMESTAMPTZ,
  ADD COLUMN reps INT NOT NULL DEFAULT 0,
  ADD COLUMN lapses INT NOT NULL DEFAULT 0,
  ADD COLUMN scheduled_days INT NOT NULL DEFAULT 0,
  ADD COLUMN elapsed_days INT NOT NULL DEFAULT 0;

-- Reset all cards to NEW (no real users, clean replacement)
UPDATE cards SET state = 'NEW', step = 0, stability = 0, difficulty = 0;

-- Drop SM-2 columns
ALTER TABLE cards
  DROP COLUMN status,
  DROP COLUMN learning_step,
  DROP COLUMN next_review_at,
  DROP COLUMN interval_days,
  DROP COLUMN ease_factor;

-- Drop old enum type
DROP TYPE IF EXISTS learning_status;

-- Replace index for new FSRS state/due columns
DROP INDEX IF EXISTS ix_cards_user_due;
CREATE INDEX ix_cards_user_due ON cards(user_id, state, due);

-- Add desired_retention to user_settings
ALTER TABLE user_settings
  ADD COLUMN desired_retention FLOAT NOT NULL DEFAULT 0.9;


-- +goose Down

-- Remove desired_retention
ALTER TABLE user_settings
  DROP COLUMN IF EXISTS desired_retention;

-- Drop new index
DROP INDEX IF EXISTS ix_cards_user_due;

-- Re-create SM-2 enum
CREATE TYPE learning_status AS ENUM ('NEW', 'LEARNING', 'REVIEW', 'MASTERED');

-- Re-add SM-2 columns
ALTER TABLE cards
  ADD COLUMN status learning_status NOT NULL DEFAULT 'NEW',
  ADD COLUMN learning_step INT NOT NULL DEFAULT 0,
  ADD COLUMN next_review_at TIMESTAMPTZ,
  ADD COLUMN interval_days INT NOT NULL DEFAULT 0,
  ADD COLUMN ease_factor FLOAT NOT NULL DEFAULT 2.5;

-- Drop FSRS-5 columns
ALTER TABLE cards
  DROP COLUMN state,
  DROP COLUMN step,
  DROP COLUMN stability,
  DROP COLUMN difficulty,
  DROP COLUMN due,
  DROP COLUMN last_review,
  DROP COLUMN reps,
  DROP COLUMN lapses,
  DROP COLUMN scheduled_days,
  DROP COLUMN elapsed_days;

-- Drop FSRS enum
DROP TYPE IF EXISTS card_state;

-- Re-create old index (partial: exclude MASTERED cards)
CREATE INDEX ix_cards_user_due ON cards(user_id, status, next_review_at) WHERE status != 'MASTERED';
