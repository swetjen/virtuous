package jsondecode

import (
	"errors"
	"strings"
	"testing"
)

func TestDecodeStrictRejectsNestedDuplicateKeys(t *testing.T) {
	var v struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
	}

	err := Decode(strings.NewReader(`{"items":[{"name":"a","name":"b"}]}`), &v, StrictOptions())
	if err == nil {
		t.Fatalf("expected duplicate key error")
	}
	var duplicate DuplicateKeyError
	if !errors.As(err, &duplicate) {
		t.Fatalf("expected DuplicateKeyError, got %T: %v", err, err)
	}
	if duplicate.Key != "name" {
		t.Fatalf("expected duplicate key name, got %q", duplicate.Key)
	}
}
