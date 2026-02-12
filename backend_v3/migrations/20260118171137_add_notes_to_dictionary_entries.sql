-- +goose Up

ALTER TABLE dictionary_entries
ADD COLUMN notes TEXT;

COMMENT ON COLUMN dictionary_entries.notes IS 'Пользовательские заметки о слове (происхождение, этимология, личные заметки и т.д.)';

-- +goose Down

ALTER TABLE dictionary_entries
DROP COLUMN notes;











