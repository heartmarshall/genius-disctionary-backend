-- ---------------------------------------------------------------------------
-- ref_data_sources
-- ---------------------------------------------------------------------------

-- name: GetAllDataSources :many
SELECT slug, name, description, source_type, is_active, dataset_version, created_at, updated_at
FROM ref_data_sources
WHERE is_active = true
ORDER BY slug;

-- name: GetDataSourceBySlug :one
SELECT slug, name, description, source_type, is_active, dataset_version, created_at, updated_at
FROM ref_data_sources
WHERE slug = $1;
