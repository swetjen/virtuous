package main

import (
	"net/http"
	"strings"

	"github.com/swetjen/virtuous/httpapi"
)

const demoBearerToken = "demo-token"
const demoAPIKey = "demo-api-key"

type bearerGuard struct{}

func (bearerGuard) Spec() httpapi.GuardSpec {
	return httpapi.GuardSpec{
		Name:   "BearerAuth",
		In:     "header",
		Param:  "Authorization",
		Prefix: "Bearer",
	}
}

func (bearerGuard) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				http.Error(w, "missing auth token", http.StatusUnauthorized)
				return
			}
			const prefix = "Bearer "
			if !strings.HasPrefix(header, prefix) {
				http.Error(w, "invalid auth token", http.StatusUnauthorized)
				return
			}
			token := strings.TrimPrefix(header, prefix)
			if token != demoBearerToken {
				http.Error(w, "invalid auth token", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type apiKeyGuard struct{}

func (apiKeyGuard) Spec() httpapi.GuardSpec {
	return httpapi.GuardSpec{
		Name:  "ApiKeyAuth",
		In:    "header",
		Param: "X-API-Key",
	}
}

func (apiKeyGuard) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-API-Key") != demoAPIKey {
				http.Error(w, "invalid api key", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
