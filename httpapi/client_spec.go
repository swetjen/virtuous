package httpapi

import (
	"reflect"
	"sort"

	"github.com/swetjen/virtuous/internal/reflectutil"
	"github.com/swetjen/virtuous/schema"
)

type clientSpec struct {
	Services []clientService
	Objects  []clientObject
}

type clientService struct {
	Name    string
	Methods []clientMethod
}

type clientMethod struct {
	Name          string
	FlatName      string
	Summary       string
	HTTPMethod    string
	Path          string
	PathParams    []clientPathParam
	HasBody       bool
	BodyOptional  bool
	BodyMode      string
	BodyFields    []clientBodyField
	RequestMedia  string
	HasQuery      bool
	QueryParams   []clientQueryParam
	AcceptType    string
	ResponseMode  string
	HasAuth       bool
	HasCookieAuth bool
	Auth          GuardSpec
	AuthParam     string
	AuthReqs      []clientAuthRequirement
	AuthParams    []clientAuthGuard
	RequestType   string
	ResponseType  string
}

type clientObject = schema.Object

type clientQueryParam struct {
	Name     string
	Optional bool
	IsArray  bool
	Doc      string
	Type     string
}

type clientPathParam struct {
	Name string
	Type string
}

type clientBodyField struct {
	Name     string
	WireName string
	Optional bool
	IsArray  bool
}

type clientAuthRequirement struct {
	Guards []clientAuthGuard
}

type clientAuthGuard struct {
	Spec      GuardSpec
	ParamName string
}

func buildClientSpec(routes []Route, overrides map[string]TypeOverride) (clientSpec, error) {
	return buildClientSpecWith(routes, overrides, func(registry *schema.Registry) func(reflect.Type) string {
		return registry.JSTypeOf
	}, "Uint8Array")
}

func buildPythonClientSpec(routes []Route, overrides map[string]TypeOverride) (clientSpec, error) {
	return buildClientSpecWith(routes, overrides, func(registry *schema.Registry) func(reflect.Type) string {
		return registry.PyTypeOf
	}, "bytes")
}

