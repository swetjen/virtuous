package rpc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultDBExplorerTimeout     = 5 * time.Second
	defaultDBExplorerMaxRows     = 1000
	defaultDBExplorerPreviewRows = 10
	defaultDBExplorerQueryRows   = 100
)

var (
	errDBExplorerDisabled = errors.New("rpc: db explorer is not configured")
	multiStatementPattern = regexp.MustCompile(`;+\s*\S`)
)

// DBExplorer exposes read-only schema discovery and query execution for admin UI use.
type DBExplorer interface {
	State(context.Context) (DBExplorerState, error)
	PreviewTable(context.Context, DBPreviewInput) (DBQueryResult, error)
	RunQuery(context.Context, DBRunQueryInput) (DBQueryResult, error)
}

// DBExplorerState is the payload rendered by the admin database explorer.
type DBExplorerState struct {
	Dialect     string       `json:"dialect"`
	TimeoutMS   int64        `json:"timeoutMs"`
	MaxRows     int          `json:"maxRows"`
	PreviewRows int          `json:"previewRows"`
	Schemas     []DBSchema   `json:"schemas"`
	Source      DBSourceInfo `json:"source"`
}

// DBSourceInfo describes how the explorer is attached.
type DBSourceInfo struct {
	Connection string `json:"connection"`
	ReadOnly   bool   `json:"readOnly"`
}

// DBSchema describes one schema and its tables.
type DBSchema struct {
	Name       string    `json:"name"`
	TableCount int       `json:"tableCount"`
	Tables     []DBTable `json:"tables"`
}

// DBTable describes one table or view in the explorer tree.
type DBTable struct {
	Schema      string `json:"schema"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	RowEstimate *int64 `json:"rowEstimate,omitempty"`
}

// DBPreviewInput identifies a table preview request.
type DBPreviewInput struct {
	Schema string `json:"schema"`
	Table  string `json:"table"`
	Limit  int    `json:"limit,omitempty"`
}

// DBRunQueryInput contains one ad hoc read-only query.
type DBRunQueryInput struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

// DBQueryResult is the tabular query response for previews and ad hoc queries.
type DBQueryResult struct {
	Query       string          `json:"query"`
	ExecutionMS int64           `json:"executionMs"`
	RowCount    int             `json:"rowCount"`
	Columns     []DBQueryColumn `json:"columns"`
	Rows        [][]any         `json:"rows"`
	Error       string          `json:"error,omitempty"`
}

// DBQueryColumn describes one column in a result set.
type DBQueryColumn struct {
	Name         string `json:"name"`
	DatabaseType string `json:"databaseType,omitempty"`
}

// DBExplorerOptions configures a DB explorer backend.
type DBExplorerOptions struct {
	Timeout              time.Duration
	MaxRows              int
	PreviewRows          int
	DefaultQueryRows     int
	IncludeSystemSchemas bool
}

// DBExplorerOption mutates DBExplorerOptions.
type DBExplorerOption func(*DBExplorerOptions)

// WithDBExplorer attaches a live database explorer to the router.
func WithDBExplorer(explorer DBExplorer) RouterOption {
	return func(o *RouterOptions) {
		o.DBExplorer = explorer
	}
}

// WithDBExplorerTimeout overrides the query timeout.
func WithDBExplorerTimeout(timeout time.Duration) DBExplorerOption {
	return func(o *DBExplorerOptions) {
		if timeout > 0 {
			o.Timeout = timeout
		}
	}
}

// WithDBExplorerMaxRows overrides the hard row cap for ad hoc queries.
func WithDBExplorerMaxRows(maxRows int) DBExplorerOption {
	return func(o *DBExplorerOptions) {
		if maxRows > 0 {
			o.MaxRows = maxRows
		}
	}
}

// WithDBExplorerPreviewRows overrides the default preview table row count.
func WithDBExplorerPreviewRows(previewRows int) DBExplorerOption {
	return func(o *DBExplorerOptions) {
		if previewRows > 0 {
			o.PreviewRows = previewRows
		}
	}
}

// WithDBExplorerSystemSchemas includes engine-owned schemas in discovery results.
func WithDBExplorerSystemSchemas(enabled bool) DBExplorerOption {
	return func(o *DBExplorerOptions) {
		o.IncludeSystemSchemas = enabled
	}
}

// NewSQLDBExplorer returns a database/sql-backed read-only explorer.
func NewSQLDBExplorer(db *sql.DB, opts ...DBExplorerOption) DBExplorer {
	if db == nil {
		return nil
	}
	config := defaultDBExplorerOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&config)
		}
	}
	return &sqlDBExplorer{
		db:     db,
		config: config,
	}
}

// NewPGXDBExplorer returns a pgxpool-backed read-only explorer.
func NewPGXDBExplorer(pool *pgxpool.Pool, opts ...DBExplorerOption) DBExplorer {
	if pool == nil {
		return nil
	}
	config := defaultDBExplorerOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&config)
		}
	}
	return &pgxDBExplorer{
		pool:   pool,
		config: config,
	}
}

func defaultDBExplorerOptions() DBExplorerOptions {
	return DBExplorerOptions{
		Timeout:          defaultDBExplorerTimeout,
		MaxRows:          defaultDBExplorerMaxRows,
		PreviewRows:      defaultDBExplorerPreviewRows,
		DefaultQueryRows: defaultDBExplorerQueryRows,
	}
}

type dbExplorerPayload struct {
	Enabled bool             `json:"enabled"`
	Snippet string           `json:"snippet,omitempty"`
	State   *DBExplorerState `json:"state,omitempty"`
	Error   string           `json:"error,omitempty"`
}

func dbExplorerSnippet() string {
	return `router := rpc.NewRouter(
	rpc.WithPrefix("/rpc"),
	rpc.WithDBExplorer(rpc.NewSQLDBExplorer(pool)),
)

// Postgres:
// router := rpc.NewRouter(
// 	rpc.WithPrefix("/rpc"),
// 	rpc.WithDBExplorer(rpc.NewPGXDBExplorer(pool)),
// )`
}

