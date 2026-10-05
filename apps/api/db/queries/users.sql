-- name: CreateUser :one
INSERT INTO users (display_name, avatar_url, email)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByIdentity :one
SELECT u.*
FROM users u
JOIN user_identities i ON i.user_id = u.id
WHERE i.provider = $1
  AND i.provider_subject = $2
  AND u.deleted_at IS NULL;

-- name: CreateIdentity :one
INSERT INTO user_identities (
    user_id, provider, provider_subject, provider_email, profile
)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

