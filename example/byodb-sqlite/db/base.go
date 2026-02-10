package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

const defaultDSN = "file:byodb.sqlite?cache=shared&mode=rwc"

//go:embed sql/schemas/*.sql
var schemaFS embed.FS

func Open(ctx context.Context, dsn string) (*Queries, *sql.DB, error) {
	if strings.TrimSpace(dsn) == "" {
		dsn = defaultDSN
	}
	slog.Info("byodb-sqlite: opening sqlite database", "dsn", dsn)
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, nil, err
	}
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)

	if err := ensureSchema(ctx, conn); err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = ON;"); err != nil {
		_ = conn.Close()
		return nil, nil, err
	}

	return New(conn), conn, nil
}

func ensureSchema(ctx context.Context, conn *sql.DB) error {
	if err := ensureMigrationsTable(ctx, conn); err != nil {
		return err
	}
	paths, err := fs.Glob(schemaFS, "sql/schemas/*.sql")
	if err != nil {
		return fmt.Errorf("list schemas: %w", err)
	}
	if len(paths) == 0 {
		slog.Info("byodb-sqlite: migrations", "count", 0, "files", "none", "current", "none")
		return nil
	}
	sort.Strings(paths)
	applied, err := loadAppliedMigrations(ctx, conn)
	if err != nil {
		return err
	}
	if len(applied) == 0 {
		existing, err := detectExistingSchema(ctx, conn)
		if err != nil {
			return err
		}
		if existing {
			for _, path := range paths {
				if _, err := conn.ExecContext(ctx, "INSERT INTO schema_migrations (name) VALUES (?)", path); err != nil {
					return fmt.Errorf("record migration %s: %w", path, err)
				}
			}
			slog.Info("byodb-sqlite: migrations", "bootstrap", "existing schema detected")
			slog.Info("byodb-sqlite: migrations applied", "files", "none")
			slog.Info("byodb-sqlite: migrations current", "version", paths[len(paths)-1])
			return nil
		}
	}
	slog.Info("byodb-sqlite: migrations", "count", len(paths), "files", strings.Join(paths, ", "))

	appliedNow := make([]string, 0, len(paths))
	for _, path := range paths {
		if applied[path] {
			continue
		}
		slog.Info("byodb-sqlite: applying migration", "file", path)
		data, err := schemaFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read schema %s: %w", path, err)
		}
		if _, err := conn.ExecContext(ctx, string(data)); err != nil {
			return fmt.Errorf("apply schema %s: %w", path, err)
		}
		if _, err := conn.ExecContext(ctx, "INSERT INTO schema_migrations (name) VALUES (?)", path); err != nil {
			return fmt.Errorf("record migration %s: %w", path, err)
		}
		appliedNow = append(appliedNow, path)
	}
	if len(appliedNow) == 0 {
		slog.Info("byodb-sqlite: migrations applied", "files", "none")
	} else {
		slog.Info("byodb-sqlite: migrations applied", "files", strings.Join(appliedNow, ", "))
	}
	current := paths[len(paths)-1]
	slog.Info("byodb-sqlite: migrations current", "version", current)
	return nil
}

func ensureMigrationsTable(ctx context.Context, conn *sql.DB) error {
	_, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		name TEXT PRIMARY KEY,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}
	return nil
}

func loadAppliedMigrations(ctx context.Context, conn *sql.DB) (map[string]bool, error) {
	rows, err := conn.QueryContext(ctx, "SELECT name FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("load migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan migration: %w", err)
		}
		applied[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}
	return applied, nil
}

func detectExistingSchema(ctx context.Context, conn *sql.DB) (bool, error) {
	users, err := tableExists(ctx, conn, "users")
	if err != nil {
		return false, err
	}
	states, err := tableExists(ctx, conn, "states")
	if err != nil {
		return false, err
	}
	return users && states, nil
}

func tableExists(ctx context.Context, conn *sql.DB, name string) (bool, error) {
	var count int
	err := conn.QueryRowContext(
		ctx,
		"SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?",
		name,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check table %s: %w", name, err)
	}
	return count > 0, nil
}
