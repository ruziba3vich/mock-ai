CREATE TABLE admins (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 email text NOT NULL UNIQUE CHECK (email=lower(email)),
 display_name text NOT NULL,
 password_hash text NOT NULL,
 disabled boolean NOT NULL DEFAULT false,
 login_window timestamptz NOT NULL DEFAULT now(),
 login_attempts integer NOT NULL DEFAULT 0,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE admin_sessions (
 token_hash text PRIMARY KEY,
 admin_id uuid NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
 expires_at timestamptz NOT NULL
);
CREATE INDEX admin_sessions_admin_idx ON admin_sessions(admin_id);
-- Preserve historical student/system authors, but new content uses admins only.
ALTER TABLE content_drafts ALTER COLUMN author_id DROP NOT NULL;
ALTER TABLE content_drafts ADD COLUMN admin_author_id uuid REFERENCES admins(id);
ALTER TABLE exams ALTER COLUMN created_by DROP NOT NULL;
ALTER TABLE exams ADD COLUMN created_by_admin uuid REFERENCES admins(id);
