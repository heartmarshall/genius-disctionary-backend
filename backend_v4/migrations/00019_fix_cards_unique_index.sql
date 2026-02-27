-- +goose Up
-- Fix: ux_cards_entry was global (entry_id only), preventing multi-user scenarios.
-- Change to per-user uniqueness: one card per entry per user.
DROP INDEX IF EXISTS ux_cards_entry;
CREATE UNIQUE INDEX ux_cards_entry ON cards(user_id, entry_id);

-- +goose Down
DROP INDEX IF EXISTS ux_cards_entry;
CREATE UNIQUE INDEX ux_cards_entry ON cards(entry_id);
