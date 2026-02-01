package rpc

import "net/http"

// Guard carries auth metadata and middleware for a route.
type Guard interface {
	Spec() GuardSpec
	Middleware() func(http.Handler) http.Handler
}

// GuardSpec describes how to inject auth for a route.
type GuardSpec struct {
	Name   string
	In     string
	Param  string
	Prefix string
}
