package httpapi

import (
	"reflect"
	"sort"

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
	Name         string
	Summary      string
	HTTPMethod   string
	Path         string
	PathParams   []string
	HasBody      bool
	HasQuery     bool
	QueryParams  []clientQueryParam
	HasAuth      bool
	Auth         GuardSpec
	AuthParam    string
	RequestType  string
	ResponseType string
}

type clientObject = schema.Object

type clientQueryParam struct {
	Name     string
	Optional bool
	IsArray  bool
	Doc      string
}

func buildClientSpec(routes []Route, overrides map[string]TypeOverride) (clientSpec, error) {
	return buildClientSpecWith(routes, overrides, func(registry *schema.Registry) func(reflect.Type) string {
		return registry.JSTypeOf
	})
}

func buildPythonClientSpec(routes []Route, overrides map[string]TypeOverride) (clientSpec, error) {
	return buildClientSpecWith(routes, overrides, func(registry *schema.Registry) func(reflect.Type) string {
		return registry.PyTypeOf
	})
}

func buildClientSpecWith(
	routes []Route,
	overrides map[string]TypeOverride,
	typeFnFactory func(*schema.Registry) func(reflect.Type) string,
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
		reqType := route.Handler.RequestType()
		respType := route.Handler.ResponseType()
		hasBody := reqType != nil
		hasQuery := false
		var queryParams []clientQueryParam
		requestType := ""
		responseType := ""
		if reqType != nil {
			reqReflect := reflect.TypeOf(reqType)
			if preferred := preferredSchemaName(route.Meta, reqReflect); preferred != "" {
				registry.PreferNameOf(reqReflect, preferred)
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
					})
				}
			}
			hasBody = queryInfo.BodyFields > 0
			if hasBody {
				registry.AddTypeOf(reqReflect)
				requestType = typeFn(reqReflect)
			}
		}
		if respType != nil {
			respReflect := reflect.TypeOf(respType)
			if !isNoResponse(respReflect, reflect.TypeOf(NoResponse200{})) &&
				!isNoResponse(respReflect, reflect.TypeOf(NoResponse204{})) &&
				!isNoResponse(respReflect, reflect.TypeOf(NoResponse500{})) {
				if preferred := preferredSchemaName(route.Meta, respReflect); preferred != "" {
					registry.PreferNameOf(respReflect, preferred)
				}
				registry.AddTypeOf(respReflect)
				responseType = typeFn(respReflect)
			}
		}
		method := clientMethod{
			Name:         methodName,
			Summary:      route.Meta.Summary,
			HTTPMethod:   route.Method,
			Path:         route.Path,
			PathParams:   route.PathParams,
			HasBody:      hasBody,
			HasQuery:     hasQuery,
			QueryParams:  queryParams,
			RequestType:  requestType,
			ResponseType: responseType,
		}
		if len(route.Guards) > 0 {
			method.HasAuth = true
			method.Auth = route.Guards[0]
			method.AuthParam = authParamName(route.Guards[0].Name)
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
