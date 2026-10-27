// SPDX-License-Identifier: AGPL-3.0-or-later

package httpapi

import (
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// spa serves the built front end; unknown paths fall back to index.html so the
// client-side router can answer them.
func (s *Server) spa() http.Handler {
	if s.Web == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("Front end not built: run `make web` or set web_dir, or use the Vite dev server (npm run dev in web/).\n"))
		})
	}
	files := http.FileServerFS(s.Web)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name != "" {
			if info, err := fs.Stat(s.Web, name); err == nil && !info.IsDir() {
				if strings.HasPrefix(name, "assets/") {
					// Vite fingerprints these file names.
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				files.ServeHTTP(w, r)
				return
			} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFileFS(w, r, s.Web, "index.html")
	})
}