func buildClientSpecWith(
	routes []Route,
	overrides map[string]TypeOverride,
	typeFnFactory func(*schema.Registry) func(reflect.Type) string,
	byteType string,
) (clientSpec, error) {
	serviceMap := make(map[string]*clientService)
	registry := schema.NewRegistry(overrides)
	typeFn := typeFnFactory(registry)
	for _, route := range routes {
		if route.Handler == nil {
			continue
		}
		service := route.Meta.Service
		methodName := camelizeDown(route.Meta.Method)
		if service == "" || methodName == "" {
			continue
		}
		cs, ok := serviceMap[service]
		if !ok {
			cs = &clientService{Name: service}
			serviceMap[service] = cs
		}
		reqInfo := resolveRequestType(route.Handler.RequestType())
		hasBody := reqInfo.Present
		hasQuery := false
		var queryParams []clientQueryParam
		pathParams := fallbackClientPathParams(route.PathParams, typeFn)
		requestType := ""
		responseType := ""
		var bodyFields []clientBodyField
		if reqInfo.Present {
			reqReflect := reqInfo.Type
			if preferred := preferredSchemaName(route.Meta, reqReflect); preferred != "" {
				registry.PreferNameOf(reqReflect, preferred)
			}
			inferredPathParams, err := clientPathParamsFor(route, reqReflect, typeFn)
			if err != nil {
				return clientSpec{}, err
			}
			if len(inferredPathParams) > 0 {
				pathParams = inferredPathParams
			}
			queryInfo, err := queryParamsFor(reqReflect)
			if err != nil {
				return clientSpec{}, err
			}
			if len(queryInfo.Params) > 0 {
				hasQuery = true
				queryParams = make([]clientQueryParam, 0, len(queryInfo.Params))
				for _, param := range queryInfo.Params {
					queryParams = append(queryParams, clientQueryParam{
						Name:     param.Name,
						Optional: param.Optional,
						IsArray:  param.IsArray,
						Doc:      param.Doc,
						Type:     typeFn(param.Type),
					})
				}
			}
			hasBody = queryInfo.BodyFields > 0
			if hasBody {
				registry.AddTypeOf(reqReflect)
				requestType = typeFn(reqReflect)
			}
		}
		requestMedia := MediaTypeJSON
		bodyMode := "json"
		if route.Meta.RequestBody != nil {
			content := primaryRequestContent(*route.Meta.RequestBody)
			requestMedia = content.MediaType
			if requestMedia == "" {
				requestMedia = MediaTypeJSON
			}
			bodyMode = bodyModeForMediaType(requestMedia)
			hasBody = true
			reqInfo.Optional = !route.Meta.RequestBody.Required
			bodyType := reflect.TypeOf(content.Body)
			if bodyType != nil {
				if preferred := preferredSchemaName(route.Meta, bodyType); preferred != "" {
					registry.PreferNameOf(bodyType, preferred)
				}
				registry.AddTypeOf(bodyType)
				requestType = typeFn(bodyType)
				if bodyMode == "form" {
					fields, err := clientFormFieldsFor(bodyType)
					if err != nil {
						return clientSpec{}, err
					}
					bodyFields = fields
				}
			} else if byteType == "bytes" {
				requestType = "Any"
			} else {
				requestType = "any"
			}
		}
		pathParams, queryParams = applyExplicitClientParams(route, pathParams, queryParams, typeFn)
		if len(queryParams) > 0 {
			hasQuery = true
		}
		primaryResp, hasPrimaryResponse, err := primaryClientResponse(route)
		if err != nil {
			return clientSpec{}, err
		}
		acceptType := "application/json"
		responseMode := "none"
		if hasPrimaryResponse {
			respReflect := primaryResp.BodyType
			if primaryResp.MediaType != "" {
				acceptType = primaryResp.MediaType
			}
			responseMode = responseModeForType(respReflect)
			if respReflect != nil &&
				!isNoResponse(respReflect, reflect.TypeOf(NoResponse200{})) &&
				!isNoResponse(respReflect, reflect.TypeOf(NoResponse204{})) &&
				!isNoResponse(respReflect, reflect.TypeOf(NoResponse500{})) {
				if isByteSliceResponse(respReflect) {
					responseType = byteType
				} else {
					if preferred := preferredSchemaName(route.Meta, respReflect); preferred != "" {
						registry.PreferNameOf(respReflect, preferred)
					}
					registry.AddTypeOf(respReflect)
					responseType = typeFn(respReflect)
				}
			}
		}
		method := clientMethod{
			Name:         methodName,
			Summary:      route.Meta.Summary,
			HTTPMethod:   route.Method,
			Path:         route.Path,
			PathParams:   pathParams,
			HasBody:      hasBody,
			BodyOptional: reqInfo.Optional && hasBody,
			BodyMode:     bodyMode,
			BodyFields:   bodyFields,
			RequestMedia: requestMedia,
			HasQuery:     hasQuery,
			QueryParams:  queryParams,
			AcceptType:   acceptType,
			ResponseMode: responseMode,
			RequestType:  requestType,
			ResponseType: responseType,
		}
		if len(route.Meta.Security.Alternatives) > 0 {
			method.HasAuth = true
			method.AuthReqs = clientAuthRequirements(route.Meta.Security)
			method.AuthParams = clientAuthParams(method.AuthReqs)
			method.HasCookieAuth = clientHasCookieAuth(method.AuthReqs)
			if len(method.AuthReqs) > 0 && len(method.AuthReqs[0].Guards) > 0 {
				method.Auth = method.AuthReqs[0].Guards[0].Spec
				method.AuthParam = method.AuthReqs[0].Guards[0].ParamName
			}
		}
		cs.Methods = append(cs.Methods, method)
	}

	services := make([]clientService, 0, len(serviceMap))
	for _, svc := range serviceMap {
		sort.Slice(svc.Methods, func(i, j int) bool {
			return svc.Methods[i].Name < svc.Methods[j].Name
		})
		services = append(services, *svc)
	}
	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	return clientSpec{
		Services: services,
		Objects:  registry.ObjectsWith(typeFn),
	}, nil
}

func authParamName(name string) string {
	if name == "" {
		return "auth"
	}
	candidate := camelizeDown(name)
	if candidate == "" {
		return "auth"
	}
	return candidate
}

func fallbackClientPathParams(names []string, typeFn func(reflect.Type) string) []clientPathParam {
	if len(names) == 0 {
		return nil
	}
	typ := typeFn(reflect.TypeOf(""))
	if typ == "" {
		typ = "string"
	}
	out := make([]clientPathParam, 0, len(names))
	for _, name := range names {
		out = append(out, clientPathParam{Name: name, Type: typ})
	}
	return out
}

func clientPathParamsFor(route Route, req reflect.Type, typeFn func(reflect.Type) string) ([]clientPathParam, error) {
	pathInfo, err := pathParamsFor(req)
	if err != nil {
		return nil, err
	}
	byName := map[string]clientPathParam{}
	for _, param := range pathInfo {
		typ := typeFn(param.Type)
		if typ == "" {
			typ = "string"
		}
		byName[param.Name] = clientPathParam{Name: param.Name, Type: typ}
	}
	for _, spec := range route.Meta.Params {
		if spec.In != ParamInPath {
			continue
		}
		typ := typeFn(reflect.TypeOf(spec.Type))
		if typ == "" {
			typ = "string"
		}
		byName[spec.Name] = clientPathParam{Name: spec.Name, Type: typ}
	}
	if len(byName) == 0 {
		return nil, nil
	}
	fallbackType := typeFn(reflect.TypeOf(""))
	if fallbackType == "" {
		fallbackType = "string"
	}
	out := make([]clientPathParam, 0, len(route.PathParams))
	for _, name := range route.PathParams {
		if param, ok := byName[name]; ok {
			out = append(out, param)
			continue
		}
		out = append(out, clientPathParam{Name: name, Type: fallbackType})
	}
	return out, nil
}

