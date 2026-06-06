package httpapi

import (
	"io"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/swetjen/virtuous/internal/clientgen"
)

type reactQueryTSSpec struct {
	HasQueries     bool
	HasMutations   bool
	AuthParams     []clientAuthGuard
	ClientServices []clientService
	Objects        []clientObject
	Services       []reactQueryTSService
}

type reactQueryTSService struct {
	Name    string
	Methods []reactQueryTSMethod
}

type reactQueryTSMethod struct {
	ServiceName             string
	Name                    string
	HTTPMethod              string
	Path                    string
	IsQuery                 bool
	IsMutation              bool
	QueryKeyName            string
	QueryOptionsName        string
	HookName                string
	PathParams              []clientPathParam
	PathParamsType          string
	HasPathParams           bool
	QueryParams             []clientQueryParam
	QueryParamsType         string
	HasQuery                bool
	HasBody                 bool
	BodyOptional            bool
	RequestType             string
	ResponseType            string
	MutationVarsType        string
	MutationOptionsVarsType string
	MutationVarsArg         string
	MutationCallArgs        string
	QueryCallArgs           string
	QueryKeyArgs            string
	QueryKeyCallArgs        string
	QueryKeyParams          string
	QueryOptionsArgs        string
	QueryOptionsParams      string
	HookParams              string
	EnabledExpr             string
}

