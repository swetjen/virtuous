package pgtypetest

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgtype/zeronull"
	"github.com/swetjen/virtuous/schema"
)

// pgNullableMatrix contrasts a pgtype wrapper (nullable, required) with the
// stdlib shapes it must line up against.
type pgNullableMatrix struct {
	Plain    string          `json:"plain"`
	PlainPtr *string         `json:"plain_ptr,omitempty"`
	Text     pgtype.Text     `json:"text"`
	TextPtr  *pgtype.Text    `json:"text_ptr,omitempty"`
	Raw      json.RawMessage `json:"raw"`
}

type pgOverridePayload struct {
	Text pgtype.Text `json:"text"`
}

// pgtypeTextOverride replaces the built-in pgtype.Text mapping with a
// deliberately different, non-nullable one.
var pgtypeTextOverride = map[string]schema.TypeOverride{
	"github.com/jackc/pgx/v5/pgtype.Text": {
		JSType:        "CustomText",
		PyType:        "CustomTextPy",
		OpenAPIType:   "integer",
		OpenAPIFormat: "int32",
	},
}

func TestOpenAPIUserOverrideBeatsBuiltInPgtypeOverride(t *testing.T) {
	gen := schema.NewGenerator(pgtypeTextOverride)
	_ = gen.SchemaFor(pgOverridePayload{})

	component := gen.Components()["pgOverridePayload"]
	text := component.Properties["text"]
	if text == nil || text.Type != "integer" || text.Format != "int32" || text.Nullable {
		t.Fatalf("text schema = %#v, want custom non-null integer override", text)
	}
	if _, ok := gen.Components()["Text"]; ok {
		t.Fatalf("pgtype.Text should stay scalar and not emit implementation schema")
	}
}

func TestOpenAPIPgtypeNullableSemantics(t *testing.T) {
	gen := schema.NewGenerator(nil)
	_ = gen.SchemaFor(pgNullableMatrix{})

	component := gen.Components()["pgNullableMatrix"]
	assertOpenAPIField(t, component, "plain", "string", "", false, true)
	assertOpenAPIField(t, component, "plain_ptr", "string", "", true, false)
	assertOpenAPIField(t, component, "text", "string", "", true, true)
	assertOpenAPIField(t, component, "text_ptr", "string", "", true, false)
	assertOpenAPIField(t, component, "raw", "", "", false, true)
	if _, ok := gen.Components()["Text"]; ok {
		t.Fatalf("pgtype.Text should not emit an implementation schema")
	}
}

func TestRegistryUserOverrideBeatsBuiltInPgtypeOverride(t *testing.T) {
	registry := schema.NewRegistry(pgtypeTextOverride)
	registry.AddType(pgOverridePayload{})

	object := findObject(registry.ObjectsWith(registry.JSTypeOf), "pgOverridePayload")
	if object == nil {
		t.Fatalf("missing pgOverridePayload object")
	}
	field := findField(object.Fields, "text")
	if field == nil {
		t.Fatalf("missing text field in %#v", object.Fields)
	}
	if field.Type != "CustomText" || field.Nullable {
		t.Fatalf("text field = %#v, want custom non-null JS override", *field)
	}
	if got := registry.PyTypeOf(reflect.TypeOf(pgtype.Text{})); got != "CustomTextPy" {
		t.Fatalf("PyTypeOf(pgtype.Text) = %q, want CustomTextPy", got)
	}
	if findObject(registry.Objects(), "Text") != nil {
		t.Fatalf("pgtype.Text should stay scalar and not emit implementation object")
	}
}

func TestRegistryPgtypeNullableSemantics(t *testing.T) {
	registry := schema.NewRegistry(nil)
	registry.AddType(pgNullableMatrix{})

	object := findObject(registry.ObjectsWith(registry.PyTypeOf), "pgNullableMatrix")
	if object == nil {
		t.Fatalf("missing pgNullableMatrix object")
	}
	assertRegistryField(t, object, "plain", "str", false, false)
	assertRegistryField(t, object, "plain_ptr", "str", true, true)
	assertRegistryField(t, object, "text", "str", false, true)
	assertRegistryField(t, object, "text_ptr", "str", true, true)
	assertRegistryField(t, object, "raw", "Any", false, false)
	if findObject(registry.Objects(), "Text") != nil {
		t.Fatalf("pgtype.Text should not emit an implementation object")
	}
}

