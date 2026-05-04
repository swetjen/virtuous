package schema

import (
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/swetjen/virtuous/internal/reflectutil"
)

// OpenAPISchema describes a JSON schema object used in OpenAPI documents.
type OpenAPISchema struct {
	Ref                  string                    `json:"$ref,omitempty"`
	Type                 string                    `json:"type,omitempty"`
	Format               string                    `json:"format,omitempty"`
	Nullable             bool                      `json:"nullable,omitempty"`
	Description          string                    `json:"description,omitempty"`
	Default              any                       `json:"default,omitempty"`
	Example              any                       `json:"example,omitempty"`
	Minimum              *float64                  `json:"minimum,omitempty"`
	Maximum              *float64                  `json:"maximum,omitempty"`
	Properties           map[string]*OpenAPISchema `json:"properties,omitempty"`
	Items                *OpenAPISchema            `json:"items,omitempty"`
	AdditionalProperties *OpenAPISchema            `json:"additionalProperties,omitempty"`
	Required             []string                  `json:"required,omitempty"`
	AllOf                []*OpenAPISchema          `json:"allOf,omitempty"`
}

// Generator builds OpenAPI schemas from Go types.
type Generator struct {
	overrides  map[string]TypeOverride
	components map[string]OpenAPISchema
	seen       map[reflect.Type]string
	seenNames  map[string]reflect.Type
	preferred  map[reflect.Type]string
}

// NewGenerator returns an OpenAPI schema generator with overrides applied.
func NewGenerator(overrides map[string]TypeOverride) *Generator {
	return &Generator{
		overrides:  mergeTypeOverrides(overrides),
		components: map[string]OpenAPISchema{},
		seen:       map[reflect.Type]string{},
		seenNames:  map[string]reflect.Type{},
		preferred:  map[reflect.Type]string{},
	}
}

// Components returns the OpenAPI component schema map.
func (g *Generator) Components() map[string]OpenAPISchema {
	return g.components
}

// PreferName hints a schema name for the provided Go type.
func (g *Generator) PreferName(v any, name string) {
	g.preferNameOf(reflect.TypeOf(v), name)
}

// PreferNameOf hints a schema name for the provided Go type.
func (g *Generator) PreferNameOf(t reflect.Type, name string) {
	g.preferNameOf(t, name)
}

func (g *Generator) preferNameOf(t reflect.Type, name string) {
	t = reflectutil.DerefType(t)
	if t == nil || name == "" {
		return
	}
	g.preferred[t] = name
}

// SchemaFor builds an OpenAPI schema for the provided value.
func (g *Generator) SchemaFor(v any) *OpenAPISchema {
	return g.schemaFor(reflect.TypeOf(v))
}

// SchemaForType builds an OpenAPI schema for the provided type.
func (g *Generator) SchemaForType(t reflect.Type) *OpenAPISchema {
	return g.schemaFor(t)
}

func (g *Generator) schemaFor(t reflect.Type) *OpenAPISchema {
	if t == nil {
		return nil
	}
	nullable := false
	for t.Kind() == reflect.Ptr {
		nullable = true
		t = t.Elem()
	}
	if t == nil {
		return nil
	}
	if name, ok := g.seen[t]; ok {
		schema := &OpenAPISchema{Ref: "#/components/schemas/" + name}
		if nullable {
			schema.Nullable = true
		}
		return schema
	}

	if g.isOverrideScalar(t) {
		schema := g.overrideSchema(t)
		if nullable && schema != nil {
			schema.Nullable = true
		}
		return schema
	}
	if isTimeType(t) {
		schema := &OpenAPISchema{Type: "string", Format: "date-time"}
		if nullable {
			schema.Nullable = true
		}
		return schema
	}
	if t.Kind() == reflect.Struct && t.Name() != "" {
		name := g.schemaNameFor(t)
		g.seen[t] = name
		g.components[name] = OpenAPISchema{}
		schema := g.structSchema(t)
		g.components[name] = *schema
		refSchema := &OpenAPISchema{Ref: "#/components/schemas/" + name}
		if nullable {
			refSchema.Nullable = true
		}
		return refSchema
	}

	schema := g.inlineSchema(t)
	if nullable && schema != nil {
		schema.Nullable = true
	}
	return schema
}

func (g *Generator) isOverrideScalar(t reflect.Type) bool {
	override, ok := typeOverrideFor(g.overrides, t)
	if !ok {
		return false
	}
	return override.OpenAPIType != "" || override.OpenAPIFormat != ""
}

func (g *Generator) overrideSchema(t reflect.Type) *OpenAPISchema {
	override, ok := typeOverrideFor(g.overrides, t)
	if !ok {
		return nil
	}
	schema := &OpenAPISchema{}
	if override.OpenAPIType != "" {
		schema.Type = override.OpenAPIType
	} else if override.OpenAPIFormat != "" {
		schema.Type = "string"
	}
	if override.OpenAPIFormat != "" {
		schema.Format = override.OpenAPIFormat
	}
	return schema
}

func isTimeType(t reflect.Type) bool {
	return t.PkgPath() == "time" && t.Name() == "Time"
}

func (g *Generator) structSchema(t reflect.Type) *OpenAPISchema {
	props := map[string]*OpenAPISchema{}
	var required []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		name, omit := reflectutil.JSONFieldName(field)
		if name == "" {
			continue
		}
		schema := g.schemaFor(field.Type)
		if schema == nil {
			continue
		}
		doc := reflectutil.FieldDoc(field)
		schema = ApplyFieldMetadata(field, schema)
		if doc != "" && schema.Description == "" {
			schema.Description = doc
		}
		props[name] = schema
		if !omit && field.Type.Kind() != reflect.Ptr {
			required = append(required, name)
		}
	}
	sortStrings(required)
	return &OpenAPISchema{
		Type:       "object",
		Properties: props,
		Required:   required,
	}
}

