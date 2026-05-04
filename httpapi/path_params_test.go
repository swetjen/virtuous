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
