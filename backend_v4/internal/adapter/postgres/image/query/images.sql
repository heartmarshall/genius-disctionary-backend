-- ---------------------------------------------------------------------------
-- entry_images (M2M catalog images)
-- ---------------------------------------------------------------------------

-- name: LinkCatalog :exec
INSERT INTO entry_images (entry_id, ref_image_id)
VALUES (@entry_id, @ref_image_id)
ON CONFLICT DO NOTHING;

-- name: UnlinkCatalog :exec
DELETE FROM entry_images
WHERE entry_id = @entry_id AND ref_image_id = @ref_image_id;

-- ---------------------------------------------------------------------------
-- user_images (user-uploaded)
-- ---------------------------------------------------------------------------

-- name: CreateUserImage :one
INSERT INTO user_images (id, entry_id, url, caption, created_at)
VALUES (@id, @entry_id, @url, @caption, @created_at)
RETURNING id, entry_id, url, caption, created_at;

-- name: DeleteUserImage :execrows
DELETE FROM user_images
WHERE id = @id;