var reactQueryTSTemplate = template.Must(template.New("virtuous-react-query-ts").Parse(`{{ if or .HasQueries .HasMutations }}
import { {{ if .HasMutations }}useMutation{{ end }}{{ if and .HasQueries .HasMutations }}, {{ end }}{{ if .HasQueries }}useQuery{{ end }}{{ if or .HasQueries .HasMutations }}, {{ end }}{{ if .HasMutations }}type UseMutationOptions{{ end }}{{ if and .HasQueries .HasMutations }}, {{ end }}{{ if .HasQueries }}type UseQueryOptions{{ end }} } from '@tanstack/react-query'
{{ end }}

export type RequestOptions = {
	signal?: AbortSignal
	auth?: RequestAuth
}

export type RequestAuth = {
	auth?: string
{{- range $param := .AuthParams }}
	{{ $param.ParamName }}?: string
{{- end }}
	[key: string]: string | undefined
}

type MaybePromise<T> = T | Promise<T>
export type AuthProvider = RequestAuth | (() => MaybePromise<RequestAuth | null | undefined>)

export type ClientOptions = {
	baseUrl?: string
	auth?: AuthProvider
}

export class AuthNotReadyError extends Error {
	route: string
	constructor(route: string) {
		super("Auth is required for " + route + " but no auth value is available")
		this.name = "AuthNotReadyError"
		this.route = route
	}
}
{{range $object := .Objects}}
export interface {{$object.Name}} {
{{- range $field := $object.Fields}}
	{{$field.Name}}{{if $field.Optional}}?{{end}}: {{$field.Type}}{{if $field.Nullable}} | null{{end}};
{{- end}}
}
{{end}}
{{- range $service := .ClientServices }}{{- range $method := $service.Methods }}
{{- if $method.PathParams }}
export type {{ $method.PathParamsType }} = { {{- range $param := $method.PathParams }}{{ $param.Name }}: {{ $param.Type }}; {{- end }} }
{{ end -}}
{{- if $method.HasQuery }}
export type {{ $method.QueryParamsType }} = { {{- range $param := $method.QueryParams }}{{ $param.Name }}{{ if $param.Optional }}?{{ end }}: {{ $param.Type }}; {{- end }} }
{{ end -}}
{{- end }}{{- end }}
export function createClient(options: ClientOptions = {}) {
	let clientOptions: ClientOptions = {
		baseUrl: options.baseUrl ?? "/",
		auth: options.auth,
	}
	return {
		configure(nextOptions: ClientOptions) {
			clientOptions = { ...clientOptions, ...nextOptions }
		},
{{- range $service := .ClientServices }}
		{{ $service.Name }}: {
{{- range $method := $service.Methods }}
			async {{ $method.Name }}({{ if $method.PathParams }}pathParams: {{ $method.PathParamsType }}, {{ end }}{{ if $method.HasBody }}request{{ if $method.BodyOptional }}?{{ end }}: {{ $method.RequestType }}, {{ end }}{{ if $method.HasQuery }}query?: {{ $method.QueryParamsType }}, {{ end }}options?: RequestOptions): Promise<{{ if eq $method.ResponseMode "none" }}void{{ else if $method.ResponseType }}{{ $method.ResponseType }}{{ else }}unknown{{ end }}> {
				let path = "{{ $method.Path }}"
{{- if $method.PathParams }}
				if (!pathParams) {
					throw new Error("pathParams is required")
				}
{{- range $param := $method.PathParams }}
				path = path.replace("{{ printf "{%s}" $param.Name }}", encodeURIComponent(String(pathParams.{{ $param.Name }})))
{{- end }}
{{- end }}
				return _request<{{ if eq $method.ResponseMode "none" }}void{{ else if $method.ResponseType }}{{ $method.ResponseType }}{{ else }}unknown{{ end }}>(clientOptions, {
					method: "{{ $method.HTTPMethod }}",
					path,
					accept: "{{ $method.AcceptType }}",
					response: "{{ $method.ResponseMode }}",
{{- if $method.HasBody }}
					bodyMode: "{{ $method.BodyMode }}",
{{- if ne $method.BodyMode "multipart" }}
					contentType: "{{ $method.RequestMedia }}",
{{- end }}
{{- if $method.BodyOptional }}
					body: request === undefined || request === null ? undefined : request,
{{- else }}
					body: request || {},
{{- end }}
{{- if $method.BodyFields }}
					bodyFields: [
{{- range $field := $method.BodyFields }}
						["{{ $field.WireName }}", "{{ $field.Name }}", {{ if $field.IsFile }}true{{ else }}false{{ end }}],
{{- end }}
					],
{{- end }}
{{- end }}
{{- if $method.HasQuery }}
					query: [
{{- range $param := $method.QueryParams }}
						["{{ $param.Name }}", query?.{{ $param.Name }}, {{ if $param.Optional }}true{{ else }}false{{ end }}],
{{- end }}
					],
{{- end }}
{{- if $method.HasAuth }}
					auth: [
{{- range $req := $method.AuthReqs }}
						[
{{- if eq (len $req.Guards) 1 }}{{- range $guard := $req.Guards }}
							{ name: "{{ $guard.ParamName }}", in: "{{ $guard.Spec.In }}", param: "{{ $guard.Spec.Param }}", prefix: "{{ $guard.Spec.Prefix }}", generic: {{ if eq (len $method.AuthReqs) 1 }}true{{ else }}false{{ end }} },
{{- end }}{{- else }}{{- range $guard := $req.Guards }}
							{ name: "{{ $guard.ParamName }}", in: "{{ $guard.Spec.In }}", param: "{{ $guard.Spec.Param }}", prefix: "{{ $guard.Spec.Prefix }}" },
{{- end }}{{- end }}
						],
{{- end }}
					],
{{- end }}
{{- if $method.HasCookieAuth }}
					cookie: true,
{{- end }}
					options,
				})
			},
{{- end }}
		},
{{- end }}
	}
}

type QueryItem = [string, unknown, boolean]
type BodyField = [string, string, boolean]
type AuthGuard = { name: string; in: string; param: string; prefix: string; generic?: boolean }
type RequestConfig = {
	method: string
	path: string
	accept: string
	response: string
	contentType?: string
	body?: unknown
	bodyMode?: string
	bodyFields?: BodyField[]
	query?: QueryItem[]
	auth?: AuthGuard[][]
	cookie?: boolean
	options?: RequestOptions
}

async function _request<T>(clientOptions: ClientOptions, config: RequestConfig): Promise<T> {
	const headers: Record<string, string> = { "Accept": config.accept }
	if (config.contentType) {
		headers["Content-Type"] = config.contentType
	}
	let url = (clientOptions.baseUrl ?? "/") + config.path
	for (const item of config.query ?? []) {
		url = _appendQuery(url, item[0], item[1], item[2])
	}
	if (config.auth) {
		const auth = config.options?.auth ?? await _resolveAuth(clientOptions.auth)
		let applied = false
		for (const requirement of config.auth) {
			const values: Array<[AuthGuard, string]> = []
			for (const guard of requirement) {
				const value = auth && (auth[guard.name] || (guard.generic ? auth.auth : undefined))
				if (!value) {
					values.length = 0
					break
				}
				values.push([guard, value])
			}
			if (values.length === requirement.length) {
				for (const [guard, value] of values) {
					url = _applyAuth(url, headers, guard, value)
				}
				applied = true
				break
			}
		}
		if (!applied) {
			throw new AuthNotReadyError(config.method + " " + config.path)
		}
	}
	const init: RequestInit = { method: config.method, headers, signal: config.options?.signal }
	if (config.cookie) {
		init.credentials = "same-origin"
	}
	const body = _encodeBody(config.body, config.bodyMode, config.bodyFields)
	if (body !== undefined) {
		init.body = body
	}
	const response = await fetch(url, init)
	return await _decodeResponse<T>(response, config.response)
}

async function _resolveAuth(provider: AuthProvider | undefined): Promise<RequestAuth | null | undefined> {
	return typeof provider === "function" ? await provider() : provider
}

function _applyAuth(url: string, headers: Record<string, string>, guard: AuthGuard, value: string): string {
	const authValue = guard.prefix ? guard.prefix + " " + value : value
	if (guard.in === "header") {
		headers[guard.param] = authValue
	} else if (guard.in === "query") {
		url = _appendQuery(url, guard.param, authValue, false)
	} else if (guard.in === "cookie") {
		document.cookie = guard.param + "=" + encodeURIComponent(authValue) + "; path=/"
	}
	return url
}

function _appendQuery(url: string, key: string, value: unknown, optional: boolean): string {
	const parts: string[] = []
	const append = (item: unknown) => {
		if (optional && (item === "" || item === 0 || item === false || item === null || item === undefined)) {
			return
		}
		parts.push(encodeURIComponent(key) + "=" + encodeURIComponent(item === null || item === undefined ? "" : String(item)))
	}
	if (Array.isArray(value)) {
		if (value.length === 0 && !optional) {
			append("")
		}
		for (const item of value) {
			append(item)
		}
	} else {
		append(value)
	}
	if (parts.length === 0) {
		return url
	}
	const sep = url.includes("?") ? "&" : "?"
	return url + sep + parts.join("&")
}

function _encodeBody(value: unknown, mode?: string, fields?: BodyField[]): BodyInit | undefined {
	if (value === undefined || value === null) {
		return undefined
	}
	if (mode === "form") {
		const form = new URLSearchParams()
		_appendFields(form, value, fields)
		return form.toString()
	}
	if (mode === "multipart") {
		const form = new FormData()
		_appendFields(form, value, fields)
		return form
	}
	if (fields && fields.length > 0 && value && typeof value === "object" && !Array.isArray(value)) {
		const data = value as Record<string, unknown>
		const body: Record<string, unknown> = {}
		for (const field of fields) {
			if (field[1] in data) {
				body[field[0]] = data[field[1]]
			}
		}
		return JSON.stringify(body)
	}
	return JSON.stringify(value)
}

function _appendFields(form: URLSearchParams | FormData, value: unknown, fields?: BodyField[]) {
	const data = (value || {}) as Record<string, unknown>
	const items: Array<[string, unknown]> = fields && fields.length > 0 ? fields.map((field) => [field[0], data[field[1]]]) : Object.entries(data)
	for (const [key, item] of items) {
		_appendField(form, key, item)
	}
}

function _appendField(form: URLSearchParams | FormData, key: string, item: unknown) {
	if (item === undefined || item === null) {
		return
	}
	if (Array.isArray(item)) {
		for (const child of item) {
			_appendField(form, key, child)
		}
		return
	}
	if (form instanceof FormData && typeof Blob !== "undefined" && item instanceof Blob) {
		form.append(key, item)
		return
	}
	form.append(key, String(item))
}

async function _decodeResponse<T>(response: Response, mode: string): Promise<T> {
	if (mode === "text") {
		const text = await response.text()
		if (!response.ok) {
			throw new Error(text || (response.status + " " + response.statusText))
		}
		return text as T
	}
	if (mode === "bytes") {
		const raw = await response.arrayBuffer()
		if (!response.ok) {
			throw new Error(response.status + " " + response.statusText)
		}
		return new Uint8Array(raw) as T
	}
	if (mode === "none") {
		if (!response.ok) {
			throw new Error(response.status + " " + response.statusText)
		}
		return undefined as T
	}
	const text = await response.text()
	let json: unknown = null
	if (text) {
		try {
			json = JSON.parse(text)
		} catch (e) {
			if (!response.ok) {
				throw new Error(response.status + " " + response.statusText)
			}
			throw e
		}
	}
	if (!response.ok) {
		const errorBody = json as { error?: string } | null
		throw new Error(errorBody?.error || (response.status + " " + response.statusText))
	}
	return json as T
}

export const virtuousClient = createClient({ baseUrl: '' })

export function configureVirtuousClient(options: ClientOptions) {
	virtuousClient.configure(options)
}
{{ range $service := .Services }}{{ range $method := $service.Methods }}
{{- if $method.IsQuery }}
export function {{ $method.QueryKeyName }}({{ $method.QueryKeyParams }}) {
	return [{{ $method.QueryKeyArgs }}] as const
}

export function {{ $method.QueryOptionsName }}({{ $method.QueryOptionsParams }}) {
	return {
		queryKey: {{ $method.QueryKeyName }}({{ $method.QueryKeyCallArgs }}),
		queryFn: ({ signal }: { signal?: AbortSignal }) => virtuousClient.{{ $method.ServiceName }}.{{ $method.Name }}({{ $method.QueryCallArgs }}),
{{- if $method.EnabledExpr }}
		enabled: {{ $method.EnabledExpr }},
{{- end }}
	}
}

export function {{ $method.HookName }}({{ $method.HookParams }}) {
	return useQuery({
		...{{ $method.QueryOptionsName }}({{ $method.QueryOptionsArgs }}),
		...queryOptions,
	})
}
{{- else }}
export function {{ $method.HookName }}({{ $method.HookParams }}) {
	return useMutation({
		mutationFn: {{ if $method.MutationVarsArg }}({{ $method.MutationVarsArg }}){{ else }}(){{ end }} => virtuousClient.{{ $method.ServiceName }}.{{ $method.Name }}({{ $method.MutationCallArgs }}),
		...mutationOptions,
	})
}
{{- end }}
{{ end }}{{ end }}`))

