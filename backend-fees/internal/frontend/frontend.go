package frontend

import (
	"io/fs"
	"net/http"
	"strings"
)

// Frontend file system - populated by embed_prod.go in production builds
var BeitraegeFS fs.FS

// BeitraegeHandler returns an http.Handler that serves the beitraege frontend
func BeitraegeHandler() http.Handler {
	if BeitraegeFS == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Frontend not embedded - use -tags embed_frontend during build", http.StatusNotFound)
		})
	}
	return spaHandler(BeitraegeFS)
}

// spaHandler serves files from the embedded FS with SPA fallback to index.html
// Note: chi.Mount strips the prefix, so r.URL.Path is already relative (e.g., "/" or "/assets/foo.js")
func spaHandler(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the path without leading slash
		path := strings.TrimPrefix(r.URL.Path, "/")

		// Try to open the requested file
		if path != "" {
			f, err := fsys.Open(path)
			if err == nil {
				f.Close()
				// File exists, serve it
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// File doesn't exist or path is empty - serve index.html for SPA routing
		// Check if index.html exists
		f, err := fsys.Open("index.html")
		if err != nil {
			http.Error(w, "index.html not found in embedded filesystem", http.StatusNotFound)
			return
		}
		f.Close()

		// Rewrite the request to serve index.html
		r.URL.Path = "/index.html"
		fileServer.ServeHTTP(w, r)
	})
}
