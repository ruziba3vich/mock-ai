DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM speaking_sessions WHERE mode='live') THEN
        RAISE EXCEPTION 'Cannot remove live-speaking metadata while live interviews exist; preserve/export them before an explicit downgrade';
    END IF;
END $$;
ALTER TABLE speaking_turns DROP COLUMN live_transcript;
ALTER TABLE speaking_sessions DROP COLUMN live_connections, DROP COLUMN live_lease_until, DROP COLUMN live_run_id, DROP COLUMN live_model, DROP COLUMN mode;
