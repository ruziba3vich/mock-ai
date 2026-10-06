CREATE TABLE ai_provider_keys (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 provider text NOT NULL CHECK(provider='gemini'),
 fingerprint text NOT NULL UNIQUE,
 ciphertext bytea NOT NULL,
 enabled boolean NOT NULL DEFAULT true,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE writing_submissions (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 parent_id uuid REFERENCES writing_submissions(id) ON DELETE SET NULL,
 prompt text NOT NULL,
 essay text NOT NULL,
 revision integer NOT NULL DEFAULT 1,
 run_id uuid NOT NULL DEFAULT gen_random_uuid(),
 status text NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','queued','processing','completed','failed')),
 report jsonb,
 model text NOT NULL DEFAULT '',
 rubric_version text NOT NULL DEFAULT '',
 error_code text NOT NULL DEFAULT '',
 lease_until timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX writing_owner_idx ON writing_submissions(user_id,updated_at DESC);
CREATE INDEX writing_queue_idx ON writing_submissions(status,created_at);
CREATE TABLE writing_evaluation_requests (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 submission_id uuid REFERENCES writing_submissions(id) ON DELETE SET NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX writing_requests_user_idx ON writing_evaluation_requests(user_id,created_at);
