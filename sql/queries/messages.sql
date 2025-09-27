-- name: CreateMessage :one
INSERT INTO messages (user_id, username, content, created_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListMessages :many
SELECT * FROM messages
ORDER BY created_at ASC
LIMIT 50;