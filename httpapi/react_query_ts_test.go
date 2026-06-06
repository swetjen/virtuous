package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/swetjen/virtuous/internal/clientgen"
)

type rqMeResponse struct {
	Name string `json:"name"`
}

type rqGetWithBodyRequest struct {
	Filter string `json:"filter"`
	Limit  int    `query:"limit,omitempty"`
}

type rqPathOnlyRequest struct {
	ID string `path:"id"`
}

type rqMutationRequest struct {
	ID     string `path:"id"`
	DryRun bool   `query:"dryRun,omitempty"`
	Name   string `json:"name"`
}

type rqMutationResponse struct {
	ID string `json:"id"`
}

type rqImportNested struct {
	Code string `json:"code"`
}

type rqImportResponse struct {
	Child rqImportNested `json:"child"`
}

func TestGeneratedReactQueryTSClientIsValid(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /users/{id}", typedParamHandler{}, testGuard{name: "ApiKeyAuth", in: "header", param: "X-API-Key"})
	router.HandleTyped("POST /assets/upload", multipartBodyHandler{})
	router.Describe("POST /api/v1/reports", optionalClientRequest{}, testResponse{}, HandlerMeta{
		Service: "API",
		Method:  "PostApiV1Reports",
	})

	tsText := compileReactQueryTS(t, router)

	assertContains(t, tsText, "export const virtuousClient = createClient({ baseUrl: '' })")
	assertContains(t, tsText, "export function configureVirtuousClient(options: ClientOptions)")
	assertContains(t, tsText, "export type RequestOptions = {")
	assertContains(t, tsText, "export type RequestAuth = {")
	assertContains(t, tsText, "apiKeyAuth?: string")
	assertContains(t, tsText, "signal?: AbortSignal")
	assertNotContains(t, tsText, "export type AuthOptions = RequestOptions")
	assertContains(t, tsText, "export class AuthNotReadyError extends Error")
	assertContains(t, tsText, "export function createClient(options: ClientOptions = {})")
	assertNotContains(t, tsText, "from './")
	assertNotContains(t, tsText, "from '../")
	assertContains(t, tsText, "export type UsersIdGetPathParams = {id: number; }")
	assertContains(t, tsText, "export type UsersIdGetQuery = {limit?: number;active?: boolean;since?: string; }")
	assertContains(t, tsText, "export function getUserQueryKey(pathParams?: UsersIdGetPathParams, query?: UsersIdGetQuery)")
	assertContains(t, tsText, "return ['GET /users/{id}', pathParams, query] as const")
	assertContains(t, tsText, "enabled: !!pathParams && pathParams.id !== undefined && pathParams.id !== null")
	assertContains(t, tsText, "queryFn: ({ signal }: { signal?: AbortSignal }) => virtuousClient.Users.getUser(pathParams!, query, { signal })")
	assertContains(t, tsText, "const auth = config.options?.auth ?? await _resolveAuth(clientOptions.auth)")
	assertContains(t, tsText, "throw new AuthNotReadyError(config.method + \" \" + config.path)")
	assertContains(t, tsText, "signal: config.options?.signal")
	assertContains(t, tsText, "FormData")
	assertContains(t, tsText, `["file", "file", true]`)
	assertNotContains(t, tsText, `"Content-Type": "multipart/form-data"`)
	assertContains(t, tsText, "export function usePostApiV1Reports(mutationOptions?: UseMutationOptions<testResponse, Error, { request: optionalClientRequest }>)")
	assertContains(t, tsText, "mutationFn: (variables: { request: optionalClientRequest }) => virtuousClient.API.postApiV1Reports(variables.request)")
}

func TestGeneratedReactQueryTSClientCompilesWithInstalledTanStackTypes(t *testing.T) {
	packagePath := installedTanStackReactQueryPackage(t)
	if packagePath == "" {
		t.Skip("@tanstack/react-query is not installed locally")
	}

	router := NewRouter()
	router.HandleTyped("GET /users/{id}", typedParamHandler{})
	router.Describe("POST /api/v1/reports", optionalClientRequest{}, testResponse{}, HandlerMeta{
		Service: "API",
		Method:  "PostApiV1Reports",
	})

	reactQueryTS := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteReactQueryTS(buf) })

	dir := t.TempDir()
	reactQueryPath := filepath.Join(dir, "react-query.client.gen.ts")
	if err := os.WriteFile(reactQueryPath, reactQueryTS, 0644); err != nil {
		t.Fatalf("write react query ts: %v", err)
	}
	linkInstalledPackage(t, dir, packagePath)

	if err := runCommand("tsc", "--noEmit", "--strict", "--target", "ES2017", "--lib", "ES2017,DOM", "--jsx", "react-jsx", "--module", "Node16", "--moduleResolution", "node16", "--skipLibCheck", reactQueryPath); err != nil {
		t.Fatalf("tsc check with installed @tanstack/react-query failed: %v", err)
	}
}

