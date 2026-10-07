ALTER TABLE speaking_sessions ADD COLUMN mode text NOT NULL DEFAULT 'recorded' CHECK(mode IN ('recorded','live'));
ALTER TABLE speaking_sessions ADD COLUMN live_model text NOT NULL DEFAULT '';
ALTER TABLE speaking_sessions ADD COLUMN live_run_id uuid;
ALTER TABLE speaking_sessions ADD COLUMN live_lease_until timestamptz;
ALTER TABLE speaking_sessions ADD COLUMN live_connections integer NOT NULL DEFAULT 0;
ALTER TABLE speaking_turns ADD COLUMN live_transcript text NOT NULL DEFAULT '';
