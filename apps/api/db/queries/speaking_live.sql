-- name: SetSpeakingLive :exec
UPDATE speaking_sessions SET mode='live',live_model=$2 WHERE id=$1;

-- name: PrepareSpeakingLive :exec
UPDATE speaking_turns SET voice_status='ready' WHERE session_id=$1;

-- name: ClaimSpeakingLive :one
UPDATE speaking_sessions SET live_run_id=gen_random_uuid(),live_lease_until=now()+interval '45 seconds',live_connections=live_connections+1
WHERE id=$1 AND user_id=$2 AND mode='live' AND status='active' AND audio_expires_at>now()
AND live_connections<12 AND (live_lease_until IS NULL OR live_lease_until<now()) RETURNING *;

-- name: RenewSpeakingLive :execrows
UPDATE speaking_sessions SET live_lease_until=now()+interval '45 seconds'
WHERE id=$1 AND live_run_id=$2 AND live_lease_until>now() AND status='active';

-- name: ReleaseSpeakingLive :exec
UPDATE speaking_sessions SET live_lease_until=NULL WHERE id=$1 AND live_run_id=$2;

-- name: SaveSpeakingLive :execrows
UPDATE speaking_turns t SET recording=$4,duration_ms=$5,live_transcript=$6,answered=$7,skipped=$8
FROM speaking_sessions s WHERE t.session_id=s.id AND s.id=$1 AND s.live_run_id=$2
AND s.live_lease_until>now() AND s.status='active' AND t.position=$3 AND NOT t.answered;

-- name: SetSpeakingLiveQuestion :execrows
UPDATE speaking_turns t SET prompt=$4 FROM speaking_sessions s
WHERE t.session_id=s.id AND s.id=$1 AND s.live_run_id=$2 AND s.live_lease_until>now()
AND s.status='active' AND t.position=$3 AND NOT t.answered;
