DELETE FROM private_media WHERE owner_user_id IS NULL;
DROP INDEX IF EXISTS private_media_shared_digest_idx;
ALTER TABLE private_media ALTER COLUMN owner_user_id SET NOT NULL;
