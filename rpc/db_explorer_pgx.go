package rpc

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxDBExplorer struct {
	pool   *pgxpool.Pool
	config DBExplorerOptions
}

func (e *pgxDBExplorer) State(ctx context.Context) (DBExplorerState, error) {
	if e == nil || e.pool == nil {
		return DBExplorerState{}, errDBExplorerDisabled
	}
	ctx, cancel := context.WithTimeout(ctx, e.config.Timeout)
	defer cancel()

	filter := ""
	if !e.config.IncludeSystemSchemas {
		filter = "WHERE t.table_schema NOT IN ('pg_catalog', 'information_schema')"
	}
	query := `
SELECT
	t.table_schema,
	t.table_name,
	t.table_type
FROM information_schema.tables t
` + filter + `
ORDER BY t.table_schema, t.table_name`

	rows, err := e.pool.Query(ctx, query)
	if err != nil {
		return DBExplorerState{}, err
	}
	defer rows.Close()

	grouped := make(map[string][]DBTable, 8)
	for rows.Next() {
		var schema string
		var name string
		var tableType string
		if err := rows.Scan(&schema, &name, &tableType); err != nil {
			return DBExplorerState{}, err
		}
		grouped[schema] = append(grouped[schema], DBTable{
			Schema: schema,
			Name:   name,
			Type:   strings.ToUpper(strings.TrimSpace(tableType)),
		})
	}
	if err := rows.Err(); err != nil {
		return DBExplorerState{}, err
	}

	schemaNames := make([]string, 0, len(grouped))
	for schemaName := range grouped {
		schemaNames = append(schemaNames, schemaName)
	}
	sort.Strings(schemaNames)

	schemas := make([]DBSchema, 0, len(schemaNames))
	for _, schemaName := range schemaNames {
		tables := grouped[schemaName]
		sort.Slice(tables, func(i, j int) bool {
			return tables[i].Name < tables[j].Name
		})
		schemas = append(schemas, DBSchema{
			Name:       schemaName,
			TableCount: len(tables),
			Tables:     tables,
		})
	}

	return DBExplorerState{
		Dialect:     "postgres",
		TimeoutMS:   e.config.Timeout.Milliseconds(),
		MaxRows:     e.config.MaxRows,
		PreviewRows: e.config.PreviewRows,
		Source: DBSourceInfo{
			Connection: "pgxpool",
			ReadOnly:   true,
		},
		Schemas: schemas,
	}, nil
}

func (e *pgxDBExplorer) PreviewTable(ctx context.Context, in DBPreviewInput) (DBQueryResult, error) {
	if e == nil || e.pool == nil {
		return DBQueryResult{}, errDBExplorerDisabled
	}
	schema := strings.TrimSpace(in.Schema)
	table := strings.TrimSpace(in.Table)
	if schema == "" || table == "" {
		return DBQueryResult{}, errors.New("schema and table are required")
	}
	limit := normalizedPreviewLimit(in.Limit, e.config.MaxRows, e.config.PreviewRows)
	query := fmt.Sprintf("SELECT * FROM %s.%s LIMIT %d", quoteIdentifier(schema), quoteIdentifier(table), limit)
	return e.run(ctx, query)
}

func (e *pgxDBExplorer) RunQuery(ctx context.Context, in DBRunQueryInput) (DBQueryResult, error) {
	if e == nil || e.pool == nil {
		return DBQueryResult{}, errDBExplorerDisabled
	}
	query, err := normalizeReadOnlyQuery(in.Query, limitWithDefault(in.Limit, e.config.DefaultQueryRows), e.config.MaxRows)
	if err != nil {
		return DBQueryResult{}, err
	}
	return e.run(ctx, query)
}

func (e *pgxDBExplorer) run(ctx context.Context, query string) (DBQueryResult, error) {
	ctx, cancel := context.WithTimeout(ctx, e.config.Timeout)
	defer cancel()

	started := time.Now()
	rows, err := e.pool.Query(ctx, query)
	if err != nil {
		return DBQueryResult{}, err
	}
	defer rows.Close()

	columns, values, err := collectPGXRows(rows, e.config.MaxRows)
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

func collectPGXRows(rows pgx.Rows, maxRows int) ([]DBQueryColumn, [][]any, error) {
	fields := rows.FieldDescriptions()
	columns := make([]DBQueryColumn, len(fields))
	for idx, field := range fields {
		columns[idx] = DBQueryColumn{
			Name:         field.Name,
			DatabaseType: fmt.Sprintf("oid:%d", field.DataTypeOID),
		}
	}

	if maxRows <= 0 {
		maxRows = defaultDBExplorerMaxRows
	}
	result := make([][]any, 0, min(maxRows, 64))
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, nil, err
		}
		row := make([]any, len(values))
		for idx, value := range values {
			row[idx] = normalizeSQLValue(value)
		}
		result = append(result, row)
		if len(result) >= maxRows {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return columns, result, nil
}
