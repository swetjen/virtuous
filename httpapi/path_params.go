package httpapi

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/swetjen/virtuous/internal/reflectutil"
)

type pathParam struct {
	Name  string
	Doc   string
	Type  reflect.Type
	Field *reflect.StructField
}

func pathParamsFor(t reflect.Type) ([]pathParam, error) {
	base := reflectutil.DerefType(t)
	if base == nil || base.Kind() != reflect.Struct {
		return nil, nil
	}
	var out []pathParam
	for i := 0; i < base.NumField(); i++ {
		field := base.Field(i)
		if field.PkgPath != "" {
			continue
		}
		name, ok, err := parsePathTag(field)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		if field.Tag.Get("json") != "" || field.Tag.Get("query") != "" {
			return nil, fmt.Errorf("path params cannot also use json/query tag: %s.%s", base.Name(), field.Name)
		}
		isArray, err := queryParamKind(field.Type)
		if err != nil {
			return nil, fmt.Errorf("path param %s.%s: %w", base.Name(), field.Name, err)
		}
		if isArray {
			return nil, fmt.Errorf("path param %s.%s: arrays are not supported", base.Name(), field.Name)
		}
		out = append(out, pathParam{
			Name:  name,
			Doc:   reflectutil.FieldDoc(field),
			Type:  field.Type,
			Field: &field,
		})
	}
	return out, nil
}

func parsePathTag(field reflect.StructField) (string, bool, error) {
	tag := field.Tag.Get("path")
	if tag == "" {
		return "", false, nil
	}
	parts := strings.Split(tag, ",")
	name := parts[0]
	if name == "" {
		name = lowerFirst(field.Name)
	}
	if name == "-" {
		return "", false, nil
	}
	for _, part := range parts[1:] {
		switch part {
		case "":
		default:
			return "", false, fmt.Errorf("unsupported path tag option %q on %s", part, field.Name)
		}
	}
	return name, true, nil
}
