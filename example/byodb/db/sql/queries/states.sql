-- name: ListStates :many
SELECT id, code, name
FROM states
ORDER BY name;

-- name: GetStateByCode :one
SELECT id, code, name
FROM states
WHERE code = $1;

-- name: CreateState :one
INSERT INTO states (code, name)
VALUES ($1, $2)
RETURNING id, code, name;
