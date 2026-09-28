package rpc

import (
	"io"
	"net/http"
	"reflect"

	"github.com/swetjen/virtuous/clientspec"
	"github.com/swetjen/virtuous/internal/clientgen"
	"github.com/swetjen/virtuous/schema"
)

// ClientSpec builds the exported, versioned client-spec document for the
// router's frozen route set. It wraps the same internal spec builder that the
// generated JS/TS/Python clients render from, running it once with the JS
// type renderer and once with the Python renderer so every field carries both
// language types. The error is always nil for the RPC router; the signature
// matches the httpapi router's.
func (r *Router) ClientSpec() (clientspec.Document, error) {
	return buildClientSpecDocument(r.Routes(), r.currentTypeOverrides()), nil
}

// WriteClientSpecJSON writes the client-spec document to w as stable,
// two-space-indented JSON. The bytes are identical to what the
// client.spec.json endpoint serves.
func (r *Router) WriteClientSpecJSON(w io.Writer) error {
	doc, err := r.ClientSpec()
	if err != nil {
		return err
	}
	return doc.WriteJSON(w)
}

// ServeClientSpec writes the client-spec JSON document as an HTTP response.
// The document is rendered once per process, then served from cache with an
// ETag; If-None-Match requests are answered with 304.
func (r *Router) ServeClientSpec(w http.ResponseWriter, req *http.Request) {
	r.serveCachedClient(w, req, &r.clientSpecCache, "application/json", "rpc client spec", r.WriteClientSpecJSON)
}

// buildClientSpecDocument runs the internal client-spec builder with both
// language renderers and zips the results into the exported document.
func buildClientSpecDocument(routes []Route, overrides map[string]TypeOverride) clientspec.Document {
	var jsRegistry *schema.Registry
	jsSpec := buildClientSpecWith(routes, overrides, func(registry *schema.Registry) func(reflect.Type) string {
		jsRegistry = registry
		return registry.JSTypeOf
	})
	var pyRegistry *schema.Registry
	pySpec := buildClientSpecWith(routes, overrides, func(registry *schema.Registry) func(reflect.Type) string {
		pyRegistry = registry
		return registry.PyTypeOf
	})

	doc := clientspec.Document{
		SpecVersion: clientspec.SpecVersion,
		Module:      clientgen.ModulePath(),
		Version:     clientgen.VirtuousVersionLabel(),
		Services:    make([]clientspec.Service, 0, len(jsSpec.Services)),
		Objects:     clientSpecDocumentObjects(jsSpec, pySpec, jsRegistry, pyRegistry),
	}

	pyMethods := map[string]clientMethod{}
	for _, service := range pySpec.Services {
		for _, method := range service.Methods {
			pyMethods[service.Name+"\x00"+method.Name] = method
		}
	}
	seenAuth := map[clientspec.AuthParam]struct{}{}
	for _, service := range jsSpec.Services {
		docService := clientspec.Service{Name: service.Name, Methods: make([]clientspec.Method, 0, len(service.Methods))}
		for _, method := range service.Methods {
			pyMethod := pyMethods[service.Name+"\x00"+method.Name]
			docMethod := clientSpecDocumentMethod(method, pyMethod)
			for _, req := range docMethod.Auth {
				for _, guard := range req.Guards {
					if _, ok := seenAuth[guard]; ok {
						continue
					}
					seenAuth[guard] = struct{}{}
					doc.AuthParams = append(doc.AuthParams, guard)
				}
			}
			docService.Methods = append(docService.Methods, docMethod)
		}
		doc.Services = append(doc.Services, docService)
	}
	return doc
}

func clientSpecDocumentMethod(js clientMethod, py clientMethod) clientspec.Method {
	out := clientspec.Method{
		Name: js.Name,
		// RPC routes are POST-only JSON calls.
		HTTPMethod: http.MethodPost,
		Path:       js.Path,
		Response: clientspec.Response{
			Mode:      "json",
			MediaType: "application/json",
			TSType:    js.ResponseType,
			PyType:    py.ResponseType,
		},
		Deprecated:      js.Deprecated,
		DeprecationNote: js.DeprecationNote,
	}
	if js.HasBody {
		out.Body = &clientspec.Body{
			Mode:      "json",
			MediaType: "application/json",
			TSType:    js.RequestType,
			PyType:    py.RequestType,
		}
	}
	if len(js.AuthGuards) > 0 {
		// RPC guards all run on every call, so they form one ANDed alternative.
		req := clientspec.AuthRequirement{Guards: make([]clientspec.AuthParam, 0, len(js.AuthGuards))}
		for _, guard := range js.AuthGuards {
			req.Guards = append(req.Guards, clientspec.AuthParam{
				Name:      guard.Spec.Name,
				In:        guard.Spec.In,
				Param:     guard.Spec.Param,
				Prefix:    guard.Spec.Prefix,
				ParamName: guard.ParamName,
			})
		}
		out.Auth = []clientspec.AuthRequirement{req}
	}
	return out
}

// clientSpecDocumentObjects zips the JS and Python object graphs. Both builds
// register the same Go types, so objects are correlated through the
// registries' type-to-name maps and fields (reflected in the same order in
// both builds) by wire name.
func clientSpecDocumentObjects(jsSpec, pySpec clientSpec, jsRegistry, pyRegistry *schema.Registry) []clientspec.Object {
	typeByJSName := map[string]reflect.Type{}
	for typ, name := range jsRegistry.ObjectNames() {
		typeByJSName[name] = typ
	}
	pyNameByType := pyRegistry.ObjectNames()
	pyObjectsByName := map[string]clientObject{}
	for _, obj := range pySpec.Objects {
		pyObjectsByName[obj.Name] = obj
	}

	out := make([]clientspec.Object, 0, len(jsSpec.Objects))
	for _, obj := range jsSpec.Objects {
		pyName := pyNameByType[typeByJSName[obj.Name]]
		pyObj := pyObjectsByName[pyName]
		pyFields := map[string]schema.Field{}
		for _, field := range pyObj.Fields {
			pyFields[field.Name] = field
		}
		docObj := clientspec.Object{Name: obj.Name, Fields: make([]clientspec.Field, 0, len(obj.Fields))}
		if pyName != obj.Name {
			docObj.PyName = pyName
		}
		for _, field := range obj.Fields {
			docObj.Fields = append(docObj.Fields, clientspec.Field{
				Name:     field.Name,
				TSType:   field.Type,
				PyType:   pyFields[field.Name].Type,
				Optional: field.Optional,
				Nullable: field.Nullable,
				Doc:      field.Doc,
			})
		}
		out = append(out, docObj)
	}
	return out
}
