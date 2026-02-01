package auth

import (
	"net/http"
	"strings"

	"github.com/swetjen/virtuous/guard"
)

const DemoToken = "demo-token"

type BearerGuard struct{}

func (BearerGuard) Spec() guard.Spec {
	return guard.Spec{
		Name:   "BearerAuth",
		In:     "header",
		Param:  "Authorization",
		Prefix: "Bearer",
	}
}

func (BearerGuard) Middleware() func(http.Handler) http.Handler {
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
			if token != DemoToken {
				http.Error(w, "invalid auth token", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
