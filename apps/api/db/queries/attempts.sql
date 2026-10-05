-- name: CreateAttempt :one
INSERT INTO attempts (
    user_id, exam_version_id, deadline_at, question_count
)
SELECT
    $1,
    ev.id,
    now() + make_interval(secs => ev.duration_seconds),
    count(q.id)::integer
FROM exam_versions ev
JOIN exam_sections es ON es.exam_version_id = ev.id
JOIN questions q ON q.section_id = es.id
WHERE ev.id = $2 AND ev.status = 'published'
GROUP BY ev.id
RETURNING *;

-- name: SaveAttemptAnswer :one
INSERT INTO attempt_answers (attempt_id, question_id, answer)
SELECT sqlc.arg(attempt_id), sqlc.arg(question_id), sqlc.arg(answer)
WHERE EXISTS (
    SELECT 1
    FROM attempts a
    JOIN questions q ON q.id = sqlc.arg(question_id)
    JOIN exam_sections es ON es.id = q.section_id
    WHERE a.id = sqlc.arg(attempt_id)
      AND a.user_id = sqlc.arg(user_id)
      AND a.status = 'in_progress'
      AND now() < a.deadline_at
      AND es.exam_version_id = a.exam_version_id
)
ON CONFLICT (attempt_id, question_id) DO UPDATE
SET answer = EXCLUDED.answer,
    revision = attempt_answers.revision + 1,
    saved_at = now()
RETURNING *;

-- name: GetActiveAttempt :one
SELECT * FROM attempts
WHERE user_id = $1
  AND exam_version_id = $2
  AND status = 'in_progress';

-- name: ExpireStaleAttemptsForExam :exec
UPDATE attempts
SET status = 'expired', submitted_at = deadline_at
WHERE user_id = $1
  AND exam_version_id = $2
  AND status = 'in_progress'
  AND deadline_at <= now();

-- name: GetAttempt :one
SELECT * FROM attempts WHERE id = $1 AND user_id = $2;

-- name: GetAttemptForUpdate :one
SELECT * FROM attempts
WHERE id = $1 AND user_id = $2
FOR UPDATE;

-- name: ListAttemptAnswers :many
SELECT attempt_id, question_id, answer, is_correct, awarded_points, revision, saved_at
FROM attempt_answers
WHERE attempt_id = $1
ORDER BY saved_at;

-- name: ListAttemptScoringRows :many
SELECT
    q.id AS question_id,
    q.type AS question_type,
    q.points,
    aa.value AS accepted_value,
    aa.is_case_sensitive,
    ans.answer
FROM attempts a
JOIN exam_sections es ON es.exam_version_id = a.exam_version_id
JOIN questions q ON q.section_id = es.id
LEFT JOIN accepted_answers aa ON aa.question_id = q.id
LEFT JOIN attempt_answers ans ON ans.attempt_id = a.id AND ans.question_id = q.id
WHERE a.id = $1
ORDER BY es.position, q.position, aa.position;

-- name: SetAttemptAnswerScore :exec
UPDATE attempt_answers
SET is_correct = $3,
    awarded_points = $4
WHERE attempt_id = $1 AND question_id = $2;

-- name: GetBandForRawScore :one
SELECT band
FROM band_conversions
WHERE exam_version_id = $1
  AND minimum_raw_score <= $2
ORDER BY minimum_raw_score DESC
LIMIT 1;

-- name: CompleteAttempt :one
UPDATE attempts
SET status = 'scored',
    submitted_at = now(),
    scored_at = now(),
    raw_score = $3,
    band_score = $4
WHERE id = $1 AND user_id = $2 AND status = 'in_progress'
RETURNING *;

-- name: ListStudentAttempts :many
SELECT
    a.*,
    ev.title,
    ev.skill
FROM attempts a
JOIN exam_versions ev ON ev.id = a.exam_version_id
WHERE a.user_id = $1
ORDER BY a.started_at DESC
LIMIT $2 OFFSET $3;

-- name: GetStudentProgress :many
SELECT
    ev.skill,
    count(*)::integer AS completed_attempts,
    round(avg(a.band_score), 2) AS average_band,
    coalesce(max(a.band_score), 0)::numeric AS best_band,
    coalesce((array_agg(a.band_score ORDER BY a.scored_at DESC))[1], 0)::numeric AS latest_band,
    coalesce(sum(a.raw_score), 0)::numeric AS correct_answers,
    sum(a.question_count)::integer AS total_questions
FROM attempts a
JOIN exam_versions ev ON ev.id = a.exam_version_id
WHERE a.user_id = $1 AND a.status = 'scored'
GROUP BY ev.skill
ORDER BY ev.skill;
