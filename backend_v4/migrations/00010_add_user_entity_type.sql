-- +goose Up
ALTER TYPE entity_type ADD VALUE IF NOT EXISTS 'USER';

-- +goose Down
-- PostgreSQL does not support removing values from enums.
-- A full recreation would be needed; leaving as no-op for safety.
