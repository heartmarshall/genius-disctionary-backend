-- ---------------------------------------------------------------------------
-- ref_word_relations
-- ---------------------------------------------------------------------------

-- name: GetRelationsByEntryID :many
SELECT id, source_entry_id, target_entry_id, relation_type, source_slug, created_at
FROM ref_word_relations
WHERE source_entry_id = $1 OR target_entry_id = $1
ORDER BY relation_type, created_at;
