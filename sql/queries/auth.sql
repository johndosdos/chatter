-- name: CreatePassword :one
INSERT INTO passwords (user_id, hashed_password, created_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, user_id, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token = $1 AND expires_at > NOW();

-- name: DeleteRefreshToken :exec
DELETE FROM refresh_tokens
WHERE token = $1;

-- name: UpdateRefreshToken :one
