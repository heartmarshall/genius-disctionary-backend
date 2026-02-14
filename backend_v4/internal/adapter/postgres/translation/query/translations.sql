-- name: CreateTranslationFromRef :one
INSERT INTO translations (id, sense_id, ref_translation_id, source_slug, position)
VALUES ($1, $2, $3, $4, COALESCE((SELECT MAX(position) FROM translations WHERE sense_id = $2), -1) + 1)
RETURNING id, sense_id, ref_translation_id, text, source_slug, position;

-- name: CreateTranslationCustom :one
INSERT INTO translations (id, sense_id, text, source_slug, position)
VALUES ($1, $2, $3, $4, COALESCE((SELECT MAX(position) FROM translations WHERE sense_id = $2), -1) + 1)
RETURNING id, sense_id, ref_translation_id, text, source_slug, position;

-- name: UpdateTranslation :one
UPDATE translations
SET text = $2
WHERE id = $1
RETURNING id, sense_id, ref_translation_id, text, source_slug, position;

-- name: DeleteTranslation :execrows
DELETE FROM translations WHERE id = $1;

-- name: CountBySense :one
SELECT count(*) FROM translations WHERE sense_id = $1;

-- name: UpdateTranslationPosition :exec
UPDATE translations SET position = $2 WHERE id = $1;