func TestReactQueryTSNoParamQueryRoute(t *testing.T) {
	router := NewRouter()
	router.Describe("GET /api/v1/me", nil, rqMeResponse{}, HandlerMeta{
		Service: "API",
		Method:  "GetApiV1Me",
	})

	tsText := compileReactQueryTS(t, router)
	assertContains(t, tsText, "export function getApiV1MeQueryKey()")
	assertContains(t, tsText, "return ['GET /api/v1/me'] as const")
	assertContains(t, tsText, "export function getApiV1MeQueryOptions()")
	assertContains(t, tsText, "queryFn: ({ signal }: { signal?: AbortSignal }) => virtuousClient.API.getApiV1Me({ signal })")
	assertContains(t, tsText, "export function useGetApiV1Me(queryOptions?: Omit<UseQueryOptions<rqMeResponse, Error>, 'queryKey' | 'queryFn'>)")
	assertNotContains(t, tsText, "(,")
}

func TestReactQueryTSQueryRouteWithBody(t *testing.T) {
	router := NewRouter()
	router.Describe("GET /reports/search", rqGetWithBodyRequest{}, rqMutationResponse{}, HandlerMeta{
		Service: "Reports",
		Method:  "Search",
	})

	tsText := compileReactQueryTS(t, router)
	assertContains(t, tsText, "export type ReportsSearchGetQuery = {limit?: number; }")
	assertContains(t, tsText, "export function searchQueryKey(request: ReportsrqGetWithBodyRequest, query?: ReportsSearchGetQuery)")
	assertContains(t, tsText, "return ['GET /reports/search', request, query] as const")
	assertContains(t, tsText, "queryFn: ({ signal }: { signal?: AbortSignal }) => virtuousClient.Reports.search(request, query, { signal })")
	assertContains(t, tsText, "export function useSearch(request: ReportsrqGetWithBodyRequest, query?: ReportsSearchGetQuery, queryOptions?: Omit<UseQueryOptions<ReportsrqMutationResponse, Error>, 'queryKey' | 'queryFn'>)")
}

func TestReactQueryTSMutationVariableShapes(t *testing.T) {
	tests := []struct {
		name string
		make func() *Router
		want []string
	}{
		{
			name: "body only",
			make: func() *Router {
				router := NewRouter()
				router.Describe("POST /reports", optionalClientRequest{}, rqMutationResponse{}, HandlerMeta{Service: "Reports", Method: "Create"})
				return router
			},
			want: []string{
				"export function useCreate(mutationOptions?: UseMutationOptions<ReportsrqMutationResponse, Error, { request: ReportsoptionalClientRequest }>)",
				"mutationFn: (variables: { request: ReportsoptionalClientRequest }) => virtuousClient.Reports.create(variables.request)",
			},
		},
		{
			name: "path only",
			make: func() *Router {
				router := NewRouter()
				router.Describe("DELETE /users/{id}", rqPathOnlyRequest{}, NoResponse204{}, HandlerMeta{Service: "Users", Method: "Delete"})
				return router
			},
			want: []string{
				"export function useDelete(mutationOptions?: UseMutationOptions<void, Error, { pathParams: UsersIdDeletePathParams }>)",
				"mutationFn: (variables: { pathParams: UsersIdDeletePathParams }) => virtuousClient.Users.delete(variables.pathParams)",
			},
		},
		{
			name: "path body query",
			make: func() *Router {
				router := NewRouter()
				router.Describe("PATCH /users/{id}", rqMutationRequest{}, rqMutationResponse{}, HandlerMeta{Service: "Users", Method: "Update"})
				return router
			},
			want: []string{
				"UseMutationOptions<UsersrqMutationResponse, Error, { pathParams: UsersIdPatchPathParams; request: UsersrqMutationRequest; query?: UsersIdPatchQuery }>",
				"virtuousClient.Users.update(variables.pathParams, variables.request, variables.query)",
			},
		},
		{
			name: "no variables",
			make: func() *Router {
				router := NewRouter()
				router.Describe("POST /ping", nil, NoResponse200{}, HandlerMeta{Service: "API", Method: "Ping"})
				return router
			},
			want: []string{
				"export function usePing(mutationOptions?: UseMutationOptions<void, Error, void>)",
				"mutationFn: () => virtuousClient.API.ping()",
			},
		},
		{
			name: "optional body",
			make: func() *Router {
				router := NewRouter()
				router.Describe("POST /optional", Optional[optionalClientRequest](), rqMutationResponse{}, HandlerMeta{Service: "API", Method: "Optional"})
				return router
			},
			want: []string{
				"UseMutationOptions<rqMutationResponse, Error, { request?: optionalClientRequest } | void>",
				"mutationFn: (variables?: { request?: optionalClientRequest }) => virtuousClient.API.optional(variables?.request)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tsText := compileReactQueryTS(t, tt.make())
			for _, want := range tt.want {
				assertContains(t, tsText, want)
			}
		})
	}
}

