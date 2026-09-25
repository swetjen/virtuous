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

type queryEmbeddedBase struct {
	Limit int    `query:"limit,optional"`
	Name  string `json:"name"`
}

type queryEmbeddedRequest struct {
	queryEmbeddedBase
	Query string `query:"q"`
}

func TestQueryParamsForEmbeddedStruct(t *testing.T) {
	info, err := queryParamsFor(reflect.TypeOf(queryEmbeddedRequest{}))
	if err != nil {
		t.Fatalf("query params: %v", err)
	}
	if len(info.Params) != 2 {
		t.Fatalf("expected 2 params (promoted + top-level), got %d: %#v", len(info.Params), info.Params)
	}
	byName := map[string]queryParam{}
	for _, param := range info.Params {
		byName[param.Name] = param
	}
	limit, ok := byName["limit"]
	if !ok {
		t.Fatalf("embedded query field should be promoted to a param: %#v", info.Params)
	}
	if !limit.Optional {
		t.Fatalf("promoted limit param should honor optional tag")
	}
	if _, ok := byName["q"]; !ok {
		t.Fatalf("top-level query field missing: %#v", info.Params)
	}
	if info.BodyFields != 1 {
		t.Fatalf("expected 1 body field (promoted name), got %d", info.BodyFields)
	}
	if _, ok := info.QueryFieldSet["Limit"]; !ok {
		t.Fatalf("QueryFieldSet should contain promoted field name, got %v", info.QueryFieldSet)
	}
}
