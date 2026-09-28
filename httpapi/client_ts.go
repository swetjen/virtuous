package httpapi

import (
	"io"
	"net/http"
	"os"
	"text/template"

	"github.com/swetjen/virtuous/internal/clientgen"
)

var clientTSTemplate = template.Must(template.New("virtuous-ts").Funcs(clientgen.TemplateFuncs()).Parse(`export type RequestOptions = {
	signal?: AbortSignal
	// A bare string is shorthand for { auth: value } (the generic slot).
	auth?: RequestAuth | string
	headers?: Record<string, string>
}

export type RequestAuth = {
	auth?: string
{{- range $auth := .AuthParams }}
	{{ tsKey $auth.ParamName }}?: string
{{- end }}
	[key: string]: string | undefined
}

type MaybePromise<T> = T | Promise<T>
export type AuthProvider = RequestAuth | string | (() => MaybePromise<RequestAuth | string | null | undefined>)

export type ClientOptions = {
	baseUrl?: string
	auth?: AuthProvider
	// Default headers sent with every request; per-call RequestOptions.headers
	// and framework-computed headers override them (see _request).
	headers?: Record<string, string>
	// Transport hook: replaces the global fetch for every request.
	fetch?: typeof fetch
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
	{{tsKey $field.Name}}{{if $field.Optional}}?{{end}}: {{$field.Type}}{{if $field.Nullable}} | null{{end}};
{{- end}}
}
{{end}}
{{- range $service := .Services }}{{- range $method := $service.Methods }}
{{- if $method.PathParams }}
export type {{ $method.PathParamsType }} = { {{- range $param := $method.PathParams }}{{ tsKey $param.Name }}: {{ $param.Type }}; {{- end }} }
{{ end -}}
{{- if $method.HasQuery }}
export type {{ $method.QueryParamsType }} = { {{- range $param := $method.QueryParams }}{{ tsKey $param.Name }}{{ if $param.Optional }}?{{ end }}: {{ $param.Type }}; {{- end }} }
{{ end -}}
{{- if $method.HasHeaders }}
export type {{ $method.HeaderParamsType }} = { {{- range $param := $method.HeaderParams }}{{ tsKey $param.Name }}{{ if $param.Optional }}?{{ end }}: {{ $param.Type }}; {{- end }} }
{{ end -}}
{{- end }}{{- end }}
export function createClient(options: ClientOptions = {}) {
	let clientOptions: ClientOptions = {
		baseUrl: options.baseUrl ?? "/",
		auth: options.auth,
		headers: options.headers,
		fetch: options.fetch,
	}
	return {
		configure(nextOptions: ClientOptions) {
			clientOptions = { ...clientOptions, ...nextOptions }
		},
{{- range $service := .Services }}
		{{ $service.Name }}: {
{{- range $method := $service.Methods }}
			{{ if $method.Deprecated }}/** @deprecated{{ if $method.DeprecationNote }} {{ jsdoc $method.DeprecationNote }}{{ end }} */
			{{ end }}async {{ $method.Name }}({{ if $method.PathParams }}pathParams: {{ $method.PathParamsType }}, {{ end }}{{ if $method.HasBody }}request{{ if not $method.BodyOptional }}: {{ $method.RequestType }}{{ else if $method.HeadersRequired }}: {{ $method.RequestType }} | undefined{{ else }}?: {{ $method.RequestType }}{{ end }}, {{ end }}{{ if $method.HasQuery }}query{{ if $method.HeadersRequired }}: {{ $method.QueryParamsType }} | undefined{{ else }}?: {{ $method.QueryParamsType }}{{ end }}, {{ end }}{{ if $method.HasHeaders }}headers{{ if not $method.HeadersRequired }}?{{ end }}: {{ $method.HeaderParamsType }}, {{ end }}options?: RequestOptions): Promise<{{ if eq $method.ResponseMode "none" }}void{{ else if $method.ResponseType }}{{ $method.ResponseType }}{{ else }}unknown{{ end }}> {
				let path = {{ tsStr $method.Path }}
{{- if $method.PathParams }}
				if (!pathParams) {
					throw new Error("pathParams is required")
				}
{{- range $param := $method.PathParams }}
				path = path.replace({{ tsStr (printf "{%s}" $param.Name) }}, encodeURIComponent(String({{ jsGet "pathParams" $param.Name }})))
{{- end }}
{{- end }}
				return _request<{{ if eq $method.ResponseMode "none" }}void{{ else if $method.ResponseType }}{{ $method.ResponseType }}{{ else }}unknown{{ end }}>(clientOptions, {
					method: {{ tsStr $method.HTTPMethod }},
					path,
					accept: {{ tsStr $method.AcceptType }},
					response: {{ tsStr $method.ResponseMode }},
{{- if $method.HasBody }}
					bodyMode: {{ tsStr $method.BodyMode }},
{{- if ne $method.BodyMode "multipart" }}
					contentType: {{ tsStr $method.RequestMedia }},
{{- end }}
{{- if $method.BodyOptional }}
					body: request === undefined || request === null ? undefined : request,
{{- else }}
					body: request || {},
{{- end }}
{{- if $method.BodyFields }}
					bodyFields: [
{{- range $field := $method.BodyFields }}
						[{{ tsStr $field.WireName }}, {{ tsStr $field.Name }}, {{ if $field.IsFile }}true{{ else }}false{{ end }}],
{{- end }}
					],
{{- end }}
{{- end }}
{{- if $method.HasQuery }}
					query: [
{{- range $param := $method.QueryParams }}
						[{{ tsStr $param.Name }}, {{ jsGetOpt "query" $param.Name }}, {{ if $param.Optional }}true{{ else }}false{{ end }}],
{{- end }}
					],
{{- end }}
{{- if $method.HasHeaders }}
					headerParams: [
{{- range $param := $method.HeaderParams }}
						[{{ tsStr $param.Name }}, {{ jsGetOpt "headers" $param.Name }}, {{ if $param.Optional }}true{{ else }}false{{ end }}],
{{- end }}
					],
{{- end }}
{{- if $method.HasAuth }}
					auth: [
{{- range $req := $method.AuthReqs }}
						[
{{- if eq (len $req.Guards) 1 }}{{- range $guard := $req.Guards }}
							{ name: {{ tsStr $guard.ParamName }}, in: {{ tsStr $guard.Spec.In }}, param: {{ tsStr $guard.Spec.Param }}, prefix: {{ tsStr $guard.Spec.Prefix }}, generic: {{ if eq (len $method.AuthReqs) 1 }}true{{ else }}false{{ end }} },
{{- end }}{{- else }}{{- range $guard := $req.Guards }}
							{ name: {{ tsStr $guard.ParamName }}, in: {{ tsStr $guard.Spec.In }}, param: {{ tsStr $guard.Spec.Param }}, prefix: {{ tsStr $guard.Spec.Prefix }} },
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
	headerParams?: QueryItem[]
	auth?: AuthGuard[][]
	cookie?: boolean
	options?: RequestOptions
}

// _setHeader sets a header case-insensitively: any existing spelling of key
// is removed before the new value is stored under the given spelling.
function _setHeader(headers: Record<string, string>, key: string, value: string) {
	const lower = key.toLowerCase()
	for (const existing of Object.keys(headers)) {
		if (existing.toLowerCase() === lower) {
			delete headers[existing]
		}
	}
	headers[key] = value
}

async function _request<T>(clientOptions: ClientOptions, config: RequestConfig): Promise<T> {
	// Header precedence: client-wide ClientOptions.headers defaults first,
	// then declared typed header params, then per-call RequestOptions.headers.
	// Framework-computed headers (Accept, Content-Type when a body is sent,
	// and auth headers) are applied last and cannot be overridden. The merge
	// is case-insensitive.
	const headers: Record<string, string> = {}
	for (const [key, value] of Object.entries(clientOptions.headers ?? {})) {
		_setHeader(headers, key, String(value))
	}
	for (const item of config.headerParams ?? []) {
		if (item[1] === undefined || item[1] === null) {
			continue
		}
		_setHeader(headers, item[0], String(item[1]))
	}
	for (const [key, value] of Object.entries(config.options?.headers ?? {})) {
		_setHeader(headers, key, String(value))
	}
	_setHeader(headers, "Accept", config.accept)
	if (config.contentType) {
		_setHeader(headers, "Content-Type", config.contentType)
	}
	let url = (clientOptions.baseUrl ?? "/") + config.path
	for (const item of config.query ?? []) {
		url = _appendQuery(url, item[0], item[1], item[2])
	}
	if (config.auth) {
		const auth = _normalizeAuth(config.options?.auth) ?? await _resolveAuth(clientOptions.auth)
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
	const fetchFn = clientOptions.fetch ?? fetch
	const response = await fetchFn(url, init)
	return await _decodeResponse<T>(response, config.response)
}

async function _resolveAuth(provider: AuthProvider | undefined): Promise<RequestAuth | null | undefined> {
	return _normalizeAuth(typeof provider === "function" ? await provider() : provider)
}

// _normalizeAuth widens the string shorthand into the generic auth slot; keyed
// objects and empty values pass through unchanged.
function _normalizeAuth(value: RequestAuth | string | null | undefined): RequestAuth | null | undefined {
	return typeof value === "string" ? { auth: value } : value
}

function _applyAuth(url: string, headers: Record<string, string>, guard: AuthGuard, value: string): string {
	const authValue = guard.prefix ? guard.prefix + " " + value : value
	if (guard.in === "header") {
		_setHeader(headers, guard.param, authValue)
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
		if (optional && (item === null || item === undefined)) {
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
		const envelope = _errorEnvelope(json)
		if (envelope) {
			const err = new Error(response.status + " " + response.statusText + ": " + envelope.message) as Error & { status: number; code: string; body: unknown }
			err.status = response.status
			err.code = envelope.code
			err.body = json
			throw err
		}
		const errorBody = json as { error?: string } | null
		throw new Error(errorBody?.error || (response.status + " " + response.statusText))
	}
	return json as T
}

function _errorEnvelope(body: unknown): { code: string; message: string } | null {
	if (!body || typeof body !== "object") {
		return null
	}
	const error = (body as { error?: unknown }).error
	if (!error || typeof error !== "object") {
		return null
	}
	const code = (error as { code?: unknown }).code
	const message = (error as { message?: unknown }).message
	if (typeof code === "string" && typeof message === "string") {
		return { code, message }
	}
	return null
}
`))