// TestPgtypeUnsupportedFamiliesAreNotBuiltInScalars pins the families that
// docs/internals/type-registry.md lists as intentionally unmodeled: arrays,
// ranges, multiranges, interval/time/geometric wrappers, Uint64, and the
// zeronull timestamp/UUID wrappers. Without a user override they must fall
// through to ordinary reflection (an object or an array), never collapse to a
// built-in scalar.
func TestPgtypeUnsupportedFamiliesAreNotBuiltInScalars(t *testing.T) {
	registry := schema.NewRegistry(nil)
	generator := schema.NewGenerator(nil)
	unsupported := []reflect.Type{
		reflect.TypeOf(pgtype.Uint64{}),
		reflect.TypeOf(pgtype.Array[pgtype.Int4]{}),
		reflect.TypeOf(pgtype.FlatArray[pgtype.Int4]{}),
		reflect.TypeOf(pgtype.Range[int32]{}),
		reflect.TypeOf(pgtype.Multirange[pgtype.Range[int32]]{}),
		reflect.TypeOf(pgtype.Interval{}),
		reflect.TypeOf(pgtype.Time{}),
		reflect.TypeOf(pgtype.Point{}),
		reflect.TypeOf(pgtype.Line{}),
		reflect.TypeOf(pgtype.Box{}),
		reflect.TypeOf(zeronull.Timestamp{}),
		reflect.TypeOf(zeronull.Timestamptz{}),
		reflect.TypeOf(zeronull.UUID{}),
	}
	// Every JS/Python type a built-in override can produce.
	builtInJS := map[string]bool{"string": true, "number": true, "boolean": true, "object|any[]": true}
	builtInPy := map[string]bool{"str": true, "int": true, "float": true, "bool": true, "Any": true, "date": true, "datetime": true}

	for _, typ := range unsupported {
		registry.AddTypeOf(typ)
		if js := registry.JSTypeOf(typ); builtInJS[js] {
			t.Fatalf("%s JS type = %q; should not be a built-in client scalar", typ, js)
		}
		if py := registry.PyTypeOf(typ); builtInPy[py] {
			t.Fatalf("%s Python type = %q; should not be a built-in client scalar", typ, py)
		}
		got := generator.SchemaForType(typ)
		if got == nil {
			t.Fatalf("%s produced no OpenAPI schema", typ)
		}
		if got.Ref == "" && got.Type != "array" {
			t.Fatalf("%s OpenAPI schema = %#v; want object $ref or array, not a built-in scalar", typ, got)
		}
	}
}

func findObject(objects []schema.Object, name string) *schema.Object {
	for i := range objects {
		if objects[i].Name == name {
			return &objects[i]
		}
	}
	return nil
}

func findField(fields []schema.Field, name string) *schema.Field {
	for i := range fields {
		if fields[i].Name == name {
			return &fields[i]
		}
	}
	return nil
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func assertOpenAPIField(t *testing.T, component schema.OpenAPISchema, name, typ, format string, nullable, required bool) {
	t.Helper()
	prop := component.Properties[name]
	if prop == nil {
		t.Fatalf("missing property %q in %#v", name, component.Properties)
	}
	if prop.Type != typ || prop.Format != format || prop.Nullable != nullable {
		t.Fatalf("property %q = %#v, want type=%q format=%q nullable=%v", name, prop, typ, format, nullable)
	}
	if containsString(component.Required, name) != required {
		t.Fatalf("property %q required=%v, want %v in %#v", name, containsString(component.Required, name), required, component.Required)
	}
}

func assertRegistryField(t *testing.T, object *schema.Object, name, typ string, optional, nullable bool) {
	t.Helper()
	field := findField(object.Fields, name)
	if field == nil {
		t.Fatalf("missing field %q in %#v", name, object.Fields)
	}
	if field.Type != typ || field.Optional != optional || field.Nullable != nullable {
		t.Fatalf("field %q = %#v, want type=%q optional=%v nullable=%v", name, *field, typ, optional, nullable)
	}
}
