package rpc

import (
	"bytes"
	"io"
	"testing"
)

func TestRPCLargeContractOutputIsDeterministic(t *testing.T) {
	router := NewRouter()
	router.SetTypeOverrides(map[string]TypeOverride{
		"PyContractDate": {
			JSType:        "string",
			PyType:        "date",
			OpenAPIType:   "string",
			OpenAPIFormat: "date",
		},
		"PyContractDecimal": {
			JSType:        "string",
			PyType:        "Decimal",
			OpenAPIType:   "string",
			OpenAPIFormat: "decimal",
		},
	})
	router.HandleRPC(pythonMegaContractHandler)
	router.HandleRPC(pgtypeHandler)

	assertRPCStableBytes(t, "openapi", func() ([]byte, error) { return router.OpenAPI() })
	assertRPCStableRender(t, "js", router.WriteClientJS)
	assertRPCStableRender(t, "ts", router.WriteClientTS)
	assertRPCStableRender(t, "py", router.WriteClientPY)
}

func assertRPCStableRender(t *testing.T, name string, render func(io.Writer) error) {
	t.Helper()
	assertRPCStableBytes(t, name, func() ([]byte, error) {
		var buf bytes.Buffer
		if err := render(&buf); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	})
}

func assertRPCStableBytes(t *testing.T, name string, render func() ([]byte, error)) {
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
