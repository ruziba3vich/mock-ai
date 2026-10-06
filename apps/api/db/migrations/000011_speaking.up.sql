CREATE TABLE speaking_sessions (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 examiner jsonb NOT NULL,
 topic text NOT NULL,
 status text NOT NULL DEFAULT 'active' CHECK(status IN ('active','queued','processing','completed','failed','expired')),
 report jsonb,
 model text NOT NULL,
 rubric_version text NOT NULL DEFAULT 'speaking-v1',
 run_id uuid NOT NULL DEFAULT gen_random_uuid(),
 lease_until timestamptz,
 error_code text NOT NULL DEFAULT '',
 consent_at timestamptz NOT NULL DEFAULT now(),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 audio_expires_at timestamptz NOT NULL DEFAULT now()+interval '30 days'
);
CREATE UNIQUE INDEX speaking_one_open ON speaking_sessions(user_id) WHERE status IN ('active','queued','processing');
CREATE INDEX speaking_history ON speaking_sessions(user_id,created_at DESC);
CREATE TABLE speaking_turns (
 session_id uuid NOT NULL REFERENCES speaking_sessions(id) ON DELETE CASCADE,
 position integer NOT NULL CHECK(position BETWEEN 0 AND 10),
 part integer NOT NULL CHECK(part BETWEEN 1 AND 3),
 prompt text NOT NULL,
 seconds integer NOT NULL CHECK(seconds BETWEEN 20 AND 120),
 prep_seconds integer NOT NULL DEFAULT 0,
 voice_status text NOT NULL DEFAULT 'pending' CHECK(voice_status IN ('pending','queued','processing','ready','failed')),
 voice_attempts integer NOT NULL DEFAULT 0,
 voice_run_id uuid NOT NULL DEFAULT gen_random_uuid(),
 voice_lease_until timestamptz,
 examiner_audio bytea,
 recording bytea,
 duration_ms integer NOT NULL DEFAULT 0,
 answered boolean NOT NULL DEFAULT false,
 skipped boolean NOT NULL DEFAULT false,
 PRIMARY KEY(session_id,position)
);
CREATE TABLE speaking_requests (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 kind text NOT NULL CHECK(kind IN ('start','evaluate')),
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX speaking_requests_quota ON speaking_requests(user_id,created_at);
