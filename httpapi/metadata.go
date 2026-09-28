package httpapi

import (
	"bytes"
	"io"
	"net/http"

	"github.com/swetjen/virtuous/internal/jsonlimit"
)

const (
	ParamInPath   = "path"
	ParamInQuery  = "query"
	ParamInHeader = "header"
	ParamInCookie = "cookie"

	MediaTypeJSON              = "application/json"
	MediaTypeFormURLEncoded    = "application/x-www-form-urlencoded"
	MediaTypeMultipartForm     = "multipart/form-data"
	MediaTypeMultipartFormData = MediaTypeMultipartForm
)

// File marks a multipart/form-data file part in request body metadata.
type File struct{}

// PathParam returns an explicit typed path parameter spec.
func PathParam(name string, typ any) ParamSpec {
	return ParamSpec{Name: name, In: ParamInPath, Type: typ, Required: true}
}

// QueryParam returns an explicit typed query parameter spec.
func QueryParam(name string, typ any) ParamSpec {
	return ParamSpec{Name: name, In: ParamInQuery, Type: typ}
}

// HeaderParam returns an explicit typed header parameter spec.
func HeaderParam(name string, typ any) ParamSpec {
	return ParamSpec{Name: name, In: ParamInHeader, Type: typ}
}

// CookieParam returns an explicit typed cookie parameter spec.
func CookieParam(name string, typ any) ParamSpec {
	return ParamSpec{Name: name, In: ParamInCookie, Type: typ}
}

// ResponseHeader returns an explicit typed response header spec for use in
// ResponseSpec.Headers. A nil typ documents the header as a string.
func ResponseHeader(name string, typ any) ResponseHeaderSpec {
	return ResponseHeaderSpec{Name: name, Type: typ}
}

// JSONBody returns an explicit JSON request body spec.
func JSONBody(body any) *RequestBodySpec {
	return &RequestBodySpec{
		Required: true,
		Content:  []RequestContentSpec{{MediaType: MediaTypeJSON, Body: body}},
	}
}

// FormBody returns an explicit application/x-www-form-urlencoded request body spec.
func FormBody(body any) *RequestBodySpec {
	return &RequestBodySpec{
		Required: true,
		Content:  []RequestContentSpec{{MediaType: MediaTypeFormURLEncoded, Body: body}},
	}
}

// MultipartBody returns an explicit multipart/form-data request body spec.
func MultipartBody(body any) *RequestBodySpec {
	return &RequestBodySpec{
		Required: true,
		Content:  []RequestContentSpec{{MediaType: MediaTypeMultipartForm, Body: body}},
	}
}

// SecurityAny declares OR auth semantics for OpenAPI and generated clients.
func SecurityAny(guards ...GuardSpec) SecuritySpec {
	spec := SecuritySpec{Alternatives: make([]SecurityRequirement, 0, len(guards))}
	for _, guard := range guards {
		if guard.Name == "" {
			continue
		}
		spec.Alternatives = append(spec.Alternatives, SecurityRequirement{Guards: []GuardSpec{guard}})
	}
	return spec
}

// SecurityAll declares AND auth semantics for OpenAPI and generated clients.
func SecurityAll(guards ...GuardSpec) SecuritySpec {
	req := SecurityRequirement{Guards: make([]GuardSpec, 0, len(guards))}
	for _, guard := range guards {
		if guard.Name == "" {
			continue
		}
		req.Guards = append(req.Guards, guard)
	}
	if len(req.Guards) == 0 {
		return SecuritySpec{}
	}
	return SecuritySpec{Alternatives: []SecurityRequirement{req}}
}

type securityProvider interface {
	SecuritySpec() SecuritySpec
}

type authAnyGuard struct {
	guards []Guard
	spec   SecuritySpec
}

// AuthAny composes guards with runtime OR semantics and exposes matching
// OpenAPI/client security alternatives.
//
// At runtime each guard is probed against a cloned request and a throwaway
// response recorder until one authorizes, so guards composed here must
// tolerate probing: a denial must have no irreversible side effects (such as
// consuming a one-time token or persisting state), because a later guard may
// still authorize the same request. Headers written by the guard that
// authorizes are replayed onto the real response; when every guard denies,
// the last guard's response is replayed.
//
// When more than one guard is present and the request has a body, AuthAny
// buffers the body in memory (up to the framework's default JSON body limit,
// 1 MiB) so each guard and the final handler read a fresh copy. Bodies larger
// than that are not buffered and are shared across guard probes, so a guard
// that reads an oversized body consumes it for the guards and handler that
// follow.
func AuthAny(guards ...Guard) Guard {
	out := &authAnyGuard{guards: make([]Guard, 0, len(guards))}
	specs := make([]GuardSpec, 0, len(guards))
	for _, guard := range guards {
		if guard == nil {
			continue
		}
		out.guards = append(out.guards, guard)
		spec := guard.Spec()
		if spec.Name != "" {
			specs = append(specs, spec)
		}
	}
	out.spec = SecurityAny(specs...)
	return out
}

func (g *authAnyGuard) Spec() GuardSpec {
	if len(g.spec.Alternatives) == 1 && len(g.spec.Alternatives[0].Guards) == 1 {
		return g.spec.Alternatives[0].Guards[0]
	}
	return GuardSpec{Name: "AuthAny"}
}