// WriteClientTS writes a generated TS client to w.
func (r *Router) WriteClientTS(w io.Writer) error {
	body, err := r.clientTSBody()
	if err != nil {
		return err
	}
	hash := clientgen.HashBytes(body)
	if err := clientgen.WriteArtifactHeader(w, "//", "Virtuous client hash", hash); err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}

// WriteClientTSFile writes a generated TS client to the file at path.
func (r *Router) WriteClientTSFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return r.WriteClientTS(f)
}

// WriteClientTSHash writes the hash of the stable TS client body to w.
func (r *Router) WriteClientTSHash(w io.Writer) error {
	hash, err := r.clientTSHash()
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, hash)
	return err
}

// ServeClientTS writes a generated TS client as an HTTP response.
// The client is rendered once per process, then served from cache with an
// ETag; If-None-Match requests are answered with 304.
func (r *Router) ServeClientTS(w http.ResponseWriter, req *http.Request) {
	r.serveCachedClient(w, req, &r.clientTSCache, "application/typescript", "client ts", r.WriteClientTS)
}

// ServeClientTSHash writes the hash of the TS client as an HTTP response.
func (r *Router) ServeClientTSHash(w http.ResponseWriter, _ *http.Request) {
	hash, err := r.clientTSHash()
	if err != nil {
		r.logger.Error("client ts hash generation failed", "error", err)
		http.Error(w, "client generation failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, hash)
}

func (r *Router) clientTSBody() ([]byte, error) {
	spec, err := buildClientSpec(r.Routes(), r.typeOverrides)
	if err != nil {
		return nil, err
	}
	return clientgen.RenderTemplate(clientTSTemplate, spec)
}

func (r *Router) clientTSHash() (string, error) {
	body, err := r.clientTSBody()
	if err != nil {
		return "", err
	}
	return clientgen.HashBytes(body), nil
}
