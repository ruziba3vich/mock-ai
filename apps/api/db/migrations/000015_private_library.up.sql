ALTER TABLE exams ADD COLUMN access_scope text NOT NULL DEFAULT 'public' CHECK(access_scope IN ('public','private'));
ALTER TABLE exams ADD COLUMN owner_user_id uuid REFERENCES users(id) ON DELETE RESTRICT;
ALTER TABLE exams ADD CONSTRAINT exams_private_owner CHECK((access_scope='private')=(owner_user_id IS NOT NULL));
CREATE INDEX exams_private_owner_idx ON exams(owner_user_id) WHERE access_scope='private';

ALTER TABLE content_drafts ADD COLUMN access_scope text NOT NULL DEFAULT 'public' CHECK(access_scope IN ('public','private'));
ALTER TABLE content_drafts ADD COLUMN owner_user_id uuid REFERENCES users(id) ON DELETE RESTRICT;
ALTER TABLE content_drafts ADD CONSTRAINT drafts_private_owner CHECK((access_scope='private')=(owner_user_id IS NOT NULL));

ALTER TABLE content_imports ADD COLUMN access_scope text NOT NULL DEFAULT 'public' CHECK(access_scope IN ('public','private'));
ALTER TABLE content_imports ADD COLUMN owner_user_id uuid REFERENCES users(id) ON DELETE RESTRICT;
ALTER TABLE content_imports ADD CONSTRAINT imports_private_owner CHECK((access_scope='private')=(owner_user_id IS NOT NULL));

CREATE TABLE private_media (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 admin_author_id uuid NOT NULL REFERENCES admins(id),
 filename text NOT NULL,
 mime_type text NOT NULL CHECK(mime_type IN ('audio/mpeg')),
 sha256 text NOT NULL CHECK(length(sha256)=64),
 data bytea NOT NULL CHECK(octet_length(data)>0 AND octet_length(data)<=26214400),
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(owner_user_id,sha256)
);
CREATE INDEX private_media_owner_idx ON private_media(owner_user_id,created_at);
