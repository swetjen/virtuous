package httpapi

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/swetjen/virtuous/internal/reflectutil"
)

type headerParam struct {
	Name     string
	Optional bool
	Doc      string
	Type     reflect.Type
	Field    *reflect.StructField
}

// headerParamsFor extracts header: tagged fields from a request struct,
// mirroring queryParamsFor. Header-tagged fields are excluded from the request
// body schema and body-field counting by queryParamsFor.
func headerParamsFor(t reflect.Type) ([]headerParam, error) {
	base := reflectutil.DerefType(t)
	if base == nil || base.Kind() != reflect.Struct {
		return nil, nil
	}
	var out []headerParam
	for _, promoted := range reflectutil.PromotedFields(base) {
		field := promoted.Field
		name, optional, ok, err := parseHeaderTag(field)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		if field.Tag.Get("json") != "" || field.Tag.Get("query") != "" || field.Tag.Get("path") != "" {
			return nil, fmt.Errorf("header params cannot also use json/query/path tag: %s.%s", base.Name(), field.Name)
		}
		isArray, err := queryParamKind(field.Type)
		if err != nil {
			return nil, fmt.Errorf("header param %s.%s: %w", base.Name(), field.Name, err)
		}
		if isArray {
			return nil, fmt.Errorf("header param %s.%s: arrays are not supported", base.Name(), field.Name)
		}
		out = append(out, headerParam{
			Name:     name,
			Optional: optional || promoted.ParentOptional || field.Type.Kind() == reflect.Ptr,
			Doc:      reflectutil.FieldDoc(field),
			Type:     field.Type,
			Field:    &field,
		})
	}
	return out, nil
}

func parseHeaderTag(field reflect.StructField) (string, bool, bool, error) {
	tag := field.Tag.Get("header")
	if tag == "" {
		return "", false, false, nil
	}
	parts := strings.Split(tag, ",")
	name := parts[0]
	if name == "" {
		name = lowerFirst(field.Name)
	}
	if name == "-" {
		return "", false, false, nil
	}
	optional := false
	for _, part := range parts[1:] {
		switch part {
		case "", "omitempty", "optional":
			if part != "" {
				optional = true
			}
		default:
			return "", false, false, fmt.Errorf("unsupported header tag option %q on %s", part, field.Name)
		}
	}
	return name, optional, true, nil
}

// frameworkOwnedHeaders are the headers the generated clients always compute
// themselves; declared header params may not collide with them.
var frameworkOwnedHeaders = []string{"Accept", "Content-Type"}

// validateHeaderParamNames enforces the registration-time header param rules:
// every declared header name (header: tag or explicit HeaderParam spec) must
// be a valid HTTP field name (RFC 9110 token), and must not collide
// case-insensitively with a framework-owned header (Accept, Content-Type) or
// with the header used by an auth guard on the same route.
func validateHeaderParamNames(route Route, tagParams []headerParam) error {
	reserved := map[string]string{}
	for _, name := range frameworkOwnedHeaders {
		reserved[strings.ToLower(name)] = name
	}
	for _, guard := range route.Guards {
		if strings.EqualFold(guard.In, "header") && guard.Param != "" {
			reserved[strings.ToLower(guard.Param)] = guard.Param + " (auth guard " + guard.Name + ")"
		}
	}
	check := func(name string) error {
		if !isHTTPToken(name) {
			return fmt.Errorf("header param %q is not a valid HTTP field name (RFC 9110 token)", name)
		}
		if owner, ok := reserved[strings.ToLower(name)]; ok {
			return fmt.Errorf("header param %q collides with framework-owned header %s", name, owner)
		}
		return nil
	}
	for _, param := range tagParams {
		if err := check(param.Name); err != nil {
			return err
		}
	}
	for _, spec := range route.Meta.Params {
		if spec.In != ParamInHeader || spec.Name == "" {
			continue
		}
		if err := check(spec.Name); err != nil {
			return err
		}
	}
	return nil
}

// isHTTPToken reports whether name is a valid RFC 9110 token (tchar+).
func isHTTPToken(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		if !isTokenChar(name[i]) {
			return false
		}
	}
	return true
}

func isTokenChar(ch byte) bool {
	switch {
	case ch >= 'a' && ch <= 'z', ch >= 'A' && ch <= 'Z', ch >= '0' && ch <= '9':
		return true
	}
	switch ch {
	case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
		return true
	}
	return false
}
