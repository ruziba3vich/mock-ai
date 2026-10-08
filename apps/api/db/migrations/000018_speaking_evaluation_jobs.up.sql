CREATE TABLE speaking_evaluation_jobs (
 session_id uuid NOT NULL REFERENCES speaking_sessions(id) ON DELETE CASCADE,
 part integer NOT NULL CHECK (part BETWEEN 0 AND 3),
 status text NOT NULL DEFAULT 'queued' CHECK (status IN ('queued','processing','completed','failed')),
 attempts integer NOT NULL DEFAULT 0,
 claim_id uuid NOT NULL DEFAULT gen_random_uuid(),
 lease_until timestamptz,
 available_at timestamptz NOT NULL DEFAULT now(),
 result jsonb,
 error_code text NOT NULL DEFAULT '',
 PRIMARY KEY(session_id,part)
);
CREATE INDEX speaking_evaluation_ready ON speaking_evaluation_jobs(status,available_at);
INSERT INTO speaking_evaluation_jobs(session_id,part)
SELECT DISTINCT s.id,t.part FROM speaking_sessions s JOIN speaking_turns t ON t.session_id=s.id
WHERE s.status IN ('queued','processing');
INSERT INTO speaking_evaluation_jobs(session_id,part)
SELECT id,0 FROM speaking_sessions WHERE status IN ('queued','processing');
UPDATE speaking_sessions SET lease_until=NULL WHERE status IN ('queued','processing');
