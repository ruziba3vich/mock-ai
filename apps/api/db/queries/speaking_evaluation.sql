-- name: QueueSpeakingEvaluation :exec
INSERT INTO speaking_evaluation_jobs(session_id,part)
SELECT t.session_id,t.part FROM speaking_turns t WHERE t.session_id=$1 GROUP BY t.session_id,t.part
UNION SELECT $1,0
ON CONFLICT(session_id,part) DO UPDATE SET
 status=CASE WHEN speaking_evaluation_jobs.status='completed' AND speaking_evaluation_jobs.part>0 THEN 'completed' ELSE 'queued' END,
 attempts=0,claim_id=gen_random_uuid(),lease_until=NULL,available_at=now(),error_code='';

-- name: ClaimSpeakingEvaluation :one
WITH candidate AS (
 SELECT j.session_id,j.part FROM speaking_evaluation_jobs j JOIN speaking_sessions s ON s.id=j.session_id
 WHERE s.status IN ('queued','processing') AND s.audio_expires_at>now()
 AND j.status='queued' AND j.available_at<=now()
 AND (j.part>0 OR NOT EXISTS (SELECT 1 FROM speaking_evaluation_jobs p WHERE p.session_id=j.session_id AND p.part>0 AND p.status<>'completed'))
 ORDER BY j.available_at,j.part DESC FOR UPDATE OF j SKIP LOCKED LIMIT 1
)
UPDATE speaking_evaluation_jobs j SET status='processing',attempts=attempts+1,claim_id=gen_random_uuid(),lease_until=now()+interval '4 minutes'
FROM candidate c WHERE j.session_id=c.session_id AND j.part=c.part RETURNING j.*;

-- name: ListSpeakingEvaluation :many
SELECT * FROM speaking_evaluation_jobs WHERE session_id=$1 ORDER BY part;

-- name: FinishSpeakingEvaluation :execrows
UPDATE speaking_evaluation_jobs SET status=$4,result=$5,error_code=$6,lease_until=NULL,available_at=now()+interval '10 seconds'
WHERE session_id=$1 AND part=$2 AND claim_id=$3 AND status='processing' AND lease_until>now();

-- name: RecoverSpeakingEvaluation :exec
UPDATE speaking_evaluation_jobs SET status=CASE WHEN attempts<3 THEN 'queued' ELSE 'failed' END,
error_code='lease_expired',lease_until=NULL,available_at=now()
WHERE status='processing' AND lease_until<now();

-- name: FailExhaustedSpeakingEvaluation :exec
UPDATE speaking_sessions s SET status='failed',error_code='evaluation_retries_exhausted',lease_until=NULL,updated_at=now()
WHERE s.status IN ('queued','processing') AND EXISTS (SELECT 1 FROM speaking_evaluation_jobs j WHERE j.session_id=s.id AND j.status='failed');

-- name: MarkSpeakingEvaluationProcessing :exec
UPDATE speaking_sessions SET status='processing',lease_until=NULL,updated_at=now() WHERE id=$1 AND status='queued';

-- name: CompleteSpeakingEvaluation :execrows
UPDATE speaking_sessions SET status='completed',report=$2,error_code='',lease_until=NULL,updated_at=now()
WHERE id=$1 AND status IN ('queued','processing');

-- name: SpeakingEvaluationOwner :one
SELECT user_id FROM speaking_sessions WHERE id=$1;
