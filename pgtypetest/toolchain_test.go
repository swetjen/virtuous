package pgtypetest

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Minimal local copies of the toolchain helpers used by the root module's
// generated-client tests. They live here rather than being imported so this
// module depends only on the public Virtuous API.

// requireCommand skips the test when name is not on PATH.
func requireCommand(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s is not installed", name)
	}
}

// runCommand runs name with args and returns combined output on failure.
func runCommand(name string, args ...string) error {
	path, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("%s is required: %w", name, err)
	}
	return runResolved(path, args...)
}

// runPython runs the Python interpreter through uv so the version is pinned.
func runPython(args ...string) error {
	path, err := exec.LookPath("uv")
	if err != nil {
		return fmt.Errorf("uv is required for generated Python contract tests: %w", err)
	}
	uvArgs := append([]string{"run", "--python", "3.12", "python"}, args...)
	return runResolved(path, uvArgs...)
}

func runResolved(path string, args ...string) error {
	output, err := exec.Command(path, args...).CombinedOutput()
	if err == nil {
		return nil
	}
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, trimmed)
}

// pythonImportSnippet loads a generated client file as module `mod`.
func pythonImportSnippet(path string) string {
	return "import importlib.util, sys; spec = importlib.util.spec_from_file_location('client_gen', r'" + path + "'); mod = importlib.util.module_from_spec(spec); sys.modules['client_gen'] = mod; spec.loader.exec_module(mod)"
}

func renderClient(t *testing.T, fn func(io.Writer) error) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := fn(&buf); err != nil {
		t.Fatalf("render client: %v", err)
	}
	return buf.Bytes()
}

// writePythonClient renders a Python client to a temp file and byte-compiles it.
func writePythonClient(t *testing.T, py []byte) string {
	t.Helper()
	pyPath := filepath.Join(t.TempDir(), "client.gen.py")
	if err := os.WriteFile(pyPath, py, 0644); err != nil {
		t.Fatalf("write python client: %v", err)
	}
	if err := runPython("-m", "py_compile", pyPath); err != nil {
		t.Fatalf("python py_compile failed: %v", err)
	}
	return pyPath
}

// writeESMDir writes a generated client into a fresh ESM package directory
// and returns the directory.
func writeESMDir(t *testing.T, fileName string, content []byte) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, fileName), content, 0644); err != nil {
		t.Fatalf("write %s: %v", fileName, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"type":"module"}`), 0644); err != nil {
		t.Fatalf("write package json: %v", err)
	}
	return dir
}

// compileReactQueryTS type-checks a rendered React Query client with tsc
// against a stub of @tanstack/react-query and returns the source.
func compileReactQueryTS(t *testing.T, render func(io.Writer) error) string {
	t.Helper()
	requireCommand(t, "tsc")
	reactQueryTS := renderClient(t, render)

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
		t.Fatalf("write react-query stub types: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stubDir, "package.json"), []byte(`{"name":"@tanstack/react-query","version":"0.0.0-stub","types":"index.d.ts"}`), 0644); err != nil {
		t.Fatalf("write react-query stub package: %v", err)
	}
}
