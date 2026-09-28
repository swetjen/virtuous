package rpc

import (
	"github.com/swetjen/virtuous/internal/clientgen"
	"io"
	"net/http"
	"os"
	"strings"
	"text/template"
)

var clientPyTemplate = template.Must(template.New("virtuous-rpc-py").Funcs(clientgen.TemplateFuncs()).Parse(`"""Runtime-generated Python client for Virtuous RPC routes."""

from dataclasses import dataclass, field, fields, is_dataclass
from datetime import date as _date, datetime as _datetime
from decimal import Decimal as _Decimal
import http
import json
import types
from typing import Any, Optional, Union, get_args, get_origin, get_type_hints
from urllib import error, parse, request

# Type definitions
{{- range $i, $object := .Objects }}
{{- if gt $i 0 }}{{ "\n" }}{{ end }}
@dataclass(kw_only=True)
class {{ $object.Name }}:
{{- if not $object.Fields }}
    pass
{{- else }}
{{- range $field := $object.Fields }}
{{- if $field.Doc }}
    # {{ pyComment $field.Doc }}
{{- end }}
    {{ $field.Declaration }}
{{- end }}
{{- end }}

{{- end }}

class RPCError(RuntimeError):
    def __init__(self, status: int, body: Any, message: str):
        envelope = _error_envelope(body)
        if envelope is not None:
            message = f"{message}: {envelope['message']}"
        super().__init__(message)
        self.status = status
        self.body = body
        self.code = envelope["code"] if envelope is not None else None
        self.message = message

{{- range $service := .Services }}
class {{ $service.ClassName }}:
    def __init__(self, base_url: str, headers: Optional[dict] = None, transport: Any = None):
        self._base_url = base_url
        self._headers = headers
        self._transport = transport

{{- range $method := $service.Methods }}
    def {{ $method.Name }}(self{{- if $method.HasBody }}, body: {{- if $method.RequestType }}{{ $method.RequestType }}{{- else }}Any{{- end }}{{- end }}{{- if $method.HasAuth }}, {{ $method.AuthParam }}: str | None = None{{- end }}, headers: Optional[dict] = None) -> {{- if $method.ResponseType }}{{ $method.ResponseType }}{{- else }}None{{- end }}:
{{ if $method.Deprecated }}        {{ if $method.DeprecationNote }}{{ pyStr (printf "Deprecated. %s" (pyComment $method.DeprecationNote)) }}{{ else }}"Deprecated."{{ end }}
{{ end }}        # Header precedence: client-wide default headers first, then the
        # per-call 'headers' argument; framework-computed headers (Accept,
        # Content-Type, auth) are applied last and cannot be overridden. The
        # merge is case-insensitive.
        request_headers: dict[str, str] = {}
        _merge_headers(request_headers, self._headers)
        _merge_headers(request_headers, headers)
        _set_header(request_headers, "Accept", "application/json")
        _set_header(request_headers, "Content-Type", "application/json")
        url = self._base_url + {{ pyStr $method.Path }}
{{- if $method.HasAuth }}
        if {{ $method.AuthParam }} is not None:
            auth_value = {{ $method.AuthParam }}
{{- if ne $method.Auth.Prefix "" }}
            auth_value = {{ pyStr (printf "%s " $method.Auth.Prefix) }} + {{ $method.AuthParam }}
{{- end }}
{{- if eq $method.Auth.In "header" }}
            _set_header(request_headers, {{ pyStr $method.Auth.Param }}, auth_value)
{{- end }}
{{- if eq $method.Auth.In "query" }}
            url = _append_query(url, {{ pyStr $method.Auth.Param }}, auth_value)
{{- end }}
{{- if eq $method.Auth.In "cookie" }}
            _set_header(request_headers, "Cookie", {{ pyStr (printf "%s=" $method.Auth.Param) }} + parse.quote(auth_value))
{{- end }}
{{- end }}
        data = None
{{- if $method.HasBody }}
        data = json.dumps(_encode_value(body)).encode("utf-8")
{{- end }}
        return _rpc_request(url, request_headers, data, {{ if $method.ResponseDecodeType }}{{ $method.ResponseDecodeType }}{{ else }}None{{ end }}, {{ $method.ErrorDecodeType }}, transport=self._transport)

{{- end }}
{{- end }}

class _VirtuousClient:
    def __init__(self, base_url: str = "/", *, headers: Optional[dict] = None, transport: Any = None):
        self._base_url = base_url
{{- range $service := .Services }}
        self.{{ $service.AttrName }} = {{ $service.ClassName }}(base_url, headers=headers, transport=transport)
{{- end }}


def create_client(base_url: str = "/", *, headers: Optional[dict] = None, transport: Any = None) -> _VirtuousClient:
    """Create a client.

    'headers' are default headers sent with every request; per-call
    'headers' arguments override them, and framework-computed headers
    (Accept, Content-Type, auth) always win. 'transport' is a callable
    receiving the prepared urllib.request.Request and returning a response
    object with .status, .read() and .headers (the default is
    urllib.request.urlopen).
    """
    return _VirtuousClient(base_url, headers=headers, transport=transport)


def _set_header(headers: dict[str, str], key: str, value: str) -> None:
    """Set a header case-insensitively, replacing any existing spelling."""
    lower = key.lower()
    for existing in list(headers.keys()):
        if existing.lower() == lower:
            del headers[existing]
    headers[key] = value


def _merge_headers(headers: dict[str, str], extra: Any) -> None:
    if not extra:
        return
    for key, value in extra.items():
        _set_header(headers, str(key), str(value))


def _open(req: Any, transport: Any) -> Any:
    """Dispatch a prepared urllib.request.Request through the transport hook.

    'transport' is a callable receiving the Request and returning a response
    with .status, .read() and .headers; the default is
    urllib.request.urlopen.
    """
    if transport is not None:
        return transport(req)
    return request.urlopen(req)


def _rpc_request(url: str, headers: dict[str, str], data: Any, response_type: Any, error_type: Any, transport: Any = None) -> Any:
    req = request.Request(url, data=data, method="POST", headers=headers)
    status = 0
    text = ""
    try:
        resp = _open(req, transport)
        try:
            status = resp.status if hasattr(resp, "status") else resp.getcode()
            text = resp.read().decode("utf-8")
        finally:
            close = getattr(resp, "close", None)
            if close is not None:
                close()
    except error.HTTPError as err:
        status = err.code
        text = err.read().decode("utf-8")
    body = None
    if text:
        try:
            body = json.loads(text)
        except json.JSONDecodeError as err:
            raise RPCError(status, None, f"{status} {_status_text(status)}") from err
    if status >= 400:
        if _error_envelope(body) is not None:
            raise RPCError(status, body, f"{status} {_status_text(status)}")
        err_body = _decode_value(error_type, body)
        raise RPCError(status, err_body, f"{status} {_status_text(status)}")
    if response_type is None:
        return None
    return _decode_value(response_type, body)


def _status_text(code: int) -> str:
    try:
        return http.HTTPStatus(code).phrase
    except ValueError:
        return "HTTP error"


def _error_envelope(body: Any) -> Any:
    if not isinstance(body, dict):
        return None
    err = body.get("error")
    if isinstance(err, dict) and isinstance(err.get("code"), str) and isinstance(err.get("message"), str):
        return err
    return None


def _decode_value(tp: Any, value: Any) -> Any:
    if value is None:
        return None
    origin = get_origin(tp)
    if origin is None:
        if tp is _datetime:
            return _decode_datetime(value)
        if tp is _date:
            return _decode_date(value)
        if tp is _Decimal:
            return _decode_decimal(value)
        if is_dataclass(tp):
            return _decode_dataclass(tp, value)
        return value
    if origin is list:
        args = get_args(tp)
        elem_type = args[0] if args else Any
        return [_decode_value(elem_type, item) for item in value]
    if origin is dict:
        args = get_args(tp)
        value_type = args[1] if len(args) > 1 else Any
        return {key: _decode_value(value_type, item) for key, item in value.items()}
    if origin in (Union, types.UnionType):
        args = [arg for arg in get_args(tp) if arg is not type(None)]
        if len(args) == 1:
            return _decode_value(args[0], value)
        return value
    return value


def _decode_datetime(value: Any) -> Any:
    if isinstance(value, _datetime):
        return value
    if isinstance(value, str):
        return _datetime.fromisoformat(value.replace("Z", "+00:00"))
    return value


def _decode_date(value: Any) -> Any:
    if isinstance(value, _date) and not isinstance(value, _datetime):
        return value
    if isinstance(value, str):
        return _date.fromisoformat(value)
    return value


def _decode_decimal(value: Any) -> Any:
    if isinstance(value, _Decimal):
        return value
    return _Decimal(str(value))


def _decode_dataclass(cls: type[Any], data: Any) -> Any:
    if data is None:
        return None
    if not isinstance(data, dict):
        return data
    type_hints = get_type_hints(cls)
    kwargs: dict[str, Any] = {}
    for field in fields(cls):
        wire_name = field.metadata.get("wire", field.name)
        if wire_name in data:
            field_type = type_hints.get(field.name, Any)
            kwargs[field.name] = _decode_value(field_type, data[wire_name])
    return cls(**kwargs)


def _encode_value(value: Any) -> Any:
    if value is None:
        return None
    if isinstance(value, _datetime):
        if value.tzinfo is None or value.tzinfo.utcoffset(value) is None:
            raise TypeError("timezone-aware datetime required")
        return value.isoformat()
    if isinstance(value, _date):
        return value.isoformat()
    if isinstance(value, _Decimal):
        return format(value, "f")
    if is_dataclass(value):
        return {field.metadata.get("wire", field.name): _encode_value(getattr(value, field.name)) for field in fields(value)}
    if isinstance(value, list):
        return [_encode_value(item) for item in value]
    if isinstance(value, dict):
        return {key: _encode_value(item) for key, item in value.items()}
    return value


def _append_query(url: str, key: str, value: str) -> str:
    parts = parse.urlsplit(url)
    query = parse.parse_qsl(parts.query, keep_blank_values=True)
    query.append((key, value))
    new_query = parse.urlencode(query)
    return parse.urlunsplit((parts.scheme, parts.netloc, parts.path, new_query, parts.fragment))
`))

