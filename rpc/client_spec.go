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
	AuthGuards   []clientAuthGuard
	RequestType  string
	ResponseType string
	ErrorType    string
	// Deprecated and DeprecationNote come from the rpc.Deprecated route
	// option and only drive documentation tags in generated clients.
	Deprecated      bool
	DeprecationNote string
}

// clientAuthGuard pairs a guard spec with the client-facing parameter name
// the generators bind its credential to.
type clientAuthGuard struct {
	Spec      GuardSpec
	ParamName string
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

		method := clientMethod{
			Name:         methodName,
			Path:         route.Path,
			HasBody:      route.RequestType != nil,
			RequestType:  requestType,
			ResponseType: responseType,
			ErrorType:    responseType,
			Deprecated:   route.Deprecated,
		}
		if route.Deprecated {
			method.DeprecationNote = route.DeprecationNote
		}
		if len(route.Guards) > 0 {
			// Current client templates expose a single auth input, so they bind
			// to the first declared guard for the route. AuthGuards carries the
			// full ANDed guard list for the exported client-spec document.
			method.HasAuth = true
			method.Auth = route.Guards[0]
			method.AuthParam = authParamName(route.Guards[0].Name)
			for _, guardSpec := range route.Guards {
				if guardSpec.Name == "" {
					continue
				}
				method.AuthGuards = append(method.AuthGuards, clientAuthGuard{
					Spec:      guardSpec,
					ParamName: authParamName(guardSpec.Name),
				})
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
