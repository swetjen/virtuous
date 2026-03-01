package httpapi

import (
	"reflect"
	"testing"
)

type optionalTypeRequest struct {
	Name string `json:"name"`
}

func TestOptionalRequestTypeFromGeneric(t *testing.T) {
	info := resolveRequestType(Optional[optionalTypeRequest]())
	if !info.Present {
		t.Fatalf("expected request type to be present")
	}
	if !info.Optional {
		t.Fatalf("expected request type to be optional")
	}
	if info.Type != reflect.TypeOf(optionalTypeRequest{}) {
		t.Fatalf("unexpected request type: %v", info.Type)
	}
}

func TestOptionalRequestTypeFromValue(t *testing.T) {
	info := resolveRequestType(Optional(optionalTypeRequest{}))
	if !info.Present {
		t.Fatalf("expected request type to be present")
	}
	if !info.Optional {
		t.Fatalf("expected request type to be optional")
	}
	if info.Type != reflect.TypeOf(optionalTypeRequest{}) {
		t.Fatalf("unexpected request type: %v", info.Type)
	}
}