func dbExplorerPayloadFor(router *Router, ctx context.Context) dbExplorerPayload {
	if router == nil || router.dbExplorer == nil {
		return dbExplorerPayload{
			Enabled: false,
			Snippet: dbExplorerSnippet(),
		}
	}
	state, err := router.dbExplorer.State(ctx)
	if err != nil {
		return dbExplorerPayload{
			Enabled: true,
			Error:   err.Error(),
		}
	}
	return dbExplorerPayload{
		Enabled: true,
		State:   &state,
	}
}

func normalizeReadOnlyQuery(raw string, defaultLimit, maxRows int) (string, error) {
	query := strings.TrimSpace(raw)
	query = strings.TrimSuffix(query, ";")
	query = strings.TrimSpace(query)
	if query == "" {
		return "", errors.New("query is required")
	}
	if multiStatementPattern.MatchString(query) {
		return "", errors.New("multiple statements are not allowed")
	}

	lower := strings.ToLower(query)
	if !(strings.HasPrefix(lower, "select ") || lower == "select" || strings.HasPrefix(lower, "with ")) {
		return "", errors.New("only SELECT statements are allowed")
	}
	for _, keyword := range []string{
		" insert ", " update ", " delete ", " drop ", " alter ", " create ",
		" truncate ", " grant ", " revoke ", " vacuum ", " pragma ", " attach ",
		" detach ", " copy ", " call ", " do ", " merge ",
	} {
		if strings.Contains(" "+lower+" ", keyword) {
			return "", fmt.Errorf("query contains disallowed keyword %q", strings.TrimSpace(keyword))
		}
	}

	if maxRows <= 0 {
		maxRows = defaultDBExplorerMaxRows
	}
	if defaultLimit <= 0 {
		defaultLimit = defaultDBExplorerQueryRows
	}
	if defaultLimit > maxRows {
		defaultLimit = maxRows
	}

	if !strings.Contains(lower, " limit ") {
		query = query + fmt.Sprintf(" LIMIT %d", defaultLimit)
	}
	return fmt.Sprintf("SELECT * FROM (%s) AS virtuous_admin_query LIMIT %d", query, maxRows), nil
}

func normalizedPreviewLimit(limit, maxRows, previewRows int) int {
	if maxRows <= 0 {
		maxRows = defaultDBExplorerMaxRows
	}
	if previewRows <= 0 {
		previewRows = defaultDBExplorerPreviewRows
	}
	if limit <= 0 {
		limit = previewRows
	}
	if limit > maxRows {
		limit = maxRows
	}
	return limit
}

func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(strings.TrimSpace(name), `"`, `""`) + `"`
}
