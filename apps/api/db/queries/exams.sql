-- name: ListPublishedExams :many
SELECT
    e.id,
    e.slug,
    ev.id AS version_id,
    ev.version,
    ev.title,
    ev.description,
    ev.skill,
    ev.duration_seconds,
    ev.published_at,
    count(q.id)::integer AS question_count
FROM exams e
JOIN exam_versions ev ON ev.exam_id = e.id
JOIN exam_sections es ON es.exam_version_id = ev.id
JOIN questions q ON q.section_id = es.id
WHERE ev.status = 'published'
  AND e.archived_at IS NULL
GROUP BY e.id, ev.id
ORDER BY ev.published_at DESC;

-- name: GetPublishedExam :one
SELECT
    e.id,
    e.slug,
    ev.id AS version_id,
    ev.version,
    ev.title,
    ev.description,
    ev.skill,
    ev.duration_seconds,
    ev.instructions,
    ev.published_at
FROM exams e
JOIN exam_versions ev ON ev.exam_id = e.id
WHERE e.id = $1
  AND ev.status = 'published'
  AND e.archived_at IS NULL;

-- name: ListExamSections :many
SELECT id, exam_version_id, position, title, instructions, audio_url, passage
FROM exam_sections
WHERE exam_version_id = $1
ORDER BY position;

-- name: ListSectionQuestions :many
SELECT id, section_id, position, type, prompt, points, config
FROM questions
WHERE section_id = $1
ORDER BY position;

-- name: ListQuestionOptions :many
SELECT id, question_id, position, label, value
FROM question_options
WHERE question_id = $1
ORDER BY position;