func TestReactQueryTSMixedRequestAndPgtypeShapes(t *testing.T) {
	router := NewRouter()
	router.Describe("PUT /contracts/{account_id}/mixed", clientRuntimeMixedRequest{}, clientRuntimeResponse{}, HandlerMeta{
		Service: "Contracts",
		Method:  "Mixed",
	})
	router.Describe("PATCH /contracts/optional", Optional[optionalClientRequest](), clientRuntimeResponse{}, HandlerMeta{
		Service: "Contracts",
		Method:  "Optional",
	})
	router.Describe("DELETE /contracts/{account_id}/cache", HTTPPythonNoBodyRequest{}, NoResponse204{}, HandlerMeta{
		Service: "Contracts",
		Method:  "ClearCache",
	})
	router.Describe("POST /db/pgtype", httpPgtypeRequest{}, httpPgtypeResponse{}, HandlerMeta{
		Service: "DB",
		Method:  "RoundTrip",
	})

	tsText := compileReactQueryTS(t, router)
	assertContains(t, tsText, "text: string | null;")
	assertContains(t, tsText, "flag: boolean | null;")
	assertContains(t, tsText, "amount: number | null;")
	assertContains(t, tsText, "when: string | null;")
	assertContains(t, tsText, "legacy_json: object|any[] | null;")
	assertContains(t, tsText, "raw: object|any[];")
	assertNotContains(t, tsText, "export interface Text")
	assertNotContains(t, tsText, "export interface Numeric")

	assertContains(t, tsText, "export type ContractsAccountIdMixedPutPathParams = {account_id: string; }")
	assertContains(t, tsText, "export type ContractsAccountIdMixedPutQuery = {id: string[];limit?: number; }")
	assertContains(t, tsText, "UseMutationOptions<ContractsclientRuntimeResponse, Error, { pathParams: ContractsAccountIdMixedPutPathParams; request: ContractsclientRuntimeMixedRequest; query?: ContractsAccountIdMixedPutQuery }>")
	assertContains(t, tsText, "virtuousClient.Contracts.mixed(variables.pathParams, variables.request, variables.query)")
	assertContains(t, tsText, `["name", "name", false]`)
	assertContains(t, tsText, `["count", "count", false]`)
	assertNotContains(t, tsText, `["account_id", "accountID", false]`)
	assertNotContains(t, tsText, `["id", "iDs", false]`)
	assertContains(t, tsText, "UseMutationOptions<ContractsclientRuntimeResponse, Error, { request?: ContractsoptionalClientRequest } | void>")
	assertContains(t, tsText, "UseMutationOptions<void, Error, { pathParams: ContractsAccountIdCacheDeletePathParams }>")
}

