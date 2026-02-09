package byodbsqlite

import (
	"embed"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
)

//go:embed static/*
var staticAssets embed.FS

func embedAndServeReact() http.Handler {
	sub, err := fs.Sub(staticAssets, "static")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "static assets missing", http.StatusInternalServerError)
		})
	}

	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if f, err := sub.Open(path); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		fallback := *r
		fallback.URL = cloneURL(r.URL)
		fallback.URL.Path = "/index.html"
		fileServer.ServeHTTP(w, &fallback)
	})
}

func cloneURL(src *url.URL) *url.URL {
	if src == nil {
		return &url.URL{}
	}
	copy := *src
	return &copy
}