func (g *Generator) inlineSchema(t reflect.Type) *OpenAPISchema {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if g.isOverrideScalar(t) {
		return g.overrideSchema(t)
	}
	switch t.Kind() {
	case reflect.String:
		return &OpenAPISchema{Type: "string"}
	case reflect.Bool:
		return &OpenAPISchema{Type: "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return &OpenAPISchema{Type: "integer", Format: "int32"}
	case reflect.Int64:
		return &OpenAPISchema{Type: "integer", Format: "int64"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &OpenAPISchema{Type: "integer"}
	case reflect.Float32:
		return &OpenAPISchema{Type: "number", Format: "float"}
	case reflect.Float64:
		return &OpenAPISchema{Type: "number", Format: "double"}
	case reflect.Slice, reflect.Array:
		return &OpenAPISchema{
			Type:  "array",
			Items: g.schemaFor(t.Elem()),
		}
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return &OpenAPISchema{Type: "object"}
		}
		return &OpenAPISchema{
			Type:                 "object",
			AdditionalProperties: g.schemaFor(t.Elem()),
		}
	case reflect.Struct:
		if isTimeType(t) {
			return &OpenAPISchema{Type: "string", Format: "date-time"}
		}
		return g.structSchema(t)
	default:
		return &OpenAPISchema{Type: "string"}
	}
}

// ApplyFieldMetadata copies OpenAPI-oriented struct tags onto a schema.
func ApplyFieldMetadata(field reflect.StructField, fieldSchema *OpenAPISchema) *OpenAPISchema {
	if fieldSchema == nil {
		return nil
	}
	schema := fieldSchema
	nullable := schema.Nullable
	if needsSchemaWrapper(schema, field) {
		wrapped := &OpenAPISchema{
			Description: schema.Description,
			Nullable:    nullable,
			Default:     schema.Default,
			Example:     schema.Example,
			Minimum:     schema.Minimum,
			Maximum:     schema.Maximum,
		}
		if schema.Ref != "" {
			wrapped.AllOf = []*OpenAPISchema{{Ref: schema.Ref}}
		} else {
			copy := *schema
			wrapped.AllOf = []*OpenAPISchema{&copy}
		}
		schema = wrapped
	}
	if doc := reflectutil.FieldDoc(field); doc != "" {
		schema.Description = doc
	}
	if format := strings.TrimSpace(field.Tag.Get("format")); format != "" {
		schema.Format = format
	}
	if value, ok := parseSchemaTagValue(field.Tag.Get("default"), field.Type); ok {
		schema.Default = value
	}
	if value, ok := parseSchemaTagValue(field.Tag.Get("example"), field.Type); ok {
		schema.Example = value
	}
	if minimum, ok := parseFloatTag(field.Tag.Get("minimum")); ok {
		schema.Minimum = &minimum
	}
	if maximum, ok := parseFloatTag(field.Tag.Get("maximum")); ok {
		schema.Maximum = &maximum
	}
	if nullable {
		schema.Nullable = true
	}
	return schema
}

func needsSchemaWrapper(schema *OpenAPISchema, field reflect.StructField) bool {
	if schema.Ref == "" {
		return false
	}
	return reflectutil.FieldDoc(field) != "" ||
		strings.TrimSpace(field.Tag.Get("format")) != "" ||
		strings.TrimSpace(field.Tag.Get("default")) != "" ||
		strings.TrimSpace(field.Tag.Get("example")) != "" ||
		strings.TrimSpace(field.Tag.Get("minimum")) != "" ||
		strings.TrimSpace(field.Tag.Get("maximum")) != ""
}

func parseSchemaTagValue(raw string, t reflect.Type) (any, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, false
	}
	base := reflectutil.DerefType(t)
	if base == nil {
		return raw, true
	}
	switch base.Kind() {
	case reflect.Bool:
		value, err := strconv.ParseBool(raw)
		if err == nil {
			return value, true
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, err := strconv.ParseInt(raw, 10, 64)
		if err == nil {
			return value, true
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value, err := strconv.ParseUint(raw, 10, 64)
		if err == nil {
			return value, true
		}
	case reflect.Float32, reflect.Float64:
		value, err := strconv.ParseFloat(raw, 64)
		if err == nil {
			return value, true
		}
	}
	return raw, true
}

func parseFloatTag(raw string) (float64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	value, err := strconv.ParseFloat(raw, 64)
	return value, err == nil
}

func schemaName(t reflect.Type) string {
	if t.PkgPath() == "" {
		return t.Name()
	}
	name := strings.ReplaceAll(t.PkgPath(), "/", "_") + "_" + t.Name()
	name = strings.ReplaceAll(name, ".", "_")
	return name
}

func schemaNameOrPanic(seen map[string]reflect.Type, t reflect.Type) string {
	name := t.Name()
	if name == "" {
		name = schemaName(t)
	}
	if other, ok := seen[name]; ok && other != t {
		panic("virtuous: schema name collision for " + name)
	}
	seen[name] = t
	return name
}

func (g *Generator) schemaNameFor(t reflect.Type) string {
	if preferred, ok := g.preferred[t]; ok {
		if other, ok := g.seenNames[preferred]; ok && other != t {
			panic("virtuous: schema name collision for " + preferred)
		}
		g.seenNames[preferred] = t
		return preferred
	}
	return schemaNameOrPanic(g.seenNames, t)
}

func sortStrings(values []string) {
	if len(values) < 2 {
		return
	}
	sort.Strings(values)
}
