package virtuous

import (
	"fmt"
	"reflect"
	"strings"
)

type queryParam struct {
	Name     string
	Optional bool
	IsArray  bool
	Doc      string
}

type queryParamsInfo struct {
	Params        []queryParam
	QueryFieldSet map[string]struct{}
	BodyFields    int
}

func queryParamsFor(t reflect.Type) (queryParamsInfo, error) {
	base := derefType(t)
	if base == nil || base.Kind() != reflect.Struct {
		return queryParamsInfo{BodyFields: 1}, nil
	}

	info := queryParamsInfo{
		QueryFieldSet: map[string]struct{}{},
	}
	for i := 0; i < base.NumField(); i++ {
		field := base.Field(i)
		if field.PkgPath != "" {
			continue
		}
		name, optional, ok, err := parseQueryTag(field)
		if err != nil {
			return queryParamsInfo{}, err
		}
		if ok {
			if field.Tag.Get("json") != "" {
				return queryParamsInfo{}, fmt.Errorf("query params cannot also use json tag: %s.%s", base.Name(), field.Name)
			}
			isArray, err := queryParamKind(field.Type)
			if err != nil {
				return queryParamsInfo{}, fmt.Errorf("query param %s.%s: %w", base.Name(), field.Name, err)
			}
			info.Params = append(info.Params, queryParam{
				Name:     name,
				Optional: optional,
				IsArray:  isArray,
				Doc:      fieldDoc(field),
			})
			info.QueryFieldSet[field.Name] = struct{}{}
			continue
		}

		if jsonName, _ := jsonFieldName(field); jsonName != "" {
			info.BodyFields++
		}
	}
	return info, nil
}

func parseQueryTag(field reflect.StructField) (string, bool, bool, error) {
	tag := field.Tag.Get("query")
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
			return "", false, false, fmt.Errorf("unsupported query tag option %q on %s", part, field.Name)
		}
	}
	return name, optional, true, nil
}

func queryParamKind(t reflect.Type) (bool, error) {
	base := derefType(t)
	if base == nil {
		return false, fmt.Errorf("invalid query param type")
	}
	switch base.Kind() {
	case reflect.Struct, reflect.Map:
		return false, fmt.Errorf("query params do not support %s", base.Kind())
	case reflect.Slice, reflect.Array:
		elem := derefType(base.Elem())
		if elem == nil {
			return false, fmt.Errorf("invalid query param element type")
		}
		switch elem.Kind() {
		case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
			return false, fmt.Errorf("query params do not support %s elements", elem.Kind())
		default:
			return true, nil
		}
	default:
		return false, nil
	}
}
