package schema

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	testa "github.com/swetjen/virtuous/internal/testtypes/a"
	testb "github.com/swetjen/virtuous/internal/testtypes/b"
)

type taggedSchema struct {
	Enabled bool    `json:"enabled" doc:"Feature flag" default:"true" example:"false"`
	Limit   int     `json:"limit" default:"20" minimum:"1" maximum:"100"`
	Ratio   float64 `json:"ratio" example:"0.5"`
	Since   string  `json:"since" format:"date"`
	Sort    string  `json:"sort" enum:"name,created_at"`
	Level   int     `json:"level" enum:"1,2,3"`
	DryRun  bool    `json:"dryRun" enum:"true,false"`
}

type childSchema struct {
	Name string `json:"name"`
}

type refTaggedSchema struct {
	Child childSchema `json:"child" doc:"Nested child"`
}

type EmbeddedOrganization struct {
	ID   string `json:"id"`
	UID  string `json:"uid"`
	Name string `json:"name"`
}

type EmbeddedBrand struct {
	LogoURL string `json:"logo_url"`
}

type embeddedDetailResponse struct {
	EmbeddedOrganization
	Branding *EmbeddedBrand `json:"branding"`
	Error    string         `json:"error"`
}

type embeddedTaggedResponse struct {
	EmbeddedOrganization `json:"organization"`
	Error                string `json:"error"`
}

type embeddedPointerResponse struct {
	*EmbeddedOrganization
	Error string `json:"error"`
}

type openAPIPgNullableMatrix struct {
	Plain       string          `json:"plain"`
	PlainPtr    *string         `json:"plain_ptr,omitempty"`
	Text        pgtype.Text     `json:"text"`
	TextPtr     *pgtype.Text    `json:"text_ptr,omitempty"`
	Raw         json.RawMessage `json:"raw"`
	OptionalRaw json.RawMessage `json:"optional_raw,omitempty"`
}

type openAPIPgOverridePayload struct {
	Text pgtype.Text `json:"text"`
}

type openAPIRawMessageShapes struct {
	Object json.RawMessage `json:"object"`
	Array  json.RawMessage `json:"array"`
	String json.RawMessage `json:"string"`
	Number json.RawMessage `json:"number"`
	Bool   json.RawMessage `json:"bool"`
	Null   json.RawMessage `json:"null"`
}

func TestOpenAPISchemaFieldMetadataTags(t *testing.T) {
	gen := NewGenerator(nil)
	schema := gen.SchemaFor(taggedSchema{})
	if schema == nil {
		t.Fatalf("schema is nil")
	}
	component := gen.Components()["taggedSchema"]
	props := component.Properties

	enabled := props["enabled"]
	if enabled.Description != "Feature flag" || enabled.Default != true || enabled.Example != false {
		t.Fatalf("enabled metadata = %#v", enabled)
	}

	limit := props["limit"]
	if limit.Default != int64(20) {
		t.Fatalf("limit default = %#v, want int64(20)", limit.Default)
	}
	if limit.Minimum == nil || *limit.Minimum != 1 {
		t.Fatalf("limit minimum = %#v, want 1", limit.Minimum)
	}
	if limit.Maximum == nil || *limit.Maximum != 100 {
		t.Fatalf("limit maximum = %#v, want 100", limit.Maximum)
	}

	ratio := props["ratio"]
	if ratio.Example != 0.5 {
		t.Fatalf("ratio example = %#v, want 0.5", ratio.Example)
	}

	since := props["since"]
	if since.Format != "date" {
		t.Fatalf("since format = %q, want date", since.Format)
	}

	sort := props["sort"]
	if len(sort.Enum) != 2 || sort.Enum[0] != "name" || sort.Enum[1] != "created_at" {
		t.Fatalf("sort enum = %#v, want name/created_at", sort.Enum)
	}

	level := props["level"]
	if len(level.Enum) != 3 || level.Enum[0] != int64(1) || level.Enum[1] != int64(2) || level.Enum[2] != int64(3) {
		t.Fatalf("level enum = %#v, want int64 1/2/3", level.Enum)
	}

	dryRun := props["dryRun"]
	if len(dryRun.Enum) != 2 || dryRun.Enum[0] != true || dryRun.Enum[1] != false {
		t.Fatalf("dryRun enum = %#v, want true/false", dryRun.Enum)
	}
}

func TestOpenAPISchemaFieldMetadataWrapsRefs(t *testing.T) {
	gen := NewGenerator(nil)
	_ = gen.SchemaFor(refTaggedSchema{})
	component := gen.Components()["refTaggedSchema"]
	child := component.Properties["child"]

	if child.Description != "Nested child" {
		t.Fatalf("child description = %q, want Nested child", child.Description)
	}
	if len(child.AllOf) != 1 || child.AllOf[0].Ref == "" {
		t.Fatalf("child should wrap ref in allOf: %#v", child)
	}
}

func TestOpenAPIGeneratorSchemaNameCollisionFallsBack(t *testing.T) {
	gen := NewGenerator(nil)
	_ = gen.SchemaFor(testa.User{})
	_ = gen.SchemaFor(testb.User{})

	components := gen.Components()
	if len(components) != 2 {
		t.Fatalf("components = %d, want 2", len(components))
	}
	if _, ok := components["User"]; !ok {
		t.Fatalf("missing first bare User schema")
	}
	if _, ok := components[QualifiedNameOf(reflect.TypeOf(testb.User{}))]; !ok {
		t.Fatalf("missing fallback qualified schema for testtypes/b.User")
	}
}

