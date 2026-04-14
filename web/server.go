// Package web provides the HTTP server for the db-diff web GUI.
package web

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
)

//go:embed all:static
var embeddedStatic embed.FS

// Server holds the HTTP server configuration.
type Server struct {
	Port int
}

// ListenAndServe starts the HTTP server on the configured port.
func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()

	// Serve static files from the embedded FS (or a local override for dev).
	staticFS := resolveStaticFS()
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	addr := fmt.Sprintf(":%d", s.Port)
	log.Printf("db-diff web server listening on http://localhost%s\n", addr)

	return http.ListenAndServe(addr, mux)
}

// resolveStaticFS returns an FS that serves static assets.
//
// If the environment variable DB_DIFF_STATIC_DIR is set (or ./web/static
// exists relative to the working directory), that directory is used directly
// — handy during development without rebuilding the binary.
// Otherwise the embedded FS is used.
func resolveStaticFS() fs.FS {
	// Allow an explicit override via environment variable.
	if dir := os.Getenv("DB_DIFF_STATIC_DIR"); dir != "" {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return os.DirFS(dir)
		}
	}

	// Sub into the "static" directory of the embedded FS.
	sub, err := fs.Sub(embeddedStatic, "static")
	if err != nil {
		// Should never happen — the "static" directory is always present.
		panic(fmt.Sprintf("web: failed to sub embedded static FS: %v", err))
	}
	return sub
}

// IsSPARequest returns true for paths that should fall back to index.html
// (all paths that don't look like asset requests).
func isSPARequest(path string) bool {
	return !strings.Contains(path, ".")
}
