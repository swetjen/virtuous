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
	router.Describe("POST /api/v1/reports", optionalClientRequest{}, testResponse{}, HandlerMeta{
		Service: "API",
		Method:  "PostApiV1Reports",
	})

	tsText := compileReactQueryTS(t, router)

	assertContains(t, tsText, "export const reactQueryClient = createClient('')")
	assertContains(t, tsText, "export type AuthOptions = {")
	assertContains(t, tsText, "export function createClient(basepath: string = \"/\")")
	assertNotContains(t, tsText, "from './")
	assertNotContains(t, tsText, "from '../")
	assertContains(t, tsText, "export function getUserQueryKey(pathParams?: { id: number }, query?: { limit?: number; active?: boolean; since?: string })")
	assertContains(t, tsText, "return ['GET /users/{id}', pathParams, query] as const")
	assertContains(t, tsText, "enabled: !!pathParams && pathParams.id !== undefined && pathParams.id !== null")
	assertContains(t, tsText, "requestOptions?: AuthOptions")
	assertContains(t, tsText, "queryFn: () => reactQueryClient.Users.getUser(pathParams!, query, requestOptions)")
	assertContains(t, tsText, "export function usePostApiV1Reports(requestOptions?: AuthOptions, mutationOptions?: UseMutationOptions<testResponse, Error, { request: optionalClientRequest }>)")
	assertContains(t, tsText, "mutationFn: (variables: { request: optionalClientRequest }) => reactQueryClient.API.postApiV1Reports(variables.request, requestOptions)")
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
	reactQueryPath := filepath.Join(dir, "react-query.gen.ts")
	if err := os.WriteFile(reactQueryPath, reactQueryTS, 0644); err != nil {
		t.Fatalf("write react query ts: %v", err)
	}
	linkInstalledPackage(t, dir, packagePath)

	if err := runCommand("tsc", "--noEmit", "--target", "ES2017", "--lib", "ES2017,DOM", "--jsx", "react-jsx", "--module", "commonjs", "--moduleResolution", "node", "--skipLibCheck", reactQueryPath); err != nil {
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
	assertContains(t, tsText, "export function getApiV1MeQueryOptions(requestOptions?: AuthOptions)")
	assertContains(t, tsText, "queryFn: () => reactQueryClient.API.getApiV1Me(requestOptions)")
	assertContains(t, tsText, "export function useGetApiV1Me(requestOptions?: AuthOptions, queryOptions?: Omit<UseQueryOptions<rqMeResponse, Error>, 'queryKey' | 'queryFn'>)")
	assertNotContains(t, tsText, "(,")
}

func TestReactQueryTSQueryRouteWithBody(t *testing.T) {
	router := NewRouter()
	router.Describe("GET /reports/search", rqGetWithBodyRequest{}, rqMutationResponse{}, HandlerMeta{
		Service: "Reports",
		Method:  "Search",
	})

	tsText := compileReactQueryTS(t, router)
	assertContains(t, tsText, "export function searchQueryKey(request: ReportsrqGetWithBodyRequest, query?: { limit?: number })")
	assertContains(t, tsText, "return ['GET /reports/search', request, query] as const")
	assertContains(t, tsText, "queryFn: () => reactQueryClient.Reports.search(request, query, requestOptions)")
	assertContains(t, tsText, "export function useSearch(request: ReportsrqGetWithBodyRequest, query?: { limit?: number }, requestOptions?: AuthOptions")
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
				"export function useCreate(requestOptions?: AuthOptions, mutationOptions?: UseMutationOptions<ReportsrqMutationResponse, Error, { request: ReportsoptionalClientRequest }>)",
				"mutationFn: (variables: { request: ReportsoptionalClientRequest }) => reactQueryClient.Reports.create(variables.request, requestOptions)",
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
				"export function useDelete(requestOptions?: AuthOptions, mutationOptions?: UseMutationOptions<void, Error, { pathParams: { id: string } }>)",
				"mutationFn: (variables: { pathParams: { id: string } }) => reactQueryClient.Users.delete(variables.pathParams, requestOptions)",
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
				"UseMutationOptions<UsersrqMutationResponse, Error, { pathParams: { id: string }; request: UsersrqMutationRequest; query?: { dryRun?: boolean } }>",
				"reactQueryClient.Users.update(variables.pathParams, variables.request, variables.query, requestOptions)",
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
				"export function usePing(requestOptions?: AuthOptions, mutationOptions?: UseMutationOptions<void, Error, void>)",
				"mutationFn: () => reactQueryClient.API.ping(requestOptions)",
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
				"mutationFn: (variables?: { request?: optionalClientRequest }) => reactQueryClient.API.optional(variables?.request, requestOptions)",
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

func TestReactQueryTSNoResponseRoutes(t *testing.T) {
	router := NewRouter()
	router.Describe("GET /health", nil, NoResponse204{}, HandlerMeta{Service: "API", Method: "Health"})
	router.Describe("POST /flush", nil, NoResponse200{}, HandlerMeta{Service: "API", Method: "Flush"})

	tsText := compileReactQueryTS(t, router)
	assertContains(t, tsText, "queryOptions?: Omit<UseQueryOptions<void, Error>, 'queryKey' | 'queryFn'>")
	assertContains(t, tsText, "export function useFlush(requestOptions?: AuthOptions, mutationOptions?: UseMutationOptions<void, Error, void>)")
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
	if !bytes.Equal(body, body2) {
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
	_, rendered, ok := strings.Cut(bodyText, "\n")
	if !ok {
		t.Fatalf("generated body missing hash line")
	}
	if got := clientgen.HashBytes([]byte(rendered)); got != hash.String() {
		t.Fatalf("hash = %s, want %s", hash.String(), got)
	}

	req := httptest.NewRequest(http.MethodGet, "/react-query.gen.ts", nil)
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

	router.ServeAllDocs(WithoutDocs(), WithReactQueryTSPath("/react-query.gen.ts"))
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
	reactQueryPath := filepath.Join(dir, "react-query.gen.ts")
	if err := os.WriteFile(reactQueryPath, reactQueryTS, 0644); err != nil {
		t.Fatalf("write react query ts: %v", err)
	}
	writeReactQueryStub(t, dir)

	if err := runCommand("tsc", "--noEmit", "--target", "ES2017", "--lib", "ES2017,DOM", "--module", "commonjs", "--moduleResolution", "node", reactQueryPath); err != nil {
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
	queryFn?: () => Promise<TQueryFnData> | TQueryFnData
	enabled?: boolean
	[key: string]: unknown
}

export declare function useQuery<TQueryFnData = unknown, TError = Error, TData = TQueryFnData, TQueryKey = readonly unknown[]>(
	options: UseQueryOptions<TQueryFnData, TError, TData, TQueryKey> & { queryKey: TQueryKey; queryFn: () => Promise<TQueryFnData> | TQueryFnData },
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
