package httpapi

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
// type renderer and once with the Python renderer so every field and
// parameter carries both language types.
func (r *Router) ClientSpec() (clientspec.Document, error) {
	return buildClientSpecDocument(r.Routes(), r.currentTypeOverrides())
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
	r.serveCachedClient(w, req, &r.clientSpecCache, "application/json", "client spec", r.WriteClientSpecJSON)
}

// buildClientSpecDocument runs the internal client-spec builder with both
// language renderers and zips the results into the exported document.
func buildClientSpecDocument(routes []Route, overrides map[string]TypeOverride) (clientspec.Document, error) {
	var jsRegistry *schema.Registry
	jsSpec, err := buildClientSpecWith(routes, overrides, func(registry *schema.Registry) func(reflect.Type) string {
		jsRegistry = registry
		return registry.JSTypeOf
	}, jsClientByteType, jsClientNaming())
	if err != nil {
		return clientspec.Document{}, err
	}
	var pyRegistry *schema.Registry
	pySpec, err := buildClientSpecWith(routes, overrides, func(registry *schema.Registry) func(reflect.Type) string {
		pyRegistry = registry
		return registry.PyTypeOf
	}, pyClientByteType, pyClientNaming())
	if err != nil {
		return clientspec.Document{}, err
	}

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
	responseHeaders, err := clientSpecResponseHeaders(routes, jsRegistry, pyRegistry)
	if err != nil {
		return clientspec.Document{}, err
	}
	seenAuth := map[clientspec.AuthParam]struct{}{}
	for _, service := range jsSpec.Services {
		docService := clientspec.Service{Name: service.Name, Methods: make([]clientspec.Method, 0, len(service.Methods))}
		for _, method := range service.Methods {
			pyMethod := pyMethods[service.Name+"\x00"+method.Name]
			docMethod := clientSpecDocumentMethod(method, pyMethod)
			docMethod.Response.Headers = responseHeaders[method.HTTPMethod+"\x00"+method.Path]
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
	return doc, nil
}

func clientSpecDocumentMethod(js clientMethod, py clientMethod) clientspec.Method {
	pyQuery := map[string]clientQueryParam{}
	for _, param := range py.QueryParams {
		pyQuery[param.Name] = param
	}
	pyHeaders := map[string]clientHeaderParam{}
	for _, param := range py.HeaderParams {
		pyHeaders[param.Name] = param
	}
	pyPaths := map[string]clientPathParam{}
	for _, param := range py.PathParams {
		pyPaths[param.Name] = param
	}

	out := clientspec.Method{
		Name:        js.Name,
		OperationID: js.OperationID,
		Summary:     js.Summary,
		HTTPMethod:  js.HTTPMethod,
		Path:        js.Path,
		Response: clientspec.Response{
			Mode:      js.ResponseMode,
			MediaType: js.AcceptType,
			TSType:    js.ResponseType,
			PyType:    py.ResponseType,
		},
		Deprecated:      js.Deprecated,
		DeprecationNote: js.DeprecationNote,
	}
	for _, param := range js.PathParams {
		out.PathParams = append(out.PathParams, clientspec.PathParam{
			Name:   param.Name,
			TSType: param.Type,
			PyType: pyPaths[param.Name].Type,
		})
	}
	for _, param := range js.QueryParams {
		out.QueryParams = append(out.QueryParams, clientspec.QueryParam{
			Name:     param.Name,
			TSType:   param.Type,
			PyType:   pyQuery[param.Name].Type,
			Optional: param.Optional,
			Array:    param.IsArray,
			Doc:      param.Doc,
		})
	}
	for _, param := range js.HeaderParams {
		out.HeaderParams = append(out.HeaderParams, clientspec.HeaderParam{
			Name:     param.Name,
			TSType:   param.Type,
			PyType:   pyHeaders[param.Name].Type,
			Optional: param.Optional,
			Doc:      param.Doc,
		})
	}
	if js.HasBody {
		body := &clientspec.Body{
			Mode:      js.BodyMode,
			MediaType: js.RequestMedia,
			Optional:  js.BodyOptional,
			TSType:    js.RequestType,
			PyType:    py.RequestType,
		}
		for _, field := range js.BodyFields {
			body.Fields = append(body.Fields, clientspec.BodyField{
				Name:     field.Name,
				WireName: field.WireName,
				Optional: field.Optional,
				Array:    field.IsArray,
				File:     field.IsFile,
			})
		}
		out.Body = body
	}
	for _, req := range js.AuthReqs {
		docReq := clientspec.AuthRequirement{Guards: make([]clientspec.AuthParam, 0, len(req.Guards))}
		for _, guard := range req.Guards {
			docReq.Guards = append(docReq.Guards, clientSpecDocumentGuard(guard))
		}
		out.Auth = append(out.Auth, docReq)
	}
	return out
}

func clientSpecDocumentGuard(guard clientAuthGuard) clientspec.AuthParam {
	return clientspec.AuthParam{
		Name:      guard.Spec.Name,
		In:        guard.Spec.In,
		Param:     guard.Spec.Param,
		Prefix:    guard.Spec.Prefix,
		ParamName: guard.ParamName,
	}
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

// clientSpecResponseHeaders maps each route (method + path) to the documented
// headers of its primary 2xx response spec, rendered with both language type
// functions. Routes without declared response headers are absent.
func clientSpecResponseHeaders(routes []Route, jsRegistry, pyRegistry *schema.Registry) (map[string][]clientspec.ResponseHeader, error) {
	out := map[string][]clientspec.ResponseHeader{}
	for _, route := range routes {
		primary, ok, err := primaryClientResponse(route)
		if err != nil {
			return nil, err
		}
		if !ok || len(primary.Headers) == 0 {
			continue
		}
		headers := make([]clientspec.ResponseHeader, 0, len(primary.Headers))
		for _, spec := range primary.Headers {
			tsType, pyType := "string", "str"
			if spec.Type != nil {
				typ := reflect.TypeOf(spec.Type)
				tsType = jsRegistry.JSTypeOf(typ)
				pyType = pyRegistry.PyTypeOf(typ)
			}
			headers = append(headers, clientspec.ResponseHeader{
				Name:        spec.Name,
				TSType:      tsType,
				PyType:      pyType,
				Optional:    spec.Optional,
				Description: spec.Description,
			})
		}
		out[route.Method+"\x00"+route.Path] = headers
	}
	return out, nil
}
