// Penyaji SPA hasil build frontend (fallback ke index.html).
package server

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

func (s *Server) spa() http.Handler {
	if s.Static == nil {
		return http.NotFoundHandler()
	}
	files := http.FileServerFS(s.Static)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(s.Static, p); err != nil {
			if _, ierr := fs.Stat(s.Static, "index.html"); ierr != nil {
				http.Error(w, "frontend belum dibuild (jalankan: make web)", http.StatusNotFound)
				return
			}
			r.URL.Path = "/" // fallback SPA
		}
		if p == "sw.js" {
			w.Header().Set("Cache-Control", "no-cache")
		}
		files.ServeHTTP(w, r)
	})
}
