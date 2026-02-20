-- ---------------------------------------------------------------------------
-- ref_entry_source_coverage
-- ---------------------------------------------------------------------------

-- name: GetCoverageByEntryID :many
SELECT ref_entry_id, source_slug, status, dataset_version, fetched_at
FROM ref_entry_source_coverage
WHERE ref_entry_id = $1
ORDER BY source_slug;

-- name: GetCoverageByEntryIDs :many
SELECT ref_entry_id, source_slug, status, dataset_version, fetched_at
FROM ref_entry_source_coverage
WHERE ref_entry_id = ANY(@entry_ids::uuid[])
ORDER BY ref_entry_id, source_slug;
