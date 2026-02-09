# SQLite Queries Styleguide

- Use sqlc annotations: `-- name: QueryName :one|:many|:exec`.
- Use positional args (`?`) for SQLite placeholders.
- Keep queries small and focused.
- After schema or query changes, run `make gen` from the example root.