// WriteReactQueryTS writes a generated TanStack React Query companion client to w.
func (r *Router) WriteReactQueryTS(w io.Writer) error {
	body, err := r.reactQueryTSBody()
	if err != nil {
		return err
	}
	hash := clientgen.HashBytes(body)
	if err := clientgen.WriteArtifactHeader(w, "//", "Virtuous React Query client hash", hash); err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}

// WriteReactQueryTSFile writes a generated TanStack React Query companion client to the file at path.
func (r *Router) WriteReactQueryTSFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return r.WriteReactQueryTS(f)
}

// WriteReactQueryTSHash writes the hash of the stable React Query TS client body to w.
func (r *Router) WriteReactQueryTSHash(w io.Writer) error {
	hash, err := r.reactQueryTSHash()
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, hash)
	return err
}

// ServeReactQueryTS writes a generated TanStack React Query companion client as an HTTP response.
func (r *Router) ServeReactQueryTS(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/typescript")
	if err := r.WriteReactQueryTS(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// ServeReactQueryTSHash writes the hash of the React Query TS client as an HTTP response.
func (r *Router) ServeReactQueryTSHash(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if err := r.WriteReactQueryTSHash(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (r *Router) reactQueryTSBody() ([]byte, error) {
	clientSpec, err := buildClientSpec(r.Routes(), r.typeOverrides)
	if err != nil {
		return nil, err
	}
	spec := buildReactQueryTSSpec(clientSpec)
	return clientgen.RenderTemplate(reactQueryTSTemplate, spec)
}

func (r *Router) reactQueryTSHash() (string, error) {
	body, err := r.reactQueryTSBody()
	if err != nil {
		return "", err
	}
	return clientgen.HashBytes(body), nil
}

func buildReactQueryTSSpec(spec clientSpec) reactQueryTSSpec {
	out := reactQueryTSSpec{
		AuthParams:     reactQueryAuthParams(spec),
		ClientServices: spec.Services,
		Objects:        spec.Objects,
	}
	nameCounts := reactQueryMethodNameCounts(spec)
	for _, service := range spec.Services {
		rqService := reactQueryTSService{Name: service.Name}
		for _, method := range service.Methods {
			rqMethod := buildReactQueryTSMethod(service.Name, method, nameCounts[method.Name] > 1)
			if rqMethod.IsQuery {
				out.HasQueries = true
			}
			if rqMethod.IsMutation {
				out.HasMutations = true
			}
			rqService.Methods = append(rqService.Methods, rqMethod)
		}
		out.Services = append(out.Services, rqService)
	}
	return out
}

func reactQueryAuthParams(spec clientSpec) []clientAuthGuard {
	seen := map[string]struct{}{"auth": {}}
	var out []clientAuthGuard
	for _, service := range spec.Services {
		for _, method := range service.Methods {
			for _, param := range method.AuthParams {
				if _, ok := seen[param.ParamName]; ok {
					continue
				}
				seen[param.ParamName] = struct{}{}
				out = append(out, param)
			}
		}
	}
	return out
}

func buildReactQueryTSMethod(serviceName string, method clientMethod, nameCollides bool) reactQueryTSMethod {
	exportBase := method.Name
	if nameCollides {
		exportBase = camelizeDown(serviceName + "_" + method.Name)
	}
	exportName := upperFirst(exportBase)
	rqMethod := reactQueryTSMethod{
		ServiceName:      serviceName,
		Name:             method.Name,
		HTTPMethod:       method.HTTPMethod,
		Path:             method.Path,
		IsQuery:          isReactQueryQueryMethod(method.HTTPMethod),
		IsMutation:       !isReactQueryQueryMethod(method.HTTPMethod),
		QueryKeyName:     exportBase + "QueryKey",
		QueryOptionsName: exportBase + "QueryOptions",
		HookName:         "use" + exportName,
		PathParams:       method.PathParams,
		PathParamsType:   method.PathParamsType,
		HasPathParams:    len(method.PathParams) > 0,
		QueryParams:      method.QueryParams,
		QueryParamsType:  method.QueryParamsType,
		HasQuery:         method.HasQuery,
		HasBody:          method.HasBody,
		BodyOptional:     method.BodyOptional,
		RequestType:      reactQueryRequestType(method.RequestType),
		ResponseType:     reactQueryResponseType(method),
	}
	if rqMethod.IsQuery {
		fillReactQueryQueryArgs(&rqMethod)
	} else {
		fillReactQueryMutationArgs(&rqMethod)
	}
	return rqMethod
}

func reactQueryMethodNameCounts(spec clientSpec) map[string]int {
	counts := map[string]int{}
	for _, service := range spec.Services {
		for _, method := range service.Methods {
			counts[method.Name]++
		}
	}
	return counts
}

func fillReactQueryQueryArgs(method *reactQueryTSMethod) {
	var keyParams []string
	var keyArgs []string
	var optionsParams []string
	var optionsArgs []string
	var hookParams []string
	var rawCallArgs []string

	if method.HasPathParams {
		keyParams = append(keyParams, "pathParams?: "+method.PathParamsType)
		keyArgs = append(keyArgs, "pathParams")
		optionsParams = append(optionsParams, "pathParams: "+method.PathParamsType+" | undefined")
		optionsArgs = append(optionsArgs, "pathParams")
		hookParams = append(hookParams, "pathParams: "+method.PathParamsType+" | undefined")
		rawCallArgs = append(rawCallArgs, "pathParams!")
		method.EnabledExpr = reactQueryEnabledExpr(method.PathParams)
	}
	if method.HasBody {
		optional := ""
		if method.BodyOptional {
			optional = "?"
		}
		keyParams = append(keyParams, "request"+optional+": "+method.RequestType)
		keyArgs = append(keyArgs, "request")
		optionsParams = append(optionsParams, "request"+optional+": "+method.RequestType)
		optionsArgs = append(optionsArgs, "request")
		hookParams = append(hookParams, "request"+optional+": "+method.RequestType)
		rawCallArgs = append(rawCallArgs, "request")
	}
	if method.HasQuery {
		keyParams = append(keyParams, "query?: "+method.QueryParamsType)
		keyArgs = append(keyArgs, "query")
		optionsParams = append(optionsParams, "query?: "+method.QueryParamsType)
		optionsArgs = append(optionsArgs, "query")
		hookParams = append(hookParams, "query?: "+method.QueryParamsType)
		rawCallArgs = append(rawCallArgs, "query")
	}
	hookParams = append(hookParams, "queryOptions?: Omit<UseQueryOptions<"+method.ResponseType+", Error>, 'queryKey' | 'queryFn'>")
	rawCallArgs = append(rawCallArgs, "{ signal }")

	method.QueryKeyParams = strings.Join(keyParams, ", ")
	method.QueryKeyArgs = reactQueryKeyArgs(method, keyArgs)
	method.QueryKeyCallArgs = strings.Join(keyArgs, ", ")
	method.QueryOptionsParams = strings.Join(optionsParams, ", ")
	method.QueryOptionsArgs = strings.Join(optionsArgs, ", ")
	method.HookParams = strings.Join(hookParams, ", ")
	method.QueryCallArgs = strings.Join(rawCallArgs, ", ")
}

func fillReactQueryMutationArgs(method *reactQueryTSMethod) {
	fields := reactQueryMutationVarFields(method)
	if len(fields) > 0 {
		method.MutationVarsType = "{ " + strings.Join(fields, "; ") + " }"
		method.MutationOptionsVarsType = method.MutationVarsType
		method.MutationVarsArg = "variables: " + method.MutationVarsType
		if method.optionalBodyOnlyMutation() {
			method.MutationOptionsVarsType = method.MutationVarsType + " | void"
			method.MutationVarsArg = "variables?: " + method.MutationVarsType
		}
	}

	method.HookParams = "mutationOptions?: UseMutationOptions<" + method.ResponseType + ", Error, "
	if method.MutationOptionsVarsType == "" {
		method.HookParams += "void"
	} else {
		method.HookParams += method.MutationOptionsVarsType
	}
	method.HookParams += ">"

	callArgs := make([]string, 0, 4)
	if method.HasPathParams {
		callArgs = append(callArgs, "variables.pathParams")
	}
	if method.HasBody {
		if method.optionalBodyOnlyMutation() {
			callArgs = append(callArgs, "variables?.request")
		} else {
			callArgs = append(callArgs, "variables.request")
		}
	}
	if method.HasQuery {
		callArgs = append(callArgs, "variables.query")
	}
	method.MutationCallArgs = strings.Join(callArgs, ", ")
}

func (method reactQueryTSMethod) optionalBodyOnlyMutation() bool {
	return method.HasBody && method.BodyOptional && !method.HasPathParams && !method.HasQuery
}

func reactQueryMutationVarFields(method *reactQueryTSMethod) []string {
	var fields []string
	if method.HasPathParams {
		fields = append(fields, "pathParams: "+method.PathParamsType)
	}
	if method.HasBody {
		optional := ""
		if method.BodyOptional {
			optional = "?"
		}
		fields = append(fields, "request"+optional+": "+method.RequestType)
	}
	if method.HasQuery {
		fields = append(fields, "query?: "+method.QueryParamsType)
	}
	return fields
}

func reactQueryEnabledExpr(params []clientPathParam) string {
	if len(params) == 0 {
		return ""
	}
	checks := make([]string, 0, len(params))
	for _, param := range params {
		checks = append(checks, "pathParams."+param.Name+" !== undefined && pathParams."+param.Name+" !== null")
	}
	return "!!pathParams && " + strings.Join(checks, " && ")
}

func reactQueryKeyArgs(method *reactQueryTSMethod, args []string) string {
	parts := []string{quoteTSString(method.HTTPMethod + " " + method.Path)}
	parts = append(parts, args...)
	return strings.Join(parts, ", ")
}

func reactQueryRequestType(requestType string) string {
	if requestType == "" {
		return "any"
	}
	return requestType
}

func reactQueryResponseType(method clientMethod) string {
	if method.ResponseMode == "none" {
		return "void"
	}
	if method.ResponseType != "" {
		return method.ResponseType
	}
	return "unknown"
}

func isReactQueryQueryMethod(method string) bool {
	method = strings.ToUpper(method)
	return method == "GET" || method == "HEAD"
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func quoteTSString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "\\'") + "'"
}
