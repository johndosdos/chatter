-- name: CreateMessage :one
INSERT INTO messages (user_id, content, created_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListMessages :many
SELECT t1.user_id, t1.content, t1.created_at, t2.username
FROM messages t1
JOIN users t2 ON t1.user_id = t2.user_id
ORDER BY t1.created_at;