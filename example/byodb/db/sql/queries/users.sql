-- name: ListUsers :many
SELECT id, email, name, role, disabled
FROM users
ORDER BY id;

-- name: GetUser :one
SELECT id, email, name, role, disabled
FROM users
WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (email, name, role)
VALUES ($1, $2, $3)
RETURNING id, email, name, role, disabled;

-- name: CreateUserWithPassword :one
INSERT INTO users (email, name, role, password_hash, confirmed, confirm_code)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, email, name, role, password_hash, confirmed, confirm_code, disabled, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, name, role, password_hash, confirmed, confirm_code, disabled, created_at, updated_at
FROM users
WHERE lower(email) = lower($1);

-- name: ConfirmUserByCode :one
UPDATE users
SET confirmed = TRUE,
    confirm_code = '',
    updated_at = now()
WHERE upper(confirm_code) = upper($1)
RETURNING id, email, name, role, password_hash, confirmed, confirm_code, disabled, created_at, updated_at;

-- name: UserByIDWithAuth :one
SELECT id, email, name, role, password_hash, confirmed, confirm_code, disabled, created_at, updated_at
FROM users
WHERE id = $1;

-- name: UserUpdateDisabled :one
UPDATE users
SET disabled = $2,
    updated_at = now()
WHERE id = $1
RETURNING id, email, name, role, disabled;
