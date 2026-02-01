-- name: ListUsers :many
SELECT id, email, name, role
FROM users
ORDER BY id;

-- name: GetUser :one
SELECT id, email, name, role
FROM users
WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (email, name, role)
VALUES ($1, $2, $3)
RETURNING id, email, name, role;
