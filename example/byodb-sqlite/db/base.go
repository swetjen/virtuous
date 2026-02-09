package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
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
	logVersion()
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
	paths, err := fs.Glob(schemaFS, "sql/schemas/*.sql")
	if err != nil {
		return fmt.Errorf("list schemas: %w", err)
	}
	if len(paths) == 0 {
		slog.Info("byodb-sqlite: migrations", "count", 0, "files", "none")
		return nil
	}
	sort.Strings(paths)
	slog.Info("byodb-sqlite: migrations", "count", len(paths), "files", strings.Join(paths, ", "))
	for _, path := range paths {
		slog.Info("byodb-sqlite: applying migration", "file", path)
		data, err := schemaFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read schema %s: %w", path, err)
		}
		if _, err := conn.ExecContext(ctx, string(data)); err != nil {
			return fmt.Errorf("apply schema %s: %w", path, err)
		}
	}
	return nil
}

func logVersion() {
	paths := []string{"VERSION", "../VERSION", "../../VERSION"}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		version := strings.TrimSpace(string(data))
		if version == "" {
			version = "unknown"
		}
		slog.Info("byodb-sqlite: version", "version", version, "path", path)
		return
	}
	slog.Info("byodb-sqlite: version", "version", "unknown")
}
