-- Refuse to discard attribution once admins have authored production content.
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM content_drafts WHERE admin_author_id IS NOT NULL)
 OR EXISTS(SELECT 1 FROM exams WHERE created_by_admin IS NOT NULL) THEN
   RAISE EXCEPTION 'Admin-authored content exists. Use a forward fix; export/migrate attribution before downgrading.';
 END IF;
END $$;
ALTER TABLE content_drafts DROP COLUMN admin_author_id;
ALTER TABLE exams DROP COLUMN created_by_admin;
ALTER TABLE content_drafts ALTER COLUMN author_id SET NOT NULL;
ALTER TABLE exams ALTER COLUMN created_by SET NOT NULL;
DROP TABLE admin_sessions;
DROP TABLE admins;
