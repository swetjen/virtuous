package byodb

import (
	"embed"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
)

//go:embed frontend-web/dist/*
var frontendAssets embed.FS

func embedAndServeReact() http.Handler {
	sub, err := fs.Sub(frontendAssets, "frontend-web/dist")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "frontend assets missing", http.StatusInternalServerError)
		})
	}

	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if f, err := sub.Open(path); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		if strings.Contains(path, ".") {
			http.NotFound(w, r)
			return
		}

		fallback := *r
		fallback.URL = cloneURL(r.URL)
		fallback.URL.Path = "/"
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
