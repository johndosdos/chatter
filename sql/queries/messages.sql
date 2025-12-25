-- name: CreateMessage :one
INSERT INTO messages (user_id, content, created_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListMessages :many
SELECT m.*, u.username
FROM messages m
JOIN users u ON m.user_id = u.user_id
ORDER BY created_at DESC
LIMIT $1;