type pythonClientSpec struct {
	Services []pythonClientService
	Objects  []pythonClientObject
}

type pythonClientService struct {
	ClassName string
	AttrName  string
	Methods   []pythonClientMethod
}

type pythonClientMethod struct {
	Name               string
	Path               string
	HasBody            bool
	HasAuth            bool
	Auth               GuardSpec
	AuthParam          string
	RequestType        string
	ResponseType       string
	ResponseDecodeType string
	ErrorType          string
	ErrorDecodeType    string
	Deprecated         bool
	DeprecationNote    string
}

type pythonClientObject struct {
	Name   string
	Fields []pythonClientField
}

type pythonClientField struct {
	Name        string
	WireName    string
	Declaration string
	Doc         string
}

func buildPythonClientRenderSpec(spec clientSpec) pythonClientSpec {
	typeNames := pythonObjectNameMap(spec.Objects, spec.Services)
	out := pythonClientSpec{
		Objects: pythonObjects(spec.Objects, typeNames),
	}
	serviceAttrs := map[string]struct{}{"_base_url": {}, "_headers": {}, "_transport": {}}
	serviceClasses := map[string]struct{}{}
	for _, service := range spec.Services {
		pyService := pythonClientService{
			ClassName: clientgen.UniquePythonIdentifier("_"+service.Name+"Service", serviceClasses),
			AttrName:  clientgen.UniquePythonIdentifier(service.Name, serviceAttrs),
		}
		methodNames := map[string]struct{}{}
		for _, method := range service.Methods {
			pyService.Methods = append(pyService.Methods, pythonMethod(method, typeNames, methodNames))
		}
		out.Services = append(out.Services, pyService)
	}
	return out
}

