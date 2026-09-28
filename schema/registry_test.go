package schema

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	testa "github.com/swetjen/virtuous/internal/testtypes/a"
	testb "github.com/swetjen/virtuous/internal/testtypes/b"
)

// registryNullableMatrix covers the stdlib nullable/optional shapes; the
// pgtype rows of this matrix live in the pgtypetest module.
type registryNullableMatrix struct {
	Plain       string          `json:"plain"`
	PlainPtr    *string         `json:"plain_ptr,omitempty"`
	Raw         json.RawMessage `json:"raw"`
	OptionalRaw json.RawMessage `json:"optional_raw,omitempty"`
}

type registryOverridePayload struct {
	When time.Time `json:"when"`
}

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

// TestRegistryUserOverrideBeatsBuiltInOverride pins that a user-supplied
// override for a type with a built-in mapping (time.Time -> string/datetime)
// replaces the built-in rather than merging with it.
func TestRegistryUserOverrideBeatsBuiltInOverride(t *testing.T) {
	registry := NewRegistry(map[string]TypeOverride{
		"time.Time": {
			JSType:        "CustomTime",
			PyType:        "CustomTimePy",
			OpenAPIType:   "integer",
			OpenAPIFormat: "int64",
		},
	})
	registry.AddType(registryOverridePayload{})

	object := findObject(registry.ObjectsWith(registry.JSTypeOf), "registryOverridePayload")
	if object == nil {
		t.Fatalf("missing registryOverridePayload object")
	}
	field := findField(object.Fields, "when")
	if field == nil {
		t.Fatalf("missing when field in %#v", object.Fields)
	}
	if field.Type != "CustomTime" || field.Nullable {
		t.Fatalf("when field = %#v, want custom non-null JS override", *field)
	}
	if got := registry.PyTypeOf(reflect.TypeOf(time.Time{})); got != "CustomTimePy" {
		t.Fatalf("PyTypeOf(time.Time) = %q, want CustomTimePy", got)
	}
	if containsObjectName(registry.Objects(), "Time") {
		t.Fatalf("time.Time should stay scalar and not emit implementation object")
	}
}

func TestRegistryNullableSemanticsMatrix(t *testing.T) {
	registry := NewRegistry(nil)
	registry.AddType(registryNullableMatrix{})

	object := findObject(registry.ObjectsWith(registry.PyTypeOf), "registryNullableMatrix")
	if object == nil {
		t.Fatalf("missing registryNullableMatrix object")
	}
	assertRegistryField(t, object, "plain", "str", false, false)
	assertRegistryField(t, object, "plain_ptr", "str", true, true)
	assertRegistryField(t, object, "raw", "Any", false, false)
	assertRegistryField(t, object, "optional_raw", "Any", true, false)
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

func findField(fields []Field, name string) *Field {
	for i := range fields {
		if fields[i].Name == name {
			return &fields[i]
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

func assertRegistryField(t *testing.T, object *Object, name, typ string, optional, nullable bool) {
	t.Helper()
	field := findField(object.Fields, name)
	if field == nil {
		t.Fatalf("missing field %q in %#v", name, object.Fields)
	}
	if field.Type != typ || field.Optional != optional || field.Nullable != nullable {
		t.Fatalf("field %q = %#v, want type=%q optional=%v nullable=%v", name, *field, typ, optional, nullable)
	}
}
