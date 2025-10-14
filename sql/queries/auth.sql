-- name: CreatePassword :one
INSERT INTO passwords (user_id, hashed_password, created_at)
VALUES ($1, $2, $3)
RETURNING *;