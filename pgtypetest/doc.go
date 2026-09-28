// Package pgtypetest holds the contract tests for Virtuous's built-in
// jackc/pgx and legacy jackc/pgtype support.
//
// It is a separate Go module so that the root module never depends on pgx:
// the library recognises pgtype wrappers by package-path string only. See
// README.md in this directory.
package pgtypetest
