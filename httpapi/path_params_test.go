package httpapi

import (
	"reflect"
	"testing"
	"time"
)

type pathDefaultRequest struct {
	UserID int       `path:","`
	Since  time.Time `path:"since"`
}

type pathInvalidJSONRequest struct {
	ID int `path:"id" json:"id"`
}

type pathInvalidQueryRequest struct {
	ID int `path:"id" query:"id"`
}

type pathInvalidArrayRequest struct {
	IDs []int `path:"id"`
}

type pathInvalidOptionRequest struct {
	ID int `path:"id,omitempty"`
}

func TestPathParamsForTypedFields(t *testing.T) {
	params, err := pathParamsFor(reflect.TypeOf(pathDefaultRequest{}))
	if err != nil {
		t.Fatalf("path params: %v", err)
	}
	if len(params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(params))
	}
	if params[0].Name != "userID" || params[0].Type.Kind() != reflect.Int {
		t.Fatalf("unexpected default path param: %#v", params[0])
	}
	if params[1].Name != "since" || params[1].Type != reflect.TypeOf(time.Time{}) {
		t.Fatalf("unexpected time path param: %#v", params[1])
	}
}

func TestPathParamsForRejectsConflictingTags(t *testing.T) {
	if _, err := pathParamsFor(reflect.TypeOf(pathInvalidJSONRequest{})); err == nil {
		t.Fatalf("expected json/path conflict error")
	}
	if _, err := pathParamsFor(reflect.TypeOf(pathInvalidQueryRequest{})); err == nil {
		t.Fatalf("expected query/path conflict error")
	}
}

func TestPathParamsForRejectsArrays(t *testing.T) {
	if _, err := pathParamsFor(reflect.TypeOf(pathInvalidArrayRequest{})); err == nil {
		t.Fatalf("expected array path param error")
	}
}

func TestPathParamsForRejectsUnsupportedOptions(t *testing.T) {
	if _, err := pathParamsFor(reflect.TypeOf(pathInvalidOptionRequest{})); err == nil {
		t.Fatalf("expected unsupported path option error")
	}
}

type pathEmbeddedBase struct {
	WidgetID int64 `path:"widgetId"`
}

type pathEmbeddedRequest struct {
	pathEmbeddedBase
	Name string `json:"name"`
}

func TestPathParamsForEmbeddedStruct(t *testing.T) {
	params, err := pathParamsFor(reflect.TypeOf(pathEmbeddedRequest{}))
	if err != nil {
		t.Fatalf("path params: %v", err)
	}
	if len(params) != 1 {
		t.Fatalf("expected 1 promoted path param, got %d: %#v", len(params), params)
	}
	if params[0].Name != "widgetId" {
		t.Fatalf("param name = %q, want widgetId", params[0].Name)
	}
	if params[0].Type.Kind() != reflect.Int64 {
		t.Fatalf("param type = %v, want int64", params[0].Type)
	}
}
