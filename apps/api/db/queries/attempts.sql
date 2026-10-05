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
VALUES ($1, $2, $3)
ON CONFLICT (attempt_id, question_id) DO UPDATE
SET answer = EXCLUDED.answer,
    revision = attempt_answers.revision + 1,
    saved_at = now()
RETURNING *;

-- name: GetAttempt :one
SELECT * FROM attempts WHERE id = $1 AND user_id = $2;

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

