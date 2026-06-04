package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHTTPAPIGeneratedClientsLiveE2E(t *testing.T) {
	router := newLiveClientE2ERouter(t)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	t.Run("python", func(t *testing.T) {
		py := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) })
		dir := t.TempDir()
		pyPath := filepath.Join(dir, "client.gen.py")
		if err := os.WriteFile(pyPath, py, 0644); err != nil {
			t.Fatalf("write python client: %v", err)
		}
		snippet := pythonImportSnippet(pyPath) + `
from datetime import datetime

client = mod.create_client(base_url="` + server.URL + `")

mixed = mod.ContractsclientRuntimeMixedRequest(
    accountID="body-leak",
    iDs=["body-leak"],
    limit=99,
    name="mixed",
    count=3,
)
resp = client.live_mixed("acct 1", id=["a", "b"], body=mixed, limit=25)
assert resp.accepted is True

optional = client.live_optional()
assert optional.accepted is False

assert client.live_clear_cache("acct-2") is None

PgReq = next(getattr(mod, name) for name in dir(mod) if name.endswith("PgtypeRequest"))
pg_resp = client.live_pgtype(body=PgReq(
    text="hello",
    flag=True,
    small=12,
    num=123,
    big=9007199254740991,
    ratio32=1.5,
    ratio64=2.25,
    count=4294967295,
    when=datetime.fromisoformat("2025-01-02T03:04:05+00:00"),
    uuid="00112233-4455-6677-8899-aabbccddeeff",
    amount=123.45,
    timestamptz=datetime.fromisoformat("2025-01-02T03:04:05+00:00"),
    date=mod._date.fromisoformat("2025-01-02"),
    raw={"ok": True},
    legacy_json={"items": [1, "two", True]},
    legacy_jsonb=["a", {"b": 2}],
))
assert pg_resp.text == "hello"
assert pg_resp.flag is True
assert pg_resp.amount == 123.45
assert pg_resp.date.isoformat() == "2025-01-02"
assert pg_resp.legacy_jsonb[1]["b"] == 2
`
		if err := runPythonCommand("-c", snippet); err != nil {
			t.Fatalf("python live E2E failed: %v", err)
		}
	})

	t.Run("typescript", func(t *testing.T) {
		requireCommand(t, "node")
		requireCommand(t, "tsc")
		ts := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientTS(buf) })
		dir := t.TempDir()
		tsPath := filepath.Join(dir, "client.gen.ts")
		if err := os.WriteFile(tsPath, ts, 0644); err != nil {
			t.Fatalf("write ts client: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"type":"module"}`), 0644); err != nil {
			t.Fatalf("write package json: %v", err)
		}
		if err := runCommand("tsc", "--target", "ES2022", "--module", "Node16", "--moduleResolution", "node16", "--lib", "ES2022,DOM", "--outDir", dir, tsPath); err != nil {
			t.Fatalf("compile ts client: %v", err)
		}
		writeLiveNodeHarness(t, filepath.Join(dir, "harness.mjs"), server.URL, true)
		if err := runCommand("node", filepath.Join(dir, "harness.mjs")); err != nil {
			t.Fatalf("typescript live E2E failed: %v", err)
		}
	})

	t.Run("javascript", func(t *testing.T) {
		requireCommand(t, "node")
		js := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientJS(buf) })
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "client.gen.js"), js, 0644); err != nil {
			t.Fatalf("write js client: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"type":"module"}`), 0644); err != nil {
			t.Fatalf("write package json: %v", err)
		}
		writeLiveNodeHarness(t, filepath.Join(dir, "harness.mjs"), server.URL, false)
		if err := runCommand("node", filepath.Join(dir, "harness.mjs")); err != nil {
			t.Fatalf("javascript live E2E failed: %v", err)
		}
	})
}

