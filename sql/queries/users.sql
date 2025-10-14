-- name: CreateUser :one
INSERT INTO users (user_id, username, email)
VALUES ($1, $2, $3)
ON CONFLICT (user_id) DO NOTHING
RETURNING *;

-- name: GetUserWithPasswordByEmail :one
SELECT u.*, p.hashed_password
FROM users AS u
JOIN passwords AS p ON u.user_id = p.user_id
WHERE u.email = $1;