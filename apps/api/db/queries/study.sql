-- name: EnsureStudentProfile :exec
INSERT INTO student_profiles (user_id)
VALUES ($1)
ON CONFLICT (user_id) DO NOTHING;

-- name: GetStudentProfile :one
SELECT user_id, target_band, exam_date, track, daily_goal_minutes, updated_at
FROM student_profiles
WHERE user_id = $1;

-- name: UpdateStudentProfile :one
INSERT INTO student_profiles (user_id, target_band, exam_date, track, daily_goal_minutes)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id) DO UPDATE
SET target_band = EXCLUDED.target_band,
    exam_date = EXCLUDED.exam_date,
    track = EXCLUDED.track,
    daily_goal_minutes = EXCLUDED.daily_goal_minutes,
    updated_at = now()
RETURNING *;

-- name: RecordPracticeAnswer :exec
INSERT INTO practice_answers (user_id, question_id, answer, is_correct)
VALUES ($1, $2, $3, $4);

-- name: RecordAttemptReview :one
INSERT INTO study_events (user_id, event_type, attempt_id)
SELECT sqlc.arg(student_id), 'attempt_review', a.id
FROM attempts a
WHERE a.id = sqlc.arg(attempt_id)
  AND a.user_id = sqlc.arg(student_id)
  AND a.status = 'scored'
RETURNING id;

-- name: GetStudyToday :one
SELECT
    (SELECT count(*)::integer
     FROM practice_answers pa
     WHERE pa.user_id = sqlc.arg(student_id) AND pa.attempted_at >= current_date) AS practice_answers,
    (SELECT count(*)::integer
     FROM attempts a
     WHERE a.user_id = sqlc.arg(student_id) AND a.status = 'scored' AND a.scored_at >= current_date) AS mocks_completed,
    (SELECT count(*)::integer
     FROM attempts a
     WHERE a.user_id = sqlc.arg(student_id) AND a.status = 'scored' AND a.scored_at >= current_date - interval '7 days') AS mocks_this_week,
    (SELECT count(*)::integer
     FROM study_events se
     WHERE se.user_id = sqlc.arg(student_id) AND se.event_type = 'attempt_review' AND se.occurred_at >= current_date) AS reviews_today;

-- name: GetStudyActivityDays :many
SELECT activity_date::date AS activity_date
FROM (
    SELECT attempted_at::date AS activity_date
    FROM practice_answers pa
    WHERE pa.user_id = sqlc.arg(student_id) AND pa.attempted_at >= current_date - interval '60 days'
    UNION
    SELECT scored_at::date AS activity_date
    FROM attempts a
    WHERE a.user_id = sqlc.arg(student_id) AND a.status = 'scored' AND a.scored_at >= current_date - interval '60 days'
    UNION
    SELECT occurred_at::date AS activity_date
    FROM study_events se
    WHERE se.user_id = sqlc.arg(student_id) AND se.occurred_at >= current_date - interval '60 days'
) activity
GROUP BY activity_date
ORDER BY activity_date DESC;

-- name: GetWeeklyStudyActivity :many
WITH days AS (
    SELECT generate_series(current_date - interval '6 days', current_date, interval '1 day')::date AS day
), practice AS (
    SELECT attempted_at::date AS day, count(*)::integer AS answer_count
    FROM practice_answers pa
    WHERE pa.user_id = sqlc.arg(student_id) AND pa.attempted_at >= current_date - interval '6 days'
    GROUP BY attempted_at::date
), mocks AS (
    SELECT scored_at::date AS day, count(*)::integer AS mock_count
    FROM attempts a
    WHERE a.user_id = sqlc.arg(student_id) AND a.status = 'scored' AND a.scored_at >= current_date - interval '6 days'
    GROUP BY scored_at::date
)
SELECT days.day,
       coalesce(practice.answer_count, 0)::integer AS practice_answers,
       coalesce(mocks.mock_count, 0)::integer AS mocks_completed
FROM days
LEFT JOIN practice ON practice.day = days.day
LEFT JOIN mocks ON mocks.day = days.day
ORDER BY days.day;

-- name: GetQuestionTypePerformance :many
SELECT q.type,
       count(*)::integer AS attempts,
       count(*) FILTER (WHERE pa.is_correct)::integer AS correct
FROM practice_answers pa
JOIN questions q ON q.id = pa.question_id
WHERE pa.user_id = sqlc.arg(student_id)
  AND pa.attempted_at >= current_date - interval '90 days'
GROUP BY q.type
ORDER BY (count(*) FILTER (WHERE pa.is_correct))::numeric / count(*) ASC,
         count(*) DESC;
