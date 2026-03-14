package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeDBExplorer struct {
	state       DBExplorerState
	stateErr    error
	previewIn   DBPreviewInput
	queryIn     DBRunQueryInput
	previewResp DBQueryResult
	queryResp   DBQueryResult
}

func (f *fakeDBExplorer) State(_ context.Context) (DBExplorerState, error) {
	if f.stateErr != nil {
		return DBExplorerState{}, f.stateErr
	}
	return f.state, nil
}

func (f *fakeDBExplorer) PreviewTable(_ context.Context, in DBPreviewInput) (DBQueryResult, error) {
	f.previewIn = in
	return f.previewResp, nil
}

func (f *fakeDBExplorer) RunQuery(_ context.Context, in DBRunQueryInput) (DBQueryResult, error) {
	f.queryIn = in
	return f.queryResp, nil
}

func TestNormalizeReadOnlyQuery(t *testing.T) {
	query, err := normalizeReadOnlyQuery("SELECT id FROM users", 10, 100)
	if err != nil {
		t.Fatalf("normalize query: %v", err)
	}
	if !strings.Contains(strings.ToLower(query), "from (select id from users limit 10)") {
		t.Fatalf("expected normalized query to include wrapped select and limit, got %q", query)
	}
	if !strings.HasSuffix(query, "LIMIT 100") {
		t.Fatalf("expected hard cap limit suffix, got %q", query)
	}

	if _, err := normalizeReadOnlyQuery("DELETE FROM users", 10, 100); err == nil {
		t.Fatalf("expected non-select query to fail")
	}
	if _, err := normalizeReadOnlyQuery("SELECT 1; SELECT 2", 10, 100); err == nil {
		t.Fatalf("expected multi statement query to fail")
	}
}

func TestRPCServeDocsDBExplorerDisabledPayload(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(testHandler)
	router.ServeDocs()

	req := httptest.NewRequest(http.MethodGet, "/rpc/docs/_admin/db", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var payload map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	enabled, _ := payload["enabled"].(bool)
	if enabled {
		t.Fatalf("expected explorer disabled payload")
	}
	if snippet, _ := payload["snippet"].(string); strings.TrimSpace(snippet) == "" {
		t.Fatalf("expected setup snippet when explorer is disabled")
	}
}

func TestRPCServeDocsDBExplorerEndpoints(t *testing.T) {
	fake := &fakeDBExplorer{
		state: DBExplorerState{
			Dialect:     "sqlite",
			TimeoutMS:   5000,
			MaxRows:     1000,
			PreviewRows: 10,
			Schemas: []DBSchema{
				{
					Name:       "main",
					TableCount: 1,
					Tables: []DBTable{
						{Schema: "main", Name: "users", Type: "TABLE"},
					},
				},
			},
		},
		previewResp: DBQueryResult{
			Query:       `SELECT * FROM "users" LIMIT 10`,
			ExecutionMS: 2,
			RowCount:    1,
			Columns: []DBQueryColumn{
				{Name: "id", DatabaseType: "INTEGER"},
			},
			Rows: [][]any{{float64(1)}},
		},
		queryResp: DBQueryResult{
			Query:       "SELECT * FROM users LIMIT 100",
			ExecutionMS: 3,
			RowCount:    1,
			Columns: []DBQueryColumn{
				{Name: "id", DatabaseType: "INTEGER"},
			},
			Rows: [][]any{{float64(1)}},
		},
	}

	router := NewRouter(WithDBExplorer(fake))
	router.HandleRPC(testHandler)
	router.ServeDocs()

	stateReq := httptest.NewRequest(http.MethodGet, "/rpc/docs/_admin/db", nil)
	stateRec := httptest.NewRecorder()
	router.ServeHTTP(stateRec, stateReq)
	if stateRec.Code != http.StatusOK {
		t.Fatalf("expected state endpoint 200, got %d", stateRec.Code)
	}
	var statePayload struct {
		Enabled bool            `json:"enabled"`
		State   DBExplorerState `json:"state"`
	}
	if err := json.NewDecoder(stateRec.Body).Decode(&statePayload); err != nil {
		t.Fatalf("decode state payload: %v", err)
	}
	if !statePayload.Enabled {
		t.Fatalf("expected enabled=true payload")
	}
	if statePayload.State.Dialect != "sqlite" {
		t.Fatalf("unexpected dialect %q", statePayload.State.Dialect)
	}

	previewReq := httptest.NewRequest(http.MethodPost, "/rpc/docs/_admin/db/preview", strings.NewReader(`{"schema":"main","table":"users"}`))
	previewReq.Header.Set("Content-Type", "application/json")
	previewRec := httptest.NewRecorder()
	router.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("expected preview endpoint 200, got %d", previewRec.Code)
	}
	if fake.previewIn.Table != "users" {
		t.Fatalf("expected preview input table users, got %q", fake.previewIn.Table)
	}

	queryReq := httptest.NewRequest(http.MethodPost, "/rpc/docs/_admin/db/query", strings.NewReader(`{"query":"SELECT * FROM users"}`))
	queryReq.Header.Set("Content-Type", "application/json")
	queryRec := httptest.NewRecorder()
	router.ServeHTTP(queryRec, queryReq)
	if queryRec.Code != http.StatusOK {
		t.Fatalf("expected query endpoint 200, got %d", queryRec.Code)
	}
	if strings.TrimSpace(fake.queryIn.Query) != "SELECT * FROM users" {
		t.Fatalf("expected query input captured, got %q", fake.queryIn.Query)
	}
}

func TestRPCServeDocsDBExplorerStateErrorPayload(t *testing.T) {
	fake := &fakeDBExplorer{
		stateErr: errors.New("permission denied"),
	}
	router := NewRouter(WithDBExplorer(fake))
	router.HandleRPC(testHandler)
	router.ServeDocs()

	req := httptest.NewRequest(http.MethodGet, "/rpc/docs/_admin/db", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected state endpoint 200 with error payload, got %d", rec.Code)
	}
	var payload struct {
		Enabled bool   `json:"enabled"`
		Error   string `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if !payload.Enabled {
		t.Fatalf("expected enabled=true when explorer is attached")
	}
	if payload.Error != "permission denied" {
		t.Fatalf("unexpected error payload %q", payload.Error)
	}
}
