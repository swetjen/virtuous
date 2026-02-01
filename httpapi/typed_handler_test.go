package httpapi

import (
	"net/http"
	"testing"
)

type typedHandlerRequest struct {
	Name string `json:"name"`
}

type typedHandlerResponse struct {
	Name string `json:"name"`
}

func typedHandlerFunc(_ http.ResponseWriter, _ *http.Request) {}

func TestWrapFuncMetadata(t *testing.T) {
	meta := HandlerMeta{Service: "Test", Method: "WrapFunc"}
	handler := WrapFunc(typedHandlerFunc, typedHandlerRequest{}, typedHandlerResponse{}, meta)

	if handler.RequestType() == nil {
		t.Fatalf("expected request type")
	}
	if handler.ResponseType() == nil {
		t.Fatalf("expected response type")
	}
	if handler.Metadata().Service != "Test" || handler.Metadata().Method != "WrapFunc" {
		t.Fatalf("unexpected metadata")
	}
}

func TestTypedHandlerFuncMetadata(t *testing.T) {
	meta := HandlerMeta{Service: "Test", Method: "TypedHandlerFunc"}
	handler := TypedHandlerFunc{
		Handler: typedHandlerFunc,
		Req:     typedHandlerRequest{},
		Resp:    typedHandlerResponse{},
		Meta:    meta,
	}

	if handler.RequestType() == nil {
		t.Fatalf("expected request type")
	}
	if handler.ResponseType() == nil {
		t.Fatalf("expected response type")
	}
	if handler.Metadata().Service != "Test" || handler.Metadata().Method != "TypedHandlerFunc" {
		t.Fatalf("unexpected metadata")
	}
}
