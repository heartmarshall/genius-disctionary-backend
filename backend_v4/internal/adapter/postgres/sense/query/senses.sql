-- name: CreateSenseFromRef :one
INSERT INTO senses (id, entry_id, ref_sense_id, source_slug, position, created_at)
VALUES ($1, $2, $3, $4, COALESCE((SELECT MAX(position) FROM senses WHERE entry_id = $2), -1) + 1, $5)
RETURNING id, entry_id, ref_sense_id, definition, part_of_speech, cefr_level, source_slug, position, created_at;

-- name: CreateSenseCustom :one
INSERT INTO senses (id, entry_id, definition, part_of_speech, cefr_level, source_slug, position, created_at)
VALUES ($1, $2, $3, $4, $5, $6, COALESCE((SELECT MAX(position) FROM senses WHERE entry_id = $2), -1) + 1, $7)
RETURNING id, entry_id, ref_sense_id, definition, part_of_speech, cefr_level, source_slug, position, created_at;

-- name: UpdateSense :one
UPDATE senses
SET definition = $2, part_of_speech = $3, cefr_level = $4
WHERE id = $1
RETURNING id, entry_id, ref_sense_id, definition, part_of_speech, cefr_level, source_slug, position, created_at;

-- name: DeleteSense :execrows
DELETE FROM senses WHERE id = $1;

-- name: CountByEntry :one
SELECT count(*) FROM senses WHERE entry_id = $1;

-- name: UpdateSensePosition :exec
UPDATE senses SET position = $2 WHERE id = $1;