func pythonObjectNameMap(objects []clientObject, services []clientService) map[string]string {
	used := pythonReservedModuleNames(services)
	out := make(map[string]string, len(objects))
	for _, object := range objects {
		out[object.Name] = clientgen.UniquePythonIdentifier(object.Name, used)
	}
	return out
}

func pythonReservedModuleNames(services []clientService) map[string]struct{} {
	names := []string{
		"Any",
		"Optional",
		"RPCError",
		"Union",
		"_VirtuousClient",
		"_append_query",
		"_date",
		"_decode_dataclass",
		"_decode_date",
		"_decode_datetime",
		"_decode_decimal",
		"_decode_value",
		"_encode_value",
		"_error_envelope",
		"_datetime",
		"_Decimal",
		"_merge_headers",
		"_open",
		"_rpc_request",
		"_set_header",
		"_status_text",
		"create_client",
		"dataclass",
		"dict",
		"error",
		"field",
		"fields",
		"get_args",
		"get_origin",
		"get_type_hints",
		"http",
		"id",
		"int",
		"is_dataclass",
		"json",
		"list",
		"object",
		"parse",
		"request",
		"set",
		"str",
		"types",
		"type",
	}
	out := make(map[string]struct{}, len(names))
	for _, name := range names {
		out[name] = struct{}{}
	}
	serviceNames := map[string]struct{}{}
	for _, service := range services {
		serviceNames[clientgen.PythonIdentifier("_"+service.Name+"Service")] = struct{}{}
	}
	for name := range serviceNames {
		out[name] = struct{}{}
	}
	return out
}

