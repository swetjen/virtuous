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

func TestRegistryFlattensAnonymousEmbeddedStruct(t *testing.T) {
	registry := NewRegistry(nil)
	registry.AddType(embeddedDetailResponse{})

	object := findObject(registry.Objects(), "embeddedDetailResponse")
	if object == nil {
		t.Fatalf("missing embeddedDetailResponse object")
	}
	for _, name := range []string{"id", "uid", "name", "branding", "error"} {
		if !containsFieldName(object.Fields, name) {
			t.Fatalf("missing flattened field %q in %#v", name, object.Fields)
		}
	}
	if containsFieldName(object.Fields, "embeddedOrganization") {
		t.Fatalf("anonymous embedded struct should not be emitted as nested field")
	}
}

func TestRegistryKeepsExplicitlyTaggedEmbeddedStructNested(t *testing.T) {
	registry := NewRegistry(nil)
	registry.AddType(embeddedTaggedResponse{})

	object := findObject(registry.Objects(), "embeddedTaggedResponse")
	if object == nil {
		t.Fatalf("missing embeddedTaggedResponse object")
	}
	if !containsFieldName(object.Fields, "organization") {
		t.Fatalf("missing explicitly tagged embedded field in %#v", object.Fields)
	}
	if containsFieldName(object.Fields, "id") {
		t.Fatalf("explicitly tagged embedded struct should not be flattened")
	}
}

func TestRegistryMakesPromotedPointerFieldsOptional(t *testing.T) {
	registry := NewRegistry(nil)
	registry.AddType(embeddedPointerResponse{})

	object := findObject(registry.Objects(), "embeddedPointerResponse")
	if object == nil {
		t.Fatalf("missing embeddedPointerResponse object")
	}
	for _, field := range object.Fields {
		switch field.Name {
		case "id", "uid", "name":
			if !field.Optional {
				t.Fatalf("promoted pointer field %q should be optional", field.Name)
			}
		case "error":
			if field.Optional {
				t.Fatalf("direct field error should not be optional")
			}
		}
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

func findObject(objects []Object, name string) *Object {
	for i := range objects {
		if objects[i].Name == name {
			return &objects[i]
		}
	}
	return nil
}

func containsFieldName(fields []Field, name string) bool {
	for _, field := range fields {
		if field.Name == name {
			return true
		}
	}
	return false
}
