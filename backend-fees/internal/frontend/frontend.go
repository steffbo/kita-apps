package frontend

import (
	"io/fs"
	"net/http"
	"os"
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

// spaHandler wraps a file server to handle SPA routing
// All requests that don't match a file are served index.html
// Note: chi.Mount strips the prefix, so r.URL.Path is already relative
func spaHandler(fsys fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the path (chi.Mount already stripped the prefix)
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Try to open the file
		f, err := fsys.Open(path)
		if err != nil {
			// File not found - serve index.html for SPA routing
			if os.IsNotExist(err) {
				serveFile(w, r, fsys, "index.html")
				return
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer f.Close()

		// Check if it's a directory
		stat, err := f.Stat()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if stat.IsDir() {
			// Try index.html in the directory
			indexPath := path + "/index.html"
			serveFile(w, r, fsys, indexPath)
			return
		}

		// Serve the file
		serveFile(w, r, fsys, path)
	})
}

func serveFile(w http.ResponseWriter, r *http.Request, fsys fs.FS, path string) {
	f, err := fsys.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Fallback to index.html for SPA
			f, err = fsys.Open("index.html")
			if err != nil {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}
			path = "index.html"
		} else {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Set content type based on extension
	contentType := "application/octet-stream"
	switch {
	case strings.HasSuffix(path, ".html"):
		contentType = "text/html; charset=utf-8"
	case strings.HasSuffix(path, ".css"):
		contentType = "text/css; charset=utf-8"
	case strings.HasSuffix(path, ".js"):
		contentType = "application/javascript; charset=utf-8"
	case strings.HasSuffix(path, ".json"):
		contentType = "application/json; charset=utf-8"
	case strings.HasSuffix(path, ".png"):
		contentType = "image/png"
	case strings.HasSuffix(path, ".jpg"), strings.HasSuffix(path, ".jpeg"):
		contentType = "image/jpeg"
	case strings.HasSuffix(path, ".svg"):
		contentType = "image/svg+xml"
	case strings.HasSuffix(path, ".ico"):
		contentType = "image/x-icon"
	case strings.HasSuffix(path, ".woff"):
		contentType = "font/woff"
	case strings.HasSuffix(path, ".woff2"):
		contentType = "font/woff2"
	}
	w.Header().Set("Content-Type", contentType)

	// Read and serve file content
	content, err := fs.ReadFile(fsys, path)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.ServeContent(w, r, path, stat.ModTime(), strings.NewReader(string(content)))
}
