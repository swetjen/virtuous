package schema

import (
	"reflect"
	"sort"
	"strings"

	"github.com/swetjen/virtuous/internal/reflectutil"
)

// TypeOverride customizes how a Go type is rendered for clients and OpenAPI.
type TypeOverride struct {
	JSType        string
	PyType        string
	OpenAPIType   string
	OpenAPIFormat string
}

// Field describes a schema field for client generation.
type Field struct {
	Name     string
	Type     string
	Optional bool
	Nullable bool
	Doc      string
}

// Object describes a named schema object for client generation.
type Object struct {
	Name   string
	Fields []Field
}

// Registry captures reflected types and renders language-specific types.
type Registry struct {
	overrides  map[string]TypeOverride
	objects    map[reflect.Type]*objectDef
	nameByType map[reflect.Type]string
	typeByName map[string]reflect.Type
	preferred  map[reflect.Type]string
}

type objectDef struct {
	Name   string
	Fields []fieldDef
}

type fieldDef struct {
	Name     string
	Type     reflect.Type
	Optional bool
	Nullable bool
	Doc      string
}

// NewRegistry returns a registry with overrides applied.
func NewRegistry(overrides map[string]TypeOverride) *Registry {
	return &Registry{
		overrides:  mergeTypeOverrides(overrides),
		objects:    map[reflect.Type]*objectDef{},
		nameByType: map[reflect.Type]string{},
		typeByName: map[string]reflect.Type{},
		preferred:  map[reflect.Type]string{},
	}
}

// AddType registers a Go value for schema generation.
func (r *Registry) AddType(v any) {
	r.addType(reflect.TypeOf(v))
}

// AddTypeOf registers a Go type for schema generation.
func (r *Registry) AddTypeOf(t reflect.Type) {
	r.addType(t)
}

// PreferName hints a schema name for a value type.
func (r *Registry) PreferName(v any, name string) {
	r.preferName(reflect.TypeOf(v), name)
}

// PreferNameOf hints a schema name for a type.
func (r *Registry) PreferNameOf(t reflect.Type, name string) {
	r.preferName(t, name)
}

// ObjectsWith maps object fields using the provided type renderer.
func (r *Registry) ObjectsWith(typeFn func(reflect.Type) string) []Object {
	objects := make([]Object, 0, len(r.objects))
	for _, obj := range r.objects {
		clientObj := Object{Name: obj.Name}
		for _, field := range obj.Fields {
			fieldType := typeFn(field.Type)
			if fieldType == "" {
				fieldType = "any"
			}
			clientObj.Fields = append(clientObj.Fields, Field{
				Name:     field.Name,
				Type:     fieldType,
				Optional: field.Optional,
				Nullable: field.Nullable,
				Doc:      field.Doc,
			})
		}
		objects = append(objects, clientObj)
	}
	sort.Slice(objects, func(i, j int) bool {
		return objects[i].Name < objects[j].Name
	})
	return objects
}

// Objects returns the reflected object definitions for client generation.
// Defaults to JS-oriented types for backward compatibility.
func (r *Registry) Objects() []Object {
	return r.ObjectsWith(r.jsType)
}

// JSType renders the JavaScript type for a value.
func (r *Registry) JSType(v any) string {
	return r.jsType(reflect.TypeOf(v))
}

// JSTypeOf renders the JavaScript type for a Go type.
func (r *Registry) JSTypeOf(t reflect.Type) string {
	return r.jsType(t)
}

// PyType renders the Python type for a value.
func (r *Registry) PyType(v any) string {
	return r.pyType(reflect.TypeOf(v))
}

// PyTypeOf renders the Python type for a Go type.
func (r *Registry) PyTypeOf(t reflect.Type) string {
	return r.pyType(t)
}

func mergeTypeOverrides(user map[string]TypeOverride) map[string]TypeOverride {
	merged := map[string]TypeOverride{}
	for key, value := range defaultTypeOverrides() {
		merged[key] = value
	}
	for key, value := range user {
		merged[key] = value
	}
	return merged
}

func defaultTypeOverrides() map[string]TypeOverride {
	return map[string]TypeOverride{
		"time.Time": {
			JSType:        "string",
			PyType:        "datetime",
			OpenAPIType:   "string",
			OpenAPIFormat: "date-time",
		},
		"encoding/json.RawMessage": {
			JSType:      "any",
			PyType:      "Any",
			OpenAPIType: "object",
		},
	}
}

