-- +goose Up
ALTER TABLE ref_senses ADD COLUMN notes TEXT;

-- +goose Down
ALTER TABLE ref_senses DROP COLUMN notes;
