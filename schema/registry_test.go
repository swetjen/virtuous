package schema

import (
	"reflect"
	"testing"

	testa "github.com/swetjen/virtuous/internal/testtypes/a"
	testb "github.com/swetjen/virtuous/internal/testtypes/b"
)

func TestRegistrySchemaNameCollisionFallsBack(t *testing.T) {
	registry := NewRegistry(nil)
	registry.AddType(testa.User{})
	registry.AddType(testb.User{})

	objects := registry.Objects()
	if len(objects) != 2 {
		t.Fatalf("objects = %d, want 2", len(objects))
	}
	if !containsObjectName(objects, "User") {
		t.Fatalf("missing first bare User object")
	}
	if !containsObjectName(objects, QualifiedNameOf(reflect.TypeOf(testb.User{}))) {
		t.Fatalf("missing fallback qualified object for testtypes/b.User")
	}
}

func TestRegistryPreferredNameCollisionFallsBack(t *testing.T) {
	registry := NewRegistry(nil)
	registry.PreferName(testa.User{}, "User")
	registry.PreferName(testb.User{}, "User")
	registry.AddType(testa.User{})
	registry.AddType(testb.User{})

	objects := registry.Objects()
	if len(objects) != 2 {
		t.Fatalf("objects = %d, want 2", len(objects))
	}
	if !containsObjectName(objects, "User") {
		t.Fatalf("missing preferred User object")
	}
	if !containsObjectName(objects, QualifiedNameOf(reflect.TypeOf(testb.User{}))) {
		t.Fatalf("missing fallback qualified object for preferred collision")
	}
}

func containsObjectName(objects []Object, name string) bool {
	for _, obj := range objects {
		if obj.Name == name {
			return true
		}
	}
	return false
}