func TestOpenAPIGeneratorPreferredNameCollisionFallsBack(t *testing.T) {
	gen := NewGenerator(nil)
	gen.PreferName(testa.User{}, "User")
	gen.PreferName(testb.User{}, "User")
	_ = gen.SchemaFor(testa.User{})
	_ = gen.SchemaFor(testb.User{})

	components := gen.Components()
	if len(components) != 2 {
		t.Fatalf("components = %d, want 2", len(components))
	}
	if _, ok := components["User"]; !ok {
		t.Fatalf("missing preferred User schema")
	}
	if _, ok := components[QualifiedNameOf(reflect.TypeOf(testb.User{}))]; !ok {
		t.Fatalf("missing fallback qualified schema for preferred collision")
	}
}

func TestOpenAPISchemaFlattensAnonymousEmbeddedStruct(t *testing.T) {
	gen := NewGenerator(nil)
	_ = gen.SchemaFor(embeddedDetailResponse{})

	component := gen.Components()["embeddedDetailResponse"]
	props := component.Properties
	for _, name := range []string{"id", "uid", "name", "branding", "error"} {
		if _, ok := props[name]; !ok {
			t.Fatalf("missing flattened property %q in %#v", name, props)
		}
	}
	if _, ok := props["embeddedOrganization"]; ok {
		t.Fatalf("anonymous embedded struct should not be emitted as nested property")
	}
	for _, name := range []string{"id", "uid", "name", "error"} {
		if !containsString(component.Required, name) {
			t.Fatalf("required = %#v, want %q", component.Required, name)
		}
	}
}

func TestOpenAPISchemaKeepsExplicitlyTaggedEmbeddedStructNested(t *testing.T) {
	gen := NewGenerator(nil)
	_ = gen.SchemaFor(embeddedTaggedResponse{})

	component := gen.Components()["embeddedTaggedResponse"]
	props := component.Properties
	if _, ok := props["organization"]; !ok {
		t.Fatalf("missing explicitly tagged embedded property in %#v", props)
	}
	if _, ok := props["id"]; ok {
		t.Fatalf("explicitly tagged embedded struct should not be flattened")
	}
}

func TestOpenAPISchemaMakesPromotedPointerFieldsOptional(t *testing.T) {
	gen := NewGenerator(nil)
	_ = gen.SchemaFor(embeddedPointerResponse{})

	component := gen.Components()["embeddedPointerResponse"]
	props := component.Properties
	if _, ok := props["id"]; !ok {
		t.Fatalf("missing promoted pointer property id in %#v", props)
	}
	if containsString(component.Required, "id") || containsString(component.Required, "uid") || containsString(component.Required, "name") {
		t.Fatalf("promoted pointer fields should not be required: %#v", component.Required)
	}
	if !containsString(component.Required, "error") {
		t.Fatalf("required = %#v, want error", component.Required)
	}
}

func TestOpenAPIUserOverrideBeatsBuiltInPgtypeOverride(t *testing.T) {
	gen := NewGenerator(map[string]TypeOverride{
		"github.com/jackc/pgx/v5/pgtype.Text": {
			JSType:        "CustomText",
			PyType:        "CustomTextPy",
			OpenAPIType:   "integer",
			OpenAPIFormat: "int32",
		},
	})
	_ = gen.SchemaFor(openAPIPgOverridePayload{})

	component := gen.Components()["openAPIPgOverridePayload"]
	text := component.Properties["text"]
	if text.Type != "integer" || text.Format != "int32" || text.Nullable {
		t.Fatalf("text schema = %#v, want custom non-null integer override", text)
	}
	if _, ok := gen.Components()["Text"]; ok {
		t.Fatalf("pgtype.Text should stay scalar and not emit implementation schema")
	}
}

func TestOpenAPINullableSemanticsMatrix(t *testing.T) {
	gen := NewGenerator(nil)
	_ = gen.SchemaFor(openAPIPgNullableMatrix{})

	component := gen.Components()["openAPIPgNullableMatrix"]
	assertOpenAPIField(t, component, "plain", "string", "", false, true)
	assertOpenAPIField(t, component, "plain_ptr", "string", "", true, false)
	assertOpenAPIField(t, component, "text", "string", "", true, true)
	assertOpenAPIField(t, component, "text_ptr", "string", "", true, false)
	assertOpenAPIField(t, component, "raw", "", "", false, true)
	assertOpenAPIField(t, component, "optional_raw", "", "", false, false)
}

func TestOpenAPIRawMessageIsArbitraryJSONForScalarShapes(t *testing.T) {
	gen := NewGenerator(nil)
	_ = gen.SchemaFor(openAPIRawMessageShapes{})

	component := gen.Components()["openAPIRawMessageShapes"]
	for _, name := range []string{"object", "array", "string", "number", "bool", "null"} {
		prop := component.Properties[name]
		if prop == nil {
			t.Fatalf("missing raw property %q", name)
		}
		if prop.Type != "" || prop.Format != "" || prop.Items != nil || prop.AdditionalProperties != nil || prop.Ref != "" {
			t.Fatalf("raw property %q should be arbitrary JSON schema: %#v", name, prop)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func assertOpenAPIField(t *testing.T, component OpenAPISchema, name, typ, format string, nullable, required bool) {
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
