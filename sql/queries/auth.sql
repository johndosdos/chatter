-- name: CreatePassword :one
INSERT INTO passwords (user_id, hashed_password, created_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, user_id, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserFromRefreshTok :one
SELECT * FROM refresh_tokens
WHERE token = $1;

-- name: DeleteRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token = $1;