package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

const defaultDSN = "file:byodb.sqlite?cache=shared&mode=rwc"

//go:embed schema.sql
var schemaFS embed.FS

func Open(ctx context.Context, dsn string) (*Queries, *sql.DB, error) {
	if strings.TrimSpace(dsn) == "" {
		dsn = defaultDSN
	}
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

	return &Queries{db: conn}, conn, nil
}

func ensureSchema(ctx context.Context, conn *sql.DB) error {
	data, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}
	if _, err := conn.ExecContext(ctx, string(data)); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}