func pythonObjects(objects []clientObject, typeNames map[string]string) []pythonClientObject {
	out := make([]pythonClientObject, 0, len(objects))
	for _, object := range objects {
		pyObject := pythonClientObject{Name: typeNames[object.Name]}
		fieldNames := map[string]struct{}{}
		for _, field := range object.Fields {
			name := clientgen.UniquePythonIdentifier(field.Name, fieldNames)
			fieldType := pythonTypeName(field.Type, typeNames)
			pyObject.Fields = append(pyObject.Fields, pythonClientField{
				Name:        name,
				WireName:    field.Name,
				Declaration: pythonFieldDeclaration(name, field.Name, fieldType, field.Optional || field.Nullable),
				Doc:         field.Doc,
			})
		}
		out = append(out, pyObject)
	}
	return out
}

func pythonFieldDeclaration(name, wireName, fieldType string, optional bool) string {
	if fieldType == "" {
		fieldType = "Any"
	}
	typeExpr := fieldType
	if optional {
		typeExpr = "Optional[" + fieldType + "]"
	}
	if name == wireName {
		if optional {
			return name + ": " + typeExpr + " = None"
		}
		return name + ": " + typeExpr
	}
	meta := `metadata={"wire": ` + clientgen.PythonStringLiteral(wireName) + `}`
	if optional {
		return name + ": " + typeExpr + " = field(default=None, " + meta + ")"
	}
	return name + ": " + typeExpr + " = field(" + meta + ")"
}

func pythonMethod(method clientMethod, typeNames map[string]string, methodNames map[string]struct{}) pythonClientMethod {
	usedParams := map[string]struct{}{"self": {}, "headers": {}}
	if method.HasBody {
		usedParams["body"] = struct{}{}
	}
	pyMethod := pythonClientMethod{
		Name:            clientgen.UniquePythonIdentifier(method.Name, methodNames),
		Path:            method.Path,
		HasBody:         method.HasBody,
		HasAuth:         method.HasAuth,
		Auth:            method.Auth,
		AuthParam:       clientgen.UniquePythonIdentifier(method.AuthParam, usedParams),
		RequestType:     pythonTypeName(method.RequestType, typeNames),
		ResponseType:    pythonTypeName(method.ResponseType, typeNames),
		ErrorType:       pythonTypeName(method.ErrorType, typeNames),
		Deprecated:      method.Deprecated,
		DeprecationNote: method.DeprecationNote,
	}
	pyMethod.ResponseDecodeType = pythonRuntimeTypeName(pyMethod.ResponseType)
	pyMethod.ErrorDecodeType = pythonRuntimeTypeName(pyMethod.ErrorType)
	return pyMethod
}

