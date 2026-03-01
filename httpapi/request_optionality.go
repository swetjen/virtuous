package httpapi

import "reflect"

// Optional marks a typed request body as optional in generated OpenAPI and SDKs.
//
// Usage:
//   - Optional[MyRequest]()
//   - Optional(MyRequest{})
func Optional[T any](req ...T) any {
	var t reflect.Type
	if len(req) > 0 {
		t = reflect.TypeOf(req[0])
	}
	if t == nil {
		var ptr *T
		t = reflect.TypeOf(ptr).Elem()
	}
	return optionalRequest{typ: t}
}

type optionalRequest struct {
	typ reflect.Type
}

type requestTypeInfo struct {
	Type     reflect.Type
	Present  bool
	Optional bool
}

func resolveRequestType(req any) requestTypeInfo {
	if req == nil {
		return requestTypeInfo{}
	}
	if marker, ok := req.(optionalRequest); ok {
		if marker.typ == nil {
			return requestTypeInfo{}
		}
		return requestTypeInfo{
			Type:     marker.typ,
			Present:  true,
			Optional: true,
		}
	}
	t := reflect.TypeOf(req)
	if t == nil {
		return requestTypeInfo{}
	}
	return requestTypeInfo{
		Type:    t,
		Present: true,
	}
}
