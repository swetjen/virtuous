package main

import (
	"net/http"
	"strings"

	"github.com/swetjen/virtuous"
)

const demoBearerToken = "demo-token"

type bearerGuard struct{}

func (bearerGuard) Spec() virtuous.GuardSpec {
	return virtuous.GuardSpec{
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
			prefix := "Bearer "
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
