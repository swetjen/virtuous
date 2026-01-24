package virtuous

import (
	"reflect"
	"sort"
	"strings"
)

// TypeOverride customizes how a Go type is rendered for clients and OpenAPI.
type TypeOverride struct {
	JSType        string
	OpenAPIType   string
	OpenAPIFormat string
}

type typeRegistry struct {
	overrides  map[string]TypeOverride
	objects    map[reflect.Type]*clientObject
	nameByType map[reflect.Type]string
	typeByName map[string]reflect.Type
}

func newTypeRegistry(overrides map[string]TypeOverride) *typeRegistry {
	return &typeRegistry{
		overrides:  mergeTypeOverrides(overrides),
		objects:    map[reflect.Type]*clientObject{},
		nameByType: map[reflect.Type]string{},
		typeByName: map[string]reflect.Type{},
	}
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
			OpenAPIType:   "string",
			OpenAPIFormat: "date-time",
		},
	}
}

func (r *typeRegistry) addType(t reflect.Type) {
	base := derefType(t)
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
		obj := &clientObject{Name: name}
		r.objects[base] = obj
		for i := 0; i < base.NumField(); i++ {
			field := base.Field(i)
			if field.PkgPath != "" {
				continue
			}
			name, omit := jsonFieldName(field)
			if name == "" {
				continue
			}
			fieldType := r.jsType(field.Type)
			if fieldType == "" {
				fieldType = "any"
			}
			obj.Fields = append(obj.Fields, clientField{
				Name:     name,
				Type:     fieldType,
				Optional: omit,
				Nullable: isOptionalType(field.Type),
				Doc:      fieldDoc(field),
			})
			r.addType(field.Type)
		}
	case reflect.Slice, reflect.Array:
		r.addType(base.Elem())
	case reflect.Map:
		r.addType(base.Elem())
	}
}

func (r *typeRegistry) objectName(t reflect.Type) string {
	if name, ok := r.nameByType[t]; ok {
		return name
	}
	name := t.Name()
	if name == "" {
		name = schemaName(t)
	}
	if other, ok := r.typeByName[name]; ok && other != t {
		name = schemaName(t)
	}
	r.nameByType[t] = name
	r.typeByName[name] = t
	return name
}

func (r *typeRegistry) objectsList() []clientObject {
	objects := make([]clientObject, 0, len(r.objects))
	for _, obj := range r.objects {
		objects = append(objects, *obj)
	}
	sort.Slice(objects, func(i, j int) bool {
		return objects[i].Name < objects[j].Name
	})
	return objects
}

func (r *typeRegistry) jsType(t reflect.Type) string {
	base := derefType(t)
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

func (r *typeRegistry) isOverrideScalar(t reflect.Type) bool {
	override, ok := typeOverrideFor(r.overrides, t)
	if !ok {
		return false
	}
	return override.OpenAPIType != "" || override.OpenAPIFormat != "" || override.JSType != ""
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

func derefType(t reflect.Type) reflect.Type {
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func isOptionalType(t reflect.Type) bool {
	return t != nil && t.Kind() == reflect.Ptr
}

func jsonFieldName(field reflect.StructField) (string, bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", false
	}
	if tag != "" {
		parts := strings.Split(tag, ",")
		name := parts[0]
		if name == "" {
			name = lowerFirst(field.Name)
		}
		return name, hasOmitEmpty(parts)
	}
	return lowerFirst(field.Name), false
}

func hasOmitEmpty(parts []string) bool {
	for _, part := range parts[1:] {
		if part == "omitempty" {
			return true
		}
	}
	return false
}

func fieldDoc(field reflect.StructField) string {
	return strings.TrimSpace(field.Tag.Get("doc"))
}
