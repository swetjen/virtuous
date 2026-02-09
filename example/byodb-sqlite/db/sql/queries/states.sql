-- name: ListStates :many
SELECT id, code, name
FROM states
ORDER BY name;

-- name: GetStateByCode :one
SELECT id, code, name
FROM states
WHERE code = ?;

-- name: CreateState :one
INSERT INTO states (code, name)
VALUES (?, ?)
RETURNING id, code, name;
