-- name: CreateMessage :one
INSERT INTO messages (user_id, content, created_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListMessages :many
SELECT m.*, u.username
FROM messages m
JOIN users u ON m.user_id = u.user_id
WHERE (
  sqlc.narg(since)::TIMESTAMPTZ IS NULL OR m.created_at > sqlc.arg(since)::TIMESTAMPTZ
)
ORDER BY m.created_at ASC
LIMIT 50;