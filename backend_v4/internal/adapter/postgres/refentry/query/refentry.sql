-- ---------------------------------------------------------------------------
-- ref_entries
-- ---------------------------------------------------------------------------

-- name: GetRefEntryByID :one
SELECT id, text, text_normalized, frequency_rank, cefr_level, is_core_lexicon, created_at
FROM ref_entries
WHERE id = $1;

-- name: GetRefEntryByNormalizedText :one
SELECT id, text, text_normalized, frequency_rank, cefr_level, is_core_lexicon, created_at
FROM ref_entries
WHERE text_normalized = $1;

-- name: SearchRefEntries :many
SELECT id, text, text_normalized, frequency_rank, cefr_level, is_core_lexicon, created_at
FROM ref_entries
WHERE text_normalized % @query::text
ORDER BY similarity(text_normalized, @query::text) DESC
LIMIT @lim::int;

-- name: InsertRefEntry :one
INSERT INTO ref_entries (id, text, text_normalized, frequency_rank, cefr_level, is_core_lexicon, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, text, text_normalized, frequency_rank, cefr_level, is_core_lexicon, created_at;

-- name: UpsertRefEntry :exec
INSERT INTO ref_entries (id, text, text_normalized)
VALUES ($1, $2, $3)
ON CONFLICT (text_normalized) DO NOTHING;

-- ---------------------------------------------------------------------------
-- ref_senses
-- ---------------------------------------------------------------------------

-- name: GetRefSensesByEntryID :many
SELECT id, ref_entry_id, definition, part_of_speech, cefr_level, source_slug, position, created_at
FROM ref_senses
WHERE ref_entry_id = $1
ORDER BY position;

-- name: InsertRefSense :one
INSERT INTO ref_senses (id, ref_entry_id, definition, part_of_speech, cefr_level, source_slug, position, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, ref_entry_id, definition, part_of_speech, cefr_level, source_slug, position, created_at;

-- name: GetRefSensesByIDs :many
SELECT id, ref_entry_id, definition, part_of_speech, cefr_level, source_slug, position, created_at
FROM ref_senses
WHERE id = ANY(@ids::uuid[])
ORDER BY position;

-- ---------------------------------------------------------------------------
-- ref_translations
-- ---------------------------------------------------------------------------

-- name: GetRefTranslationsBySenseIDs :many
SELECT id, ref_sense_id, text, source_slug, position
FROM ref_translations
WHERE ref_sense_id = ANY(@sense_ids::uuid[])
ORDER BY position;

-- name: InsertRefTranslation :one
INSERT INTO ref_translations (id, ref_sense_id, text, source_slug, position)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, ref_sense_id, text, source_slug, position;

-- name: GetRefTranslationsByIDs :many
SELECT id, ref_sense_id, text, source_slug, position
FROM ref_translations
WHERE id = ANY(@ids::uuid[])
ORDER BY position;

-- ---------------------------------------------------------------------------
-- ref_examples
-- ---------------------------------------------------------------------------

-- name: GetRefExamplesBySenseIDs :many
SELECT id, ref_sense_id, sentence, translation, source_slug, position
FROM ref_examples
WHERE ref_sense_id = ANY(@sense_ids::uuid[])
ORDER BY position;

-- name: InsertRefExample :one
INSERT INTO ref_examples (id, ref_sense_id, sentence, translation, source_slug, position)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, ref_sense_id, sentence, translation, source_slug, position;

-- name: GetRefExamplesByIDs :many
SELECT id, ref_sense_id, sentence, translation, source_slug, position
FROM ref_examples
WHERE id = ANY(@ids::uuid[])
ORDER BY position;

-- ---------------------------------------------------------------------------
-- ref_pronunciations
-- ---------------------------------------------------------------------------

-- name: GetRefPronunciationsByEntryID :many
SELECT id, ref_entry_id, transcription, audio_url, region, source_slug
FROM ref_pronunciations
WHERE ref_entry_id = $1;

-- name: InsertRefPronunciation :one
INSERT INTO ref_pronunciations (id, ref_entry_id, transcription, audio_url, region, source_slug)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, ref_entry_id, transcription, audio_url, region, source_slug;

-- name: GetRefPronunciationsByIDs :many
SELECT id, ref_entry_id, transcription, audio_url, region, source_slug
FROM ref_pronunciations
WHERE id = ANY(@ids::uuid[]);

-- ---------------------------------------------------------------------------
-- ref_images
-- ---------------------------------------------------------------------------

-- name: GetRefImagesByEntryID :many
SELECT id, ref_entry_id, url, caption, source_slug
FROM ref_images
WHERE ref_entry_id = $1;

-- name: InsertRefImage :one
INSERT INTO ref_images (id, ref_entry_id, url, caption, source_slug)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, ref_entry_id, url, caption, source_slug;

-- name: GetRefImagesByIDs :many
SELECT id, ref_entry_id, url, caption, source_slug
FROM ref_images
WHERE id = ANY(@ids::uuid[]);
