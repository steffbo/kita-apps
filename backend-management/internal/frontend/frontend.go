package frontend

import (
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
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
			http.Error(w, "Frontend not embedded - use -tags embed_frontend during build", http.StatusNotFound)
		})
	}
	return spaHandler(PlanFS)
}

// ZeitHandler returns an http.Handler that serves the zeit frontend
func ZeitHandler() http.Handler {
	if ZeitFS == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Frontend not embedded - use -tags embed_frontend during build", http.StatusNotFound)
		})
	}
	return spaHandler(ZeitFS)
}

// DebugHandler returns a handler that lists all files in the plan FS
func DebugHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		if PlanFS == nil {
			fmt.Fprintln(w, "PlanFS is nil")
			return
		}
		fmt.Fprintln(w, "Files in PlanFS:")
		fs.WalkDir(PlanFS, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				fmt.Fprintf(w, "ERROR: %s: %v\n", path, err)
				return nil
			}
			if d.IsDir() {
				fmt.Fprintf(w, "[DIR]  %s\n", path)
			} else {
				fmt.Fprintf(w, "[FILE] %s\n", path)
			}
			return nil
		})
	})
}

// spaHandler serves files from the embedded FS with SPA fallback to index.html
// Note: chi.Mount strips the prefix, so r.URL.Path is already relative (e.g., "/" or "/assets/foo.js")
func spaHandler(fsys fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the path without leading slash
		path := strings.TrimPrefix(r.URL.Path, "/")

		// For root path or empty path, serve index.html
		if path == "" {
			serveFile(w, r, fsys, "index.html")
			return
		}

		// Try to open the requested file
		f, err := fsys.Open(path)
		if err != nil {
			// Log the error for debugging
			println("DEBUG: fsys.Open failed for path:", path, "error:", err.Error())
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