func (r *Registry) addType(t reflect.Type) {
	base := reflectutil.DerefType(t)
	if base == nil {
		return
	}
	if r.isOverrideScalar(base) {
		return
	}
	switch base.Kind() {
	case reflect.Struct:
		if base.Name() == "" {
			return
		}
		if _, ok := r.objects[base]; ok {
			return
		}
		name := r.objectName(base)
		obj := &objectDef{Name: name}
		r.objects[base] = obj
		for i := 0; i < base.NumField(); i++ {
			field := base.Field(i)
			if field.PkgPath != "" {
				continue
			}
			name, omit := reflectutil.JSONFieldName(field)
			if name == "" {
				continue
			}
			obj.Fields = append(obj.Fields, fieldDef{
				Name:     name,
				Type:     field.Type,
				Optional: omit,
				Nullable: isOptionalType(field.Type),
				Doc:      reflectutil.FieldDoc(field),
			})
			r.addType(field.Type)
		}
	case reflect.Slice, reflect.Array:
		r.addType(base.Elem())
	case reflect.Map:
		r.addType(base.Elem())
	}
}

func (r *Registry) objectName(t reflect.Type) string {
	if name, ok := r.nameByType[t]; ok {
		return name
	}
	if preferred, ok := r.preferred[t]; ok {
		if other, ok := r.typeByName[preferred]; ok && other != t {
			panic("virtuous: schema name collision for " + preferred)
		}
		r.nameByType[t] = preferred
		r.typeByName[preferred] = t
		return preferred
	}
	name := t.Name()
	if name == "" {
		name = schemaName(t)
	}
	if other, ok := r.typeByName[name]; ok && other != t {
		panic("virtuous: schema name collision for " + name)
	}
	r.nameByType[t] = name
	r.typeByName[name] = t
	return name
}

func (r *Registry) preferName(t reflect.Type, name string) {
	t = reflectutil.DerefType(t)
	if t == nil || name == "" {
		return
	}
	r.preferred[t] = name
}

func (r *Registry) jsType(t reflect.Type) string {
	base := reflectutil.DerefType(t)
	if base == nil {
		return ""
	}
	if override, ok := typeOverrideFor(r.overrides, base); ok && override.JSType != "" {
		return override.JSType
	}
	switch base.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Interface:
		return "any"
	case reflect.Slice, reflect.Array:
		elemType := r.jsType(base.Elem())
		if elemType == "" {
			elemType = "any"
		}
		return elemType + "[]"
	case reflect.Map:
		return "object"
	case reflect.Struct:
		if base.Name() == "" {
			return "object"
		}
		return r.objectName(base)
	default:
		return "any"
	}
}

func (r *Registry) pyType(t reflect.Type) string {
	base := reflectutil.DerefType(t)
	if base == nil {
		return ""
	}
	if override, ok := typeOverrideFor(r.overrides, base); ok && override.PyType != "" {
		return override.PyType
	}
	switch base.Kind() {
	case reflect.Bool:
		return "bool"
	case reflect.String:
		return "str"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "int"
	case reflect.Float32, reflect.Float64:
		return "float"
	case reflect.Interface:
		return "Any"
	case reflect.Slice, reflect.Array:
		elemType := r.pyType(base.Elem())
		if elemType == "" {
			elemType = "Any"
		}
		return "list[" + elemType + "]"
	case reflect.Map:
		valueType := r.pyType(base.Elem())
		if valueType == "" {
			valueType = "Any"
		}
		if base.Key().Kind() == reflect.String {
			return "dict[str, " + valueType + "]"
		}
		return "dict[Any, " + valueType + "]"
	case reflect.Struct:
		if base.Name() == "" {
			return "dict[str, Any]"
		}
		return quotePyType(r.objectName(base))
	default:
		return "Any"
	}
}

func quotePyType(name string) string {
	if name == "" {
		return ""
	}
	return "\"" + name + "\""
}

func (r *Registry) isOverrideScalar(t reflect.Type) bool {
	override, ok := typeOverrideFor(r.overrides, t)
	if !ok {
		return false
	}
	return override.OpenAPIType != "" || override.OpenAPIFormat != "" || override.JSType != "" || override.PyType != ""
}

func typeOverrideFor(overrides map[string]TypeOverride, t reflect.Type) (TypeOverride, bool) {
	if overrides == nil || t == nil {
		return TypeOverride{}, false
	}
	name := strings.TrimPrefix(t.String(), "*")
	if override, ok := overrides[name]; ok {
		return override, true
	}
	if t.Name() == "" {
		return TypeOverride{}, false
	}
	if override, ok := overrides[t.Name()]; ok {
		return override, true
	}
	full := t.PkgPath() + "." + t.Name()
	override, ok := overrides[full]
	return override, ok
}

// PreferredName derives a schema name from a service prefix and Go type.
func PreferredName(service string, v any) string {
	return PreferredNameOf(service, reflect.TypeOf(v))
}

// PreferredNameOf derives a schema name from a service prefix and Go type.
func PreferredNameOf(service string, t reflect.Type) string {
	if service == "" || service == "API" {
		return ""
	}
	base := reflectutil.DerefType(t)
	if base == nil || base.Name() == "" {
		return ""
	}
	return service + base.Name()
}

func isOptionalType(t reflect.Type) bool {
	return t != nil && t.Kind() == reflect.Ptr
}
