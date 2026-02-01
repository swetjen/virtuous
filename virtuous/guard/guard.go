package guard

import "net/http"

// Guard carries auth metadata and middleware for a route.
type Guard interface {
	Spec() Spec
	Middleware() func(http.Handler) http.Handler
}

// Spec describes how to inject auth for a route.
type Spec struct {
	Name   string
	In     string
	Param  string
	Prefix string
}