func TestReactQueryTSRawClientRuntimeRequestEncoding(t *testing.T) {
	requireCommand(t, "node")
	requireCommand(t, "tsc")

	router := newRuntimeClientContractRouter()
	reactQueryTS := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteReactQueryTS(buf) })

	dir := t.TempDir()
	reactQueryPath := filepath.Join(dir, "react-query.client.gen.ts")
	if err := os.WriteFile(reactQueryPath, reactQueryTS, 0644); err != nil {
		t.Fatalf("write react query ts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"type":"module"}`), 0644); err != nil {
		t.Fatalf("write package json: %v", err)
	}
	writeReactQueryRuntimeStub(t, dir)
	if err := runCommand("tsc", "--target", "ES2022", "--module", "Node16", "--moduleResolution", "node16", "--lib", "ES2022,DOM", "--outDir", dir, reactQueryPath); err != nil {
		t.Fatalf("compile react query client: %v", err)
	}

	harness := `
import { createClient } from "./react-query.client.gen.js";

const calls = [];

class FakeResponse {
  constructor(status, body) {
    this.status = status;
    this.statusText = status === 204 ? "No Content" : "OK";
    this.ok = status >= 200 && status < 300;
    this._body = body;
  }
  async text() { return this._body; }
  async arrayBuffer() { return new TextEncoder().encode(this._body).buffer; }
}

globalThis.fetch = async (url, init = {}) => {
  calls.push({ url: String(url), init });
  if (String(url).includes("/mixed")) {
    const parsed = new URL(String(url));
    if (parsed.pathname !== "/contracts/acct%201/mixed") throw new Error("bad mixed path " + parsed.pathname);
    if (parsed.searchParams.getAll("id").join(",") !== "a,b") throw new Error("bad id query " + parsed.search);
    if (parsed.searchParams.get("limit") !== "25") throw new Error("bad limit query " + parsed.search);
    const body = JSON.parse(init.body);
    if (JSON.stringify(body) !== JSON.stringify({ name: "mixed", count: 3 })) throw new Error("bad mixed body " + JSON.stringify(body));
    return new FakeResponse(200, '{"accepted":true}');
  }
  if (String(url).endsWith("/optional")) {
    if ("body" in init) throw new Error("optional absent body should not dispatch a body");
    return new FakeResponse(200, '{"accepted":false}');
  }
  if (String(url).endsWith("/cache")) {
    if ("body" in init) throw new Error("no-body route should not dispatch a body");
    return new FakeResponse(204, "");
  }
  throw new Error("unexpected fetch " + url);
};

const client = createClient({ baseUrl: "https://core.example" });
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

if (calls.length !== 3) throw new Error("unexpected call count " + calls.length);
`
	harnessPath := filepath.Join(dir, "harness.mjs")
	if err := os.WriteFile(harnessPath, []byte(harness), 0644); err != nil {
		t.Fatalf("write react query runtime harness: %v", err)
	}
	if err := runCommand("node", harnessPath); err != nil {
		t.Fatalf("react query raw client runtime failed: %v", err)
	}
}

func TestReactQueryTSNoResponseRoutes(t *testing.T) {
	router := NewRouter()
	router.Describe("GET /health", nil, NoResponse204{}, HandlerMeta{Service: "API", Method: "Health"})
	router.Describe("POST /flush", nil, NoResponse200{}, HandlerMeta{Service: "API", Method: "Flush"})

	tsText := compileReactQueryTS(t, router)
	assertContains(t, tsText, "queryOptions?: Omit<UseQueryOptions<void, Error>, 'queryKey' | 'queryFn'>")
	assertContains(t, tsText, "export function useFlush(mutationOptions?: UseMutationOptions<void, Error, void>)")
}

func TestReactQueryTSAuthFallbackOnlyAppliesToSingleGuardRoutes(t *testing.T) {
	router := NewRouter()
	router.Describe("GET /single", nil, NoResponse200{}, HandlerMeta{Service: "Secure", Method: "Single"}, testGuard{name: "ApiKeyAuth", in: "header", param: "X-API-Key"})
	router.Describe("GET /either", nil, NoResponse200{}, HandlerMeta{Service: "Secure", Method: "Either"}, AuthAny(
		testGuard{name: "ApiKeyAuth", in: "header", param: "X-API-Key"},
		testGuard{name: "TokenAuth", in: "header", param: "Authorization"},
	))

	tsText := compileReactQueryTS(t, router)
	assertContains(t, tsText, `{ name: "apiKeyAuth", in: "header", param: "X-API-Key", prefix: "", generic: true }`)
	assertContains(t, tsText, `{ name: "apiKeyAuth", in: "header", param: "X-API-Key", prefix: "", generic: false }`)
	assertContains(t, tsText, `{ name: "tokenAuth", in: "header", param: "Authorization", prefix: "", generic: false }`)
	assertContains(t, tsText, "throw new AuthNotReadyError(config.method + \" \" + config.path)")
}

