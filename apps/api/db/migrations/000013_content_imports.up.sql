CREATE TABLE content_imports (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_author_id uuid NOT NULL REFERENCES admins(id),
    filename text NOT NULL,
    size_bytes bigint NOT NULL CHECK(size_bytes > 0 AND size_bytes <= 52428800),
    sha256 text NOT NULL CHECK(length(sha256) = 64),
    requested_skills text NOT NULL CHECK(requested_skills IN ('all','reading','listening')),
    status text NOT NULL DEFAULT 'queued' CHECK(status IN ('queued','processing','completed','failed')),
    model text NOT NULL,
    provider_file_uri text NOT NULL,
    progress_current integer NOT NULL DEFAULT 0 CHECK(progress_current >= 0),
    progress_total integer NOT NULL DEFAULT 0 CHECK(progress_total >= 0),
    draft_ids jsonb NOT NULL DEFAULT '[]'::jsonb,
    warnings jsonb NOT NULL DEFAULT '[]'::jsonb,
    error_message text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz
);

CREATE INDEX content_imports_admin_created_idx ON content_imports(admin_author_id, created_at DESC);
CREATE INDEX content_imports_queue_idx ON content_imports(created_at) WHERE status = 'queued';
