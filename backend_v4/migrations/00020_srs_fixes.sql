-- +goose Up

-- CHECK constraints on stability and difficulty
ALTER TABLE cards ADD CONSTRAINT chk_stability_nonneg CHECK (stability >= 0);
ALTER TABLE cards ADD CONSTRAINT chk_difficulty_range CHECK (difficulty >= 0 AND difficulty <= 10);

-- Index for NEW cards ordered by created_at
CREATE INDEX ix_cards_new_created ON cards(user_id, created_at) WHERE state = 'NEW';

-- Add user_id to review_logs for efficient user-scoped queries
ALTER TABLE review_logs ADD COLUMN user_id UUID REFERENCES users(id);

-- Backfill user_id from cards
UPDATE review_logs rl SET user_id = c.user_id FROM cards c WHERE rl.card_id = c.id;

-- Make NOT NULL after backfill
ALTER TABLE review_logs ALTER COLUMN user_id SET NOT NULL;

-- Indexes for user-scoped review_log queries
CREATE INDEX ix_review_logs_user_reviewed ON review_logs(user_id, reviewed_at);
CREATE INDEX ix_review_logs_user_card ON review_logs(user_id, card_id);

-- +goose Down
DROP INDEX IF EXISTS ix_review_logs_user_card;
DROP INDEX IF EXISTS ix_review_logs_user_reviewed;
ALTER TABLE review_logs DROP COLUMN IF EXISTS user_id;
DROP INDEX IF EXISTS ix_cards_new_created;
ALTER TABLE cards DROP CONSTRAINT IF EXISTS chk_difficulty_range;
ALTER TABLE cards DROP CONSTRAINT IF EXISTS chk_stability_nonneg;