func TestReactQueryTSDuplicateMethodNamesAreServicePrefixed(t *testing.T) {
	router := NewRouter()
	router.Describe("GET /users", nil, rqMeResponse{}, HandlerMeta{Service: "Users", Method: "Get"})
	router.Describe("GET /reports", nil, rqMutationResponse{}, HandlerMeta{Service: "Reports", Method: "Get"})

	tsText := compileReactQueryTS(t, router)
	assertContains(t, tsText, "export function usersGetQueryKey()")
	assertContains(t, tsText, "export function reportsGetQueryKey()")
	assertContains(t, tsText, "export function useUsersGet(")
	assertContains(t, tsText, "export function useReportsGet(")
	assertNotContains(t, tsText, "export function getQueryKey()")
	assertNotContains(t, tsText, "export function useGet(")
}

func TestReactQueryTSHashAndServe(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /users/{id}", typedParamHandler{})

	body := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteReactQueryTS(buf) })
	body2 := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteReactQueryTS(buf) })
	if !bytes.Equal(stripGeneratedTimestamp(body), stripGeneratedTimestamp(body2)) {
		t.Fatalf("react query output is not stable across renders")
	}
	var hash bytes.Buffer
	if err := router.WriteReactQueryTSHash(&hash); err != nil {
		t.Fatalf("write hash: %v", err)
	}
	if hash.Len() != 64 {
		t.Fatalf("hash length = %d, want 64", hash.Len())
	}
	bodyText := string(body)
	if !strings.Contains(bodyText, "// Virtuous React Query client hash: "+hash.String()) {
		t.Fatalf("generated body missing hash prefix")
	}
	_, afterHash, ok := strings.Cut(bodyText, "\n")
	if !ok {
		t.Fatalf("generated body missing hash line")
	}
	expectedHeader := "// Code generated by Virtuous " + clientgen.VirtuousVersionLabel() + " on "
	if !strings.Contains(afterHash, expectedHeader) {
		t.Fatalf("generated body missing versioned generated header")
	}
	_, rendered, ok := strings.Cut(afterHash, "\n")
	if !ok {
		t.Fatalf("generated body missing generated header line")
	}
	if got := clientgen.HashBytes([]byte(rendered)); got != hash.String() {
		t.Fatalf("hash = %s, want %s", hash.String(), got)
	}

	req := httptest.NewRequest(http.MethodGet, "/react-query.client.gen.ts", nil)
	rec := httptest.NewRecorder()
	router.ServeReactQueryTS(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/typescript" {
		t.Fatalf("content type = %q, want application/typescript", got)
	}
	if !strings.Contains(rec.Body.String(), "useGetUser") {
		t.Fatalf("served body missing generated hook")
	}

	defaultRouter := NewRouter()
	defaultRouter.HandleTyped("GET /users/{id}", typedParamHandler{})
	defaultRouter.ServeAllDocs(WithoutDocs())
	rec = httptest.NewRecorder()
	defaultRouter.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("default ServeAllDocs react query status = %d, want 404", rec.Code)
	}

	router.ServeAllDocs(WithoutDocs(), WithReactQueryTSPath("/react-query.client.gen.ts"))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ServeAllDocs react query status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "useGetUser") {
		t.Fatalf("ServeAllDocs react query body missing generated hook")
	}
}

func TestReactQueryTSEmbedsReferencedTypes(t *testing.T) {
	router := NewRouter()
	router.Describe("GET /import", nil, rqImportResponse{}, HandlerMeta{Service: "API", Method: "Import"})

	tsText := compileReactQueryTS(t, router)
	assertContains(t, tsText, "export interface rqImportResponse")
	assertContains(t, tsText, "export interface rqImportNested")
	assertContains(t, tsText, "child: rqImportNested")
}

