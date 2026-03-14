package rpc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

type sqlDBExplorer struct {
	db     *sql.DB
	config DBExplorerOptions
}

func (e *sqlDBExplorer) State(ctx context.Context) (DBExplorerState, error) {
	if e == nil || e.db == nil {
		return DBExplorerState{}, errDBExplorerDisabled
	}

	ctx, cancel := context.WithTimeout(ctx, e.config.Timeout)
	defer cancel()

	rows, err := e.db.QueryContext(ctx, `
SELECT name, type
FROM sqlite_master
WHERE type IN ('table', 'view') AND name NOT LIKE 'sqlite_%'
ORDER BY name`)
	if err != nil {
		return DBExplorerState{}, err
	}
	defer rows.Close()

	tables := make([]DBTable, 0, 32)
	for rows.Next() {
		var name string
		var tableType string
		if err := rows.Scan(&name, &tableType); err != nil {
			return DBExplorerState{}, err
		}
		tables = append(tables, DBTable{
			Schema: "main",
			Name:   name,
			Type:   strings.ToUpper(strings.TrimSpace(tableType)),
		})
	}
	if err := rows.Err(); err != nil {
		return DBExplorerState{}, err
	}

	state := DBExplorerState{
		Dialect:     "sqlite",
		TimeoutMS:   e.config.Timeout.Milliseconds(),
		MaxRows:     e.config.MaxRows,
		PreviewRows: e.config.PreviewRows,
		Source: DBSourceInfo{
			Connection: "database/sql",
			ReadOnly:   true,
		},
		Schemas: []DBSchema{
			{
				Name:       "main",
				TableCount: len(tables),
				Tables:     tables,
			},
		},
	}
	return state, nil
}

func (e *sqlDBExplorer) PreviewTable(ctx context.Context, in DBPreviewInput) (DBQueryResult, error) {
	if e == nil || e.db == nil {
		return DBQueryResult{}, errDBExplorerDisabled
	}
	table := strings.TrimSpace(in.Table)
	if table == "" {
		return DBQueryResult{}, errors.New("table is required")
	}
	limit := normalizedPreviewLimit(in.Limit, e.config.MaxRows, e.config.PreviewRows)
	query := fmt.Sprintf("SELECT * FROM %s LIMIT %d", quoteIdentifier(table), limit)
	return e.run(ctx, query)
}

func (e *sqlDBExplorer) RunQuery(ctx context.Context, in DBRunQueryInput) (DBQueryResult, error) {
	if e == nil || e.db == nil {
		return DBQueryResult{}, errDBExplorerDisabled
	}
	query, err := normalizeReadOnlyQuery(in.Query, limitWithDefault(in.Limit, e.config.DefaultQueryRows), e.config.MaxRows)
	if err != nil {
		return DBQueryResult{}, err
	}
	return e.run(ctx, query)
}

func (e *sqlDBExplorer) run(ctx context.Context, query string) (DBQueryResult, error) {
	ctx, cancel := context.WithTimeout(ctx, e.config.Timeout)
	defer cancel()

	started := time.Now()
	rows, err := e.db.QueryContext(ctx, query)
	if err != nil {
		return DBQueryResult{}, err
	}
	defer rows.Close()

	columns, values, err := collectSQLRows(rows, e.config.MaxRows)
	if err != nil {
		return DBQueryResult{}, err
	}

	return DBQueryResult{
		Query:       query,
		ExecutionMS: time.Since(started).Milliseconds(),
		RowCount:    len(values),
		Columns:     columns,
		Rows:        values,
	}, nil
}

func collectSQLRows(rows *sql.Rows, maxRows int) ([]DBQueryColumn, [][]any, error) {
	names, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}
	colTypes, _ := rows.ColumnTypes()

	columns := make([]DBQueryColumn, len(names))
	for idx, name := range names {
		columns[idx] = DBQueryColumn{Name: name}
		if idx < len(colTypes) {
			columns[idx].DatabaseType = colTypes[idx].DatabaseTypeName()
		}
	}

	scanValues := make([]any, len(names))
	scanTargets := make([]any, len(names))
	for idx := range scanValues {
		scanTargets[idx] = &scanValues[idx]
	}

	if maxRows <= 0 {
		maxRows = defaultDBExplorerMaxRows
	}
	results := make([][]any, 0, min(maxRows, 64))
	for rows.Next() {
		if err := rows.Scan(scanTargets...); err != nil {
			return nil, nil, err
		}
		row := make([]any, len(names))
		for idx, value := range scanValues {
			row[idx] = normalizeSQLValue(value)
		}
		results = append(results, row)
		if len(results) >= maxRows {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return columns, results, nil
}

func normalizeSQLValue(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case []byte:
		if utf8.Valid(typed) {
			return string(typed)
		}
		return fmt.Sprintf("0x%x", typed)
	case time.Time:
		return typed.UTC().Format(time.RFC3339Nano)
	case bool, string, int64, int32, int16, int8, int,
		uint64, uint32, uint16, uint8, uint,
		float32, float64:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func limitWithDefault(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func collapseSQLiteSchemas(schemas []DBSchema) []DBSchema {
	if len(schemas) <= 1 {
		return schemas
	}
	sort.Slice(schemas, func(i, j int) bool { return schemas[i].Name < schemas[j].Name })
	return schemas
}
