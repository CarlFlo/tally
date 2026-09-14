package api

import (
	"io/fs"
	"net/http"
	"strings"
)

func (s *Server) assets(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "HEAD" {
		http.NotFound(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		jsonResponse(w, 404, map[string]string{"error": "endpoint not found"})
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path != "" {
		if info, e := fs.Stat(s.Assets, path); e == nil && !info.IsDir() {
			if strings.HasPrefix(path, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			http.FileServer(http.FS(s.Assets)).ServeHTTP(w, r)
			return
		}
	}
	page, e := fs.ReadFile(s.Assets, "index.html")
	if e != nil {
		http.Error(w, "Frontend assets have not been built", 503)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(page)
}
