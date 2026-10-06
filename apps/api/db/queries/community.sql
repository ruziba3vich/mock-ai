-- name: SaveCommunityProfile :exec
INSERT INTO community_profiles(user_id, nickname, enabled) VALUES ($1,$2,$3)
ON CONFLICT(user_id) DO UPDATE SET nickname = EXCLUDED.nickname, enabled = EXCLUDED.enabled;

-- name: GetCommunityProfile :one
SELECT coalesce(p.nickname,'')::text AS nickname, coalesce(p.enabled,false)::boolean AS enabled
FROM users u LEFT JOIN community_profiles p ON p.user_id=u.id WHERE u.id=$1;

-- name: GetCommunityStats :one
SELECT
 count(DISTINCT activity_key) FILTER (WHERE points=1)::integer AS questions,
 count(DISTINCT activity_key) FILTER (WHERE points=10)::integer AS mocks,
 count(DISTINCT (occurred_at AT TIME ZONE 'UTC')::date)::integer AS days,
 count(DISTINCT activity_key) FILTER (WHERE points=1 AND occurred_at >= date_trunc('day',now() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC')::integer AS today_questions,
 count(DISTINCT activity_key) FILTER (WHERE points=10 AND occurred_at >= date_trunc('week',now() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC')::integer AS week_mocks,
 count(DISTINCT (occurred_at AT TIME ZONE 'UTC')::date) FILTER (WHERE occurred_at >= date_trunc('week',now() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC')::integer AS week_days,
 (count(DISTINCT activity_key) FILTER (WHERE points=1 AND occurred_at >= date_trunc('week',now() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC') + 10*count(DISTINCT activity_key) FILTER (WHERE points=10 AND occurred_at >= date_trunc('week',now() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC'))::integer AS week_points
FROM community_activity WHERE user_id=$1;

-- name: GetCommunityBoard :many
SELECT r.nickname, r.points, r.position, (r.user_id = sqlc.arg(student_id))::boolean AS is_me
FROM community_weekly_ranks r WHERE r.ordinal<=50 OR r.user_id=sqlc.arg(student_id)
ORDER BY r.position,r.nickname,r.user_id;
