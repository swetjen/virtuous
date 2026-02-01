package httpapi

import (
	"reflect"
	"testing"
)

type queryOptionalRequest struct {
	Query string `query:"q,omitempty"`
	Limit int    `query:"limit"`
}

type queryArrayRequest struct {
	IDs []string `query:"id,omitempty"`
}

type queryInvalidRequest struct {
	Query string `query:"q" json:"q"`
}

type queryNestedRequest struct {
	Filters map[string]string `query:"filters"`
}

func TestQueryParamsForOptional(t *testing.T) {
	info, err := queryParamsFor(reflect.TypeOf(queryOptionalRequest{}))
	if err != nil {
		t.Fatalf("query params: %v", err)
	}
	if len(info.Params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(info.Params))
	}
	if info.Params[0].Optional != true && info.Params[1].Optional != true {
		t.Fatalf("expected at least one optional param")
	}
}

func TestQueryParamsForArray(t *testing.T) {
	info, err := queryParamsFor(reflect.TypeOf(queryArrayRequest{}))
	if err != nil {
		t.Fatalf("query params: %v", err)
	}
	if len(info.Params) != 1 || !info.Params[0].IsArray {
		t.Fatalf("expected array param")
	}
}

func TestQueryParamsForRejectsJSONTag(t *testing.T) {
	_, err := queryParamsFor(reflect.TypeOf(queryInvalidRequest{}))
	if err == nil {
		t.Fatalf("expected json/query conflict error")
	}
}

func TestQueryParamsForRejectsNested(t *testing.T) {
	_, err := queryParamsFor(reflect.TypeOf(queryNestedRequest{}))
	if err == nil {
		t.Fatalf("expected nested query error")
	}
}
