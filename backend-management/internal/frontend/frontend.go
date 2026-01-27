package frontend

import (
	"io/fs"
	"net/http"
	"os"
	"strings"
)

// Frontend file systems - populated by embed_prod.go in production builds
var (
	PlanFS fs.FS
	ZeitFS fs.FS
)

// PlanHandler returns an http.Handler that serves the plan frontend
func PlanHandler() http.Handler {
	if PlanFS == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Frontend not embedded - use EMBED_FRONTEND=true during build", http.StatusNotFound)
		})
	}
	return spaHandler(PlanFS, "/plan")
}

// ZeitHandler returns an http.Handler that serves the zeit frontend
func ZeitHandler() http.Handler {
	if ZeitFS == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Frontend not embedded - use EMBED_FRONTEND=true during build", http.StatusNotFound)
		})
	}
	return spaHandler(ZeitFS, "/zeit")
}

// spaHandler wraps a file server to handle SPA routing
// All requests that don't match a file are served index.html
func spaHandler(fsys fs.FS, basePath string) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip the base path prefix
		path := strings.TrimPrefix(r.URL.Path, basePath)
		if path == "" || path == "/" {
			path = "index.html"
		} else {
			path = strings.TrimPrefix(path, "/")
		}

		// Try to open the file
		f, err := fsys.Open(path)
		if err != nil {
			// File not found - serve index.html for SPA routing
			if os.IsNotExist(err) {
				path = "index.html"
				r.URL.Path = "/" + path
				fileServer.ServeHTTP(w, r)
				return
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		f.Close()

		// File exists - serve it
		r.URL.Path = "/" + path
		fileServer.ServeHTTP(w, r)
	})
}