func newLiveClientE2ERouter(t *testing.T) *Router {
	t.Helper()
	router := NewRouter()
	router.HandleTyped("PUT /contracts/{account_id}/mixed", WrapFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.PathValue("account_id"); got != "acct 1" {
			http.Error(w, "bad path account", http.StatusBadRequest)
			return
		}
		if got := r.URL.Query()["id"]; strings.Join(got, ",") != "a,b" {
			http.Error(w, "bad query id", http.StatusBadRequest)
			return
		}
		if got := r.URL.Query().Get("limit"); got != "25" {
			http.Error(w, "bad query limit", http.StatusBadRequest)
			return
		}
		var raw map[string]any
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(raw) != 2 || raw["name"] != "mixed" || raw["count"] != float64(3) {
			http.Error(w, "bad mixed body", http.StatusBadRequest)
			return
		}
		Encode(w, r, http.StatusOK, clientRuntimeResponse{Accepted: true})
	}, clientRuntimeMixedRequest{}, clientRuntimeResponse{}, HandlerMeta{
		Service:     "Contracts",
		Method:      "Mixed",
		OperationID: "live_mixed",
	}))
	router.HandleTyped("PATCH /contracts/optional", WrapFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > 0 {
			http.Error(w, "optional absent body should be empty", http.StatusBadRequest)
			return
		}
		Encode(w, r, http.StatusOK, clientRuntimeResponse{Accepted: false})
	}, Optional[optionalClientRequest](), clientRuntimeResponse{}, HandlerMeta{
		Service:     "Contracts",
		Method:      "Optional",
		OperationID: "live_optional",
	}))
	router.HandleTyped("DELETE /contracts/{account_id}/cache", WrapFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.PathValue("account_id"); got != "acct-2" {
			http.Error(w, "bad cache account", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}, HTTPPythonNoBodyRequest{}, NoResponse204{}, HandlerMeta{
		Service:     "Contracts",
		Method:      "ClearCache",
		OperationID: "live_clear_cache",
	}))
	router.HandleTyped("POST /db/pgtype", WrapFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := Decode[httpPgtypeRequest](r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		assertHTTPPgtypeDecoded(t, req)
		Encode(w, r, http.StatusOK, httpPgtypeResponse(req))
	}, httpPgtypeRequest{}, httpPgtypeResponse{}, HandlerMeta{
		Service:     "DB",
		Method:      "RoundTrip",
		OperationID: "live_pgtype",
	}))
	return router
}

func writeLiveNodeHarness(t *testing.T, path, baseURL string, tsClient bool) {
	t.Helper()
	createClient := `const client = createClient({ baseUrl: "` + baseURL + `" });`
	if !tsClient {
		createClient = `const client = createClient("` + baseURL + `");`
	}
	harness := `
import { createClient } from "./client.gen.js";

` + createClient + `

const mixed = await client.Contracts.mixed(
  { account_id: "acct 1" },
  { accountID: "body-leak", iDs: ["body-leak"], limit: 99, name: "mixed", count: 3 },
  { id: ["a", "b"], limit: 25 },
);
if (!mixed.accepted) throw new Error("mixed response failed");

const optional = await client.Contracts.optional();
if (optional.accepted !== false) throw new Error("optional response failed");

const cleared = await client.Contracts.clearCache({ account_id: "acct-2" });
if (cleared !== undefined) throw new Error("clear cache should return undefined");

const pg = await client.DB.roundTrip({
  text: "hello",
  flag: true,
  small: 12,
  num: 123,
  big: 9007199254740991,
  ratio32: 1.5,
  ratio64: 2.25,
  count: 4294967295,
  when: "2025-01-02T03:04:05Z",
  uuid: "00112233-4455-6677-8899-aabbccddeeff",
  amount: 123.45,
  timestamptz: "2025-01-02T03:04:05Z",
  date: "2025-01-02",
  raw: { ok: true },
  legacy_json: { items: [1, "two", true] },
  legacy_jsonb: ["a", { b: 2 }],
});
if (pg.text !== "hello" || pg.flag !== true || pg.amount !== 123.45 || pg.date !== "2025-01-02") {
  throw new Error("bad pgtype response " + JSON.stringify(pg));
}
if (pg.legacy_jsonb[1].b !== 2) throw new Error("bad pgtype jsonb");
`
	if err := os.WriteFile(path, []byte(harness), 0644); err != nil {
		t.Fatalf("write live node harness: %v", err)
	}
}
