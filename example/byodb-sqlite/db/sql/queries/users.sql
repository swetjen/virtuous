-- name: ListUsers :many
SELECT id, email, name, role
FROM users
ORDER BY id;

-- name: GetUser :one
SELECT id, email, name, role
FROM users
WHERE id = ?;

-- name: CreateUser :one
INSERT INTO users (email, name, role)
VALUES (?, ?, ?)
RETURNING id, email, name, role;