func pythonTypeName(typeName string, names map[string]string) string {
	out := typeName
	for oldName, newName := range names {
		if oldName == newName {
			continue
		}
		out = strings.ReplaceAll(out, `"`+oldName+`"`, `"`+newName+`"`)
	}
	return pythonRuntimeAnnotationName(out)
}

func pythonRuntimeTypeName(typeName string) string {
	if typeName == "" {
		return ""
	}
	return strings.ReplaceAll(typeName, `"`, "")
}

func pythonRuntimeAnnotationName(typeName string) string {
	out := replacePythonTypeToken(typeName, "datetime", "_datetime")
	out = replacePythonTypeToken(out, "date", "_date")
	out = replacePythonTypeToken(out, "Decimal", "_Decimal")
	return out
}

func replacePythonTypeToken(value, oldToken, newToken string) string {
	if value == "" {
		return value
	}
	var out strings.Builder
	for i := 0; i < len(value); {
		if strings.HasPrefix(value[i:], oldToken) &&
			(i == 0 || !isPythonTypeIdentChar(value[i-1])) &&
			(i+len(oldToken) == len(value) || !isPythonTypeIdentChar(value[i+len(oldToken)])) {
			out.WriteString(newToken)
			i += len(oldToken)
			continue
		}
		out.WriteByte(value[i])
		i++
	}
	return out.String()
}

func isPythonTypeIdentChar(ch byte) bool {
	return ch == '_' || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
}

// WriteClientPY writes a runtime-generated Python client to w.
func (r *Router) WriteClientPY(w io.Writer) error {
	body, err := r.clientPYBody()
	if err != nil {
		return err
	}
	hash := clientgen.HashBytes(body)
	if err := clientgen.WriteArtifactHeader(w, "#", "Virtuous client hash", hash); err != nil {
		return err
	}
	if r.pythonSigning != nil {
		if err := clientgen.WritePythonSignatureEnvelope(w, *r.pythonSigning, body, hash); err != nil {
			return err
		}
	}
	_, err = w.Write(body)
	return err
}

// WriteClientPYFile writes a runtime-generated Python client to the file at path.
func (r *Router) WriteClientPYFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return r.WriteClientPY(f)
}

// WriteClientPYHash writes the hash of the stable Python client body to w.
func (r *Router) WriteClientPYHash(w io.Writer) error {
	hash, err := r.clientPYHash()
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, hash)
	return err
}

// ServeClientPY writes a runtime-generated Python client as an HTTP response.
// The client is rendered once per process, then served from cache with an
// ETag; If-None-Match requests are answered with 304. The cache stores the
// fully enveloped bytes, so a signed client carries one issued-at per process.
func (r *Router) ServeClientPY(w http.ResponseWriter, req *http.Request) {
	r.serveCachedClient(w, req, &r.clientPYCache, "text/x-python; charset=utf-8", "rpc client py", r.WriteClientPY)
}

// ServeClientPYHash writes the hash of the Python client as an HTTP response.
func (r *Router) ServeClientPYHash(w http.ResponseWriter, _ *http.Request) {
	hash, err := r.clientPYHash()
	if err != nil {
		r.logger.Error("rpc client py hash generation failed", "error", err)
		http.Error(w, "client generation failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, hash)
}

func (r *Router) clientPYBody() ([]byte, error) {
	spec := buildPythonClientSpec(r.Routes(), r.typeOverrides)
	return clientgen.RenderTemplate(clientPyTemplate, buildPythonClientRenderSpec(spec))
}

func (r *Router) clientPYHash() (string, error) {
	body, err := r.clientPYBody()
	if err != nil {
		return "", err
	}
	return clientgen.HashBytes(body), nil
}
