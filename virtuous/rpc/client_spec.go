package rpc

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
	Path         string
	HasBody      bool
	HasAuth      bool
	Auth         GuardSpec
	AuthParam    string
	RequestType  string
	ResponseType string
	ErrorType    string
}

type clientObject = schema.Object

func buildClientSpec(routes []Route, overrides map[string]TypeOverride) clientSpec {
	return buildClientSpecWith(routes, overrides, func(registry *schema.Registry) func(reflect.Type) string {
		return registry.JSTypeOf
	})
}

func buildPythonClientSpec(routes []Route, overrides map[string]TypeOverride) clientSpec {
	return buildClientSpecWith(routes, overrides, func(registry *schema.Registry) func(reflect.Type) string {
		return registry.PyTypeOf
	})
}

func buildClientSpecWith(
	routes []Route,
	overrides map[string]TypeOverride,
	typeFnFactory func(*schema.Registry) func(reflect.Type) string,
) clientSpec {
	serviceMap := make(map[string]*clientService)
	registry := schema.NewRegistry(overrides)
	typeFn := typeFnFactory(registry)
	for _, route := range routes {
		service := route.Service
		methodName := route.Method
		if service == "" || methodName == "" {
			continue
		}
		cs, ok := serviceMap[service]
		if !ok {
			cs = &clientService{Name: service}
			serviceMap[service] = cs
		}

		requestType := ""
		if route.RequestType != nil {
			reqType := route.RequestType
			registry.AddTypeOf(reqType)
			requestType = typeFn(reqType)
		}

		responseType := ""
		if route.ResponseType != nil {
			respType := route.ResponseType
			registry.AddTypeOf(respType)
			responseType = typeFn(respType)
		}

		errorType := ""
		if route.ErrorType != nil {
			errType := route.ErrorType
			registry.AddTypeOf(errType)
			errorType = typeFn(errType)
		}

		method := clientMethod{
			Name:         methodName,
			Path:         route.Path,
			HasBody:      route.RequestType != nil,
			RequestType:  requestType,
			ResponseType: responseType,
			ErrorType:    errorType,
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
	}
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
