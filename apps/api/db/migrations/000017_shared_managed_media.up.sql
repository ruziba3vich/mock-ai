ALTER TABLE private_media ALTER COLUMN owner_user_id DROP NOT NULL;

CREATE UNIQUE INDEX private_media_shared_digest_idx
ON private_media(admin_author_id, sha256)
WHERE owner_user_id IS NULL;
