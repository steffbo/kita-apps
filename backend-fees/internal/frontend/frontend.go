package frontend

import (
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
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
// chi.Mount sets up the route context but doesn't modify r.URL.Path, so we use chi's RoutePath
func spaHandler(fsys fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the path from chi's route context (this is the path after the mount prefix)
		rctx := chi.RouteContext(r.Context())
		path := rctx.RoutePath
		if path == "" {
			path = r.URL.Path
		}
		// Remove leading slash
		path = strings.TrimPrefix(path, "/")

		// For root path or empty path, serve index.html
		if path == "" {
			serveFile(w, r, fsys, "index.html")
			return
		}

		// Try to open the requested file
		f, err := fsys.Open(path)
		if err != nil {
			// File not found - serve index.html for SPA routing
			serveFile(w, r, fsys, "index.html")
			return
		}
		defer f.Close()

		// Check if it's a directory
		stat, err := f.Stat()
		if err != nil {
			serveFile(w, r, fsys, "index.html")
			return
		}

		if stat.IsDir() {
			// Try to serve index.html from the directory
			indexPath := strings.TrimSuffix(path, "/") + "/index.html"
			if indexFile, err := fsys.Open(indexPath); err == nil {
				indexFile.Close()
				serveFile(w, r, fsys, indexPath)
				return
			}
			// Fall back to root index.html for SPA routing
			serveFile(w, r, fsys, "index.html")
			return
		}

		// Serve the actual file
		serveFile(w, r, fsys, path)
	})
}

func serveFile(w http.ResponseWriter, r *http.Request, fsys fs.FS, path string) {
	f, err := fsys.Open(path)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Set content type based on extension
	contentType := mime.TypeByExtension(filepath.Ext(path))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)

	// Use http.ServeContent for proper handling of Range requests, etc.
	readSeeker, ok := f.(io.ReadSeeker)
	if ok {
		http.ServeContent(w, r, path, stat.ModTime(), readSeeker)
	} else {
		// Fallback: read all and write
		content, err := io.ReadAll(f)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Write(content)
	}
}
