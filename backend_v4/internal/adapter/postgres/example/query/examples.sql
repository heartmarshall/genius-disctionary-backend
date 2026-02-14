-- name: CreateExampleFromRef :one
INSERT INTO examples (id, sense_id, ref_example_id, source_slug, position, created_at)
VALUES ($1, $2, $3, $4, COALESCE((SELECT MAX(position) FROM examples WHERE sense_id = $2), -1) + 1, $5)
RETURNING id, sense_id, ref_example_id, sentence, translation, source_slug, position, created_at;

-- name: CreateExampleCustom :one
INSERT INTO examples (id, sense_id, sentence, translation, source_slug, position, created_at)
VALUES ($1, $2, $3, $4, $5, COALESCE((SELECT MAX(position) FROM examples WHERE sense_id = $2), -1) + 1, $6)
RETURNING id, sense_id, ref_example_id, sentence, translation, source_slug, position, created_at;

-- name: UpdateExample :one
UPDATE examples
SET sentence = $2, translation = $3
WHERE id = $1
RETURNING id, sense_id, ref_example_id, sentence, translation, source_slug, position, created_at;

-- name: DeleteExample :execrows
DELETE FROM examples WHERE id = $1;

-- name: CountBySense :one
SELECT count(*) FROM examples WHERE sense_id = $1;

-- name: UpdateExamplePosition :exec
UPDATE examples SET position = $2 WHERE id = $1;
