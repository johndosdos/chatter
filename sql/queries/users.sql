-- name: CreateUser :one
INSERT INTO users (user_id, username, email)
VALUES ($1, $2, $3)
ON CONFLICT (user_id) DO NOTHING
RETURNING *;