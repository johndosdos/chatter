-- name: CreateMessage :one
INSERT INTO messages (user_id, username, content)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListMessages :many
SELECT * FROM messages
ORDER BY created_at ASC
LIMIT 50;