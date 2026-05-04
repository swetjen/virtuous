package httpapi

import (
	"bytes"
	"net/http"
)

const (
	ParamInPath   = "path"
	ParamInQuery  = "query"
	ParamInHeader = "header"
	ParamInCookie = "cookie"

	MediaTypeJSON           = "application/json"
	MediaTypeFormURLEncoded = "application/x-www-form-urlencoded"
)

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
			var last *captureResponse
			for _, guard := range g.guards {
				if guard == nil || guard.Middleware() == nil {
					continue
				}
				allowed := false
				var allowedReq *http.Request
				probe := guard.Middleware()(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
					allowed = true
					allowedReq = req
				}))
				rec := newCaptureResponse()
				probe.ServeHTTP(rec, r.Clone(r.Context()))
				if allowed {
					if allowedReq == nil {
						allowedReq = r
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
