# SQLite Schemas Styleguide

- Each file is a migration step.
- Use `INTEGER PRIMARY KEY AUTOINCREMENT` for ID columns.
- Keep schema changes backwards-compatible when possible.
- After schema changes, run `make gen` and run tests.