func applyExplicitClientParams(route Route, pathParams []clientPathParam, queryParams []clientQueryParam, typeFn func(reflect.Type) string) ([]clientPathParam, []clientQueryParam) {
	pathByName := map[string]int{}
	for i, param := range pathParams {
		pathByName[param.Name] = i
	}
	queryByName := map[string]int{}
	for i, param := range queryParams {
		queryByName[param.Name] = i
	}
	for _, spec := range route.Meta.Params {
		typ := typeFn(reflect.TypeOf(spec.Type))
		if typ == "" {
			typ = "string"
		}
		switch spec.In {
		case ParamInPath:
			param := clientPathParam{Name: spec.Name, Type: typ}
			if idx, ok := pathByName[spec.Name]; ok {
				pathParams[idx] = param
			}
		case ParamInQuery:
			param := clientQueryParam{
				Name:     spec.Name,
				Optional: !spec.Required,
				IsArray:  isArrayType(reflect.TypeOf(spec.Type)),
				Doc:      spec.Description,
				Type:     typ,
			}
			if idx, ok := queryByName[spec.Name]; ok {
				queryParams[idx] = param
			} else {
				queryByName[spec.Name] = len(queryParams)
				queryParams = append(queryParams, param)
			}
		}
	}
	return pathParams, queryParams
}

func isArrayType(t reflect.Type) bool {
	base := reflectutil.DerefType(t)
	return base != nil && (base.Kind() == reflect.Slice || base.Kind() == reflect.Array)
}

func primaryRequestContent(spec RequestBodySpec) RequestContentSpec {
	if len(spec.Content) == 0 {
		return RequestContentSpec{MediaType: MediaTypeJSON}
	}
	return spec.Content[0]
}

func clientFormFieldsFor(t reflect.Type) ([]clientBodyField, error) {
	base := reflectutil.DerefType(t)
	if base == nil || base.Kind() != reflect.Struct {
		return nil, nil
	}
	fields := make([]clientBodyField, 0, base.NumField())
	for i := 0; i < base.NumField(); i++ {
		field := base.Field(i)
		if field.PkgPath != "" {
			continue
		}
		wireName, omit := formFieldName(field)
		if wireName == "" {
			continue
		}
		clientName, _ := reflectutil.JSONFieldName(field)
		if clientName == "" {
			continue
		}
		fields = append(fields, clientBodyField{
			Name:     clientName,
			WireName: wireName,
			Optional: omit || field.Type.Kind() == reflect.Ptr,
			IsArray:  isArrayType(field.Type),
		})
	}
	return fields, nil
}

func bodyModeForMediaType(mediaType string) string {
	switch mediaType {
	case MediaTypeFormURLEncoded:
		return "form"
	default:
		return "json"
	}
}

func clientAuthRequirements(spec SecuritySpec) []clientAuthRequirement {
	out := make([]clientAuthRequirement, 0, len(spec.Alternatives))
	for _, alt := range spec.Alternatives {
		req := clientAuthRequirement{Guards: make([]clientAuthGuard, 0, len(alt.Guards))}
		for _, guard := range alt.Guards {
			if guard.Name == "" {
				continue
			}
			req.Guards = append(req.Guards, clientAuthGuard{
				Spec:      guard,
				ParamName: authParamName(guard.Name),
			})
		}
		if len(req.Guards) > 0 {
			out = append(out, req)
		}
	}
	return out
}

func clientAuthParams(reqs []clientAuthRequirement) []clientAuthGuard {
	seen := map[string]struct{}{}
	var out []clientAuthGuard
	for _, req := range reqs {
		for _, guard := range req.Guards {
			if _, ok := seen[guard.ParamName]; ok {
				continue
			}
			seen[guard.ParamName] = struct{}{}
			out = append(out, guard)
		}
	}
	return out
}

func clientHasCookieAuth(reqs []clientAuthRequirement) bool {
	for _, req := range reqs {
		for _, guard := range req.Guards {
			if guard.Spec.In == "cookie" {
				return true
			}
		}
	}
	return false
}

func responseModeFor(respType any) string {
	if respType == nil {
		return "none"
	}
	return responseModeForType(reflect.TypeOf(respType))
}

func responseModeForType(t reflect.Type) string {
	if t == nil {
		return "none"
	}
	if isNoResponse(t, reflect.TypeOf(NoResponse200{})) ||
		isNoResponse(t, reflect.TypeOf(NoResponse204{})) ||
		isNoResponse(t, reflect.TypeOf(NoResponse500{})) {
		return "none"
	}
	if isStringResponse(t) {
		return "text"
	}
	if isByteSliceResponse(t) {
		return "bytes"
	}
	return "json"
}
