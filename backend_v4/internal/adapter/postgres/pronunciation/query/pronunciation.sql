-- name: LinkPronunciation :exec
INSERT INTO entry_pronunciations (entry_id, ref_pronunciation_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: UnlinkPronunciation :exec
DELETE FROM entry_pronunciations
WHERE entry_id = $1 AND ref_pronunciation_id = $2;

-- name: UnlinkAllPronunciations :exec
DELETE FROM entry_pronunciations
WHERE entry_id = $1;
