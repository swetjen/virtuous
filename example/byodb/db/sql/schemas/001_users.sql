-- +goose Up
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    role TEXT NOT NULL,
    password_hash TEXT NOT NULL DEFAULT '',
    confirmed BOOLEAN NOT NULL DEFAULT FALSE,
    confirm_code TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX users_confirm_code_idx ON users (confirm_code)
WHERE confirm_code <> '';
