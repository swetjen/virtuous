-- +goose Up
ALTER TABLE users
ADD COLUMN disabled BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE users
DROP COLUMN disabled;
