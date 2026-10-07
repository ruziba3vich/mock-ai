-- name: ListManagedContent :many
SELECT * FROM managed_content
WHERE (sqlc.arg(kind)::text = '' OR kind = sqlc.arg(kind))
ORDER BY kind, sort_order, updated_at DESC;

-- name: ListPublishedContent :many
SELECT * FROM managed_content
WHERE status='published' AND (sqlc.arg(kind)::text = '' OR kind=sqlc.arg(kind))
  AND (starts_at IS NULL OR starts_at<=now()) AND (ends_at IS NULL OR ends_at>now())
ORDER BY kind, sort_order, created_at;

-- name: CreateManagedContent :one
INSERT INTO managed_content(kind,slug,title,summary,status,sort_order,starts_at,ends_at,data,created_by,updated_by)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$10) RETURNING *;

-- name: UpdateManagedContent :one
UPDATE managed_content SET kind=$2,slug=$3,title=$4,summary=$5,status=$6,sort_order=$7,
 starts_at=$8,ends_at=$9,data=$10,updated_by=$11,updated_at=now()
WHERE id=$1 RETURNING *;

-- name: DeleteManagedContent :execrows
DELETE FROM managed_content WHERE id=$1;

-- name: RandomSpeakingPlan :one
SELECT title,data FROM managed_content
WHERE kind='speaking_plan' AND status='published'
 AND (starts_at IS NULL OR starts_at<=now()) AND (ends_at IS NULL OR ends_at>now())
ORDER BY random() LIMIT 1;
