package httpapi

import (
	"bytes"
	"io"
	"testing"
)

func TestHTTPAPILargeContractOutputIsDeterministic(t *testing.T) {
	router := NewRouter()
	router.SetTypeOverrides(httpPythonContractTypeOverrides())
	router.Describe("POST /contracts/python/mega", HTTPPythonMegaRequest{}, HTTPPythonMegaResponse{}, HandlerMeta{
		Service:     "Contracts",
		Method:      "Mega",
		OperationID: "python_mega",
	})
	router.Describe("PUT /contracts/{account_id}/mixed", HTTPPythonMixedRequest{}, HTTPPythonOptionalResponse{}, HandlerMeta{
		Service:     "Contracts",
		Method:      "Mixed",
		OperationID: "python_mixed",
	})
	router.Describe("POST /db/pgtype", httpPgtypeRequest{}, httpPgtypeResponse{}, HandlerMeta{
		Service:     "DB",
		Method:      "RoundTrip",
		OperationID: "pgtype_round_trip",
	})

	assertStableBytes(t, "openapi", func() ([]byte, error) { return router.OpenAPI() })
	assertStableRender(t, "js", router.WriteClientJS)
	assertStableRender(t, "ts", router.WriteClientTS)
	assertStableRender(t, "py", router.WriteClientPY)
	assertStableRender(t, "react-query-ts", router.WriteReactQueryTS)
}

func assertStableRender(t *testing.T, name string, render func(io.Writer) error) {
	t.Helper()
	assertStableBytes(t, name, func() ([]byte, error) {
		var buf bytes.Buffer
		if err := render(&buf); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	})
}

func assertStableBytes(t *testing.T, name string, render func() ([]byte, error)) {
	t.Helper()
	first, err := render()
	if err != nil {
		t.Fatalf("%s first render: %v", name, err)
	}
	second, err := render()
	if err != nil {
		t.Fatalf("%s second render: %v", name, err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("%s output is not deterministic", name)
	}
}