func (g *authAnyGuard) SecuritySpec() SecuritySpec {
	return g.spec
}

func (g *authAnyGuard) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewares := make([]func(http.Handler) http.Handler, 0, len(g.guards))
			for _, guard := range g.guards {
				if guard == nil {
					continue
				}
				if mw := guard.Middleware(); mw != nil {
					middlewares = append(middlewares, mw)
				}
			}
			// With a single guard the probe passes the original body straight
			// through; buffering only matters when a failed probe could drain
			// the body before the next guard or the handler reads it.
			var getBody func() io.ReadCloser
			if len(middlewares) > 1 && r.Body != nil && r.Body != http.NoBody {
				getBody = bufferAuthAnyBody(r)
			}
			var last *captureResponse
			for _, mw := range middlewares {
				allowed := false
				var allowedReq *http.Request
				probe := mw(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
					allowed = true
					allowedReq = req
				}))
				rec := newCaptureResponse()
				clone := r.Clone(r.Context())
				if getBody != nil {
					clone.Body = getBody()
				}
				probe.ServeHTTP(rec, clone)
				if allowed {
					if allowedReq == nil {
						allowedReq = clone
					}
					if getBody != nil {
						allowedReq.Body = getBody()
					}
					// The winning guard only ever ran against the probe
					// recorder, so replay everything it wrote (for example a
					// refreshed session cookie) onto the real response.
					for key, values := range rec.Header() {
						for _, value := range values {
							w.Header().Add(key, value)
						}
					}
					next.ServeHTTP(w, allowedReq)
					return
				}
				last = rec
			}
			if last == nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			for key, values := range last.Header() {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			status := last.status
			if status < 400 {
				status = http.StatusUnauthorized
			}
			w.WriteHeader(status)
			_, _ = w.Write(last.body.Bytes())
		})
	}
}

// authAnyMaxBufferBytes bounds how much of a request body AuthAny buffers for
// guard probing. Guards run before the handler's own decode-time limiter
// (jsonlimit wraps the body inside Decode*, not in middleware), so the raw
// body is unbounded here and the buffer must impose its own cap. It matches
// the framework's default JSON body limit.
const authAnyMaxBufferBytes = jsonlimit.DefaultMaxBytes

// bufferAuthAnyBody reads the request body into memory and returns a factory
// for fresh readers over it, also installing one (plus GetBody) on r. It
// returns nil when the body cannot be fully buffered within the cap; r then
// keeps a reader that replays the consumed prefix followed by the rest, and
// probes fall back to sharing that reader.
func bufferAuthAnyBody(r *http.Request) func() io.ReadCloser {
	orig := r.Body
	buf, err := io.ReadAll(io.LimitReader(orig, authAnyMaxBufferBytes+1))
	if err != nil || int64(len(buf)) > authAnyMaxBufferBytes {
		r.Body = stitchedBody{Reader: io.MultiReader(bytes.NewReader(buf), orig), Closer: orig}
		return nil
	}
	_ = orig.Close()
	getBody := func() io.ReadCloser { return io.NopCloser(bytes.NewReader(buf)) }
	r.Body = getBody()
	r.GetBody = func() (io.ReadCloser, error) { return getBody(), nil }
	return getBody
}

// stitchedBody rejoins an already-consumed body prefix with its remainder.
type stitchedBody struct {
	io.Reader
	io.Closer
}

type captureResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newCaptureResponse() *captureResponse {
	return &captureResponse{header: http.Header{}}
}

func (r *captureResponse) Header() http.Header {
	return r.header
}

func (r *captureResponse) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
}

func (r *captureResponse) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.body.Write(data)
}

func securitySpecFromGuards(guards []Guard) SecuritySpec {
	var and []GuardSpec
	var alternatives []SecurityRequirement
	for _, guard := range guards {
		if guard == nil {
			continue
		}
		if provider, ok := guard.(securityProvider); ok {
			spec := provider.SecuritySpec()
			if len(spec.Alternatives) > 0 {
				alternatives = append(alternatives, spec.Alternatives...)
			}
			continue
		}
		spec := guard.Spec()
		if spec.Name != "" {
			and = append(and, spec)
		}
	}
	if len(alternatives) > 0 {
		if len(and) == 0 {
			return SecuritySpec{Alternatives: alternatives}
		}
		combined := make([]SecurityRequirement, 0, len(alternatives))
		for _, alt := range alternatives {
			guards := append([]GuardSpec(nil), and...)
			guards = append(guards, alt.Guards...)
			combined = append(combined, SecurityRequirement{Guards: guards})
		}
		return SecuritySpec{Alternatives: combined}
	}
	return SecurityAll(and...)
}

func securitySpecEmpty(spec SecuritySpec) bool {
	return len(spec.Alternatives) == 0
}

func flattenSecuritySpec(spec SecuritySpec) []GuardSpec {
	seen := map[string]struct{}{}
	var out []GuardSpec
	for _, alt := range spec.Alternatives {
		for _, guard := range alt.Guards {
			if guard.Name == "" {
				continue
			}
			key := guard.Name + "\x00" + guard.In + "\x00" + guard.Param + "\x00" + guard.Prefix
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, guard)
		}
	}
	return out
}