func compileReactQueryTS(t *testing.T, router *Router) string {
	t.Helper()
	reactQueryTS := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteReactQueryTS(buf) })

	dir := t.TempDir()
	reactQueryPath := filepath.Join(dir, "react-query.client.gen.ts")
	if err := os.WriteFile(reactQueryPath, reactQueryTS, 0644); err != nil {
		t.Fatalf("write react query ts: %v", err)
	}
	writeReactQueryStub(t, dir)

	if err := runCommand("tsc", "--noEmit", "--strict", "--target", "ES2017", "--lib", "ES2017,DOM", "--module", "Node16", "--moduleResolution", "node16", reactQueryPath); err != nil {
		t.Fatalf("tsc check failed: %v", err)
	}
	return string(reactQueryTS)
}

func installedTanStackReactQueryPackage(t *testing.T) string {
	t.Helper()
	nodePath, err := exec.LookPath("node")
	if err != nil {
		return ""
	}
	cmd := exec.Command(nodePath, "-e", "try { console.log(require.resolve('@tanstack/react-query/package.json')) } catch (e) { process.exit(2) }")
	cmd.Dir = "."
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	packageJSON := strings.TrimSpace(string(output))
	if packageJSON == "" {
		return ""
	}
	return filepath.Dir(packageJSON)
}

func linkInstalledPackage(t *testing.T, dir, packagePath string) {
	t.Helper()
	scopeDir := filepath.Join(dir, "node_modules", "@tanstack")
	if err := os.MkdirAll(scopeDir, 0755); err != nil {
		t.Fatalf("make @tanstack dir: %v", err)
	}
	target := filepath.Join(scopeDir, "react-query")
	if err := os.Symlink(packagePath, target); err != nil {
		t.Fatalf("link @tanstack/react-query: %v", err)
	}
}

func writeReactQueryStub(t *testing.T, dir string) {
	t.Helper()
	stubDir := filepath.Join(dir, "node_modules", "@tanstack", "react-query")
	if err := os.MkdirAll(stubDir, 0755); err != nil {
		t.Fatalf("make react-query stub dir: %v", err)
	}
	stub := `export type UseQueryOptions<TQueryFnData = unknown, TError = Error, TData = TQueryFnData, TQueryKey = readonly unknown[]> = {
	queryKey?: TQueryKey
	queryFn?: (context: { signal?: AbortSignal }) => Promise<TQueryFnData> | TQueryFnData
	enabled?: boolean
	[key: string]: unknown
}

export declare function useQuery<TQueryFnData = unknown, TError = Error, TData = TQueryFnData, TQueryKey = readonly unknown[]>(
	options: UseQueryOptions<TQueryFnData, TError, TData, TQueryKey> & { queryKey: TQueryKey; queryFn: (context: { signal?: AbortSignal }) => Promise<TQueryFnData> | TQueryFnData },
): unknown

export type UseMutationOptions<TData = unknown, TError = Error, TVariables = void> = {
	mutationFn?: (variables: TVariables) => Promise<TData> | TData
	[key: string]: unknown
}

export declare function useMutation<TData = unknown, TError = Error, TVariables = void>(
	options: UseMutationOptions<TData, TError, TVariables> & { mutationFn: (variables: TVariables) => Promise<TData> | TData },
): unknown
`
	if err := os.WriteFile(filepath.Join(stubDir, "index.d.ts"), []byte(stub), 0644); err != nil {
		t.Fatalf("write react-query stub: %v", err)
	}
}

func writeReactQueryRuntimeStub(t *testing.T, dir string) {
	t.Helper()
	writeReactQueryStub(t, dir)
	stubDir := filepath.Join(dir, "node_modules", "@tanstack", "react-query")
	if err := os.WriteFile(filepath.Join(stubDir, "package.json"), []byte(`{"type":"module","main":"./index.js","types":"./index.d.ts"}`), 0644); err != nil {
		t.Fatalf("write react-query runtime package json: %v", err)
	}
	stub := `export function useQuery(options) { return { options } }
export function useMutation(options) { return { options } }
`
	if err := os.WriteFile(filepath.Join(stubDir, "index.js"), []byte(stub), 0644); err != nil {
		t.Fatalf("write react-query runtime stub: %v", err)
	}
}

func assertContains(t *testing.T, text, want string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Fatalf("generated output missing %q", want)
	}
}

func assertNotContains(t *testing.T, text, unwanted string) {
	t.Helper()
	if strings.Contains(text, unwanted) {
		t.Fatalf("generated output unexpectedly contains %q", unwanted)
	}
}
