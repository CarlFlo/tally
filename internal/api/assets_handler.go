package api

import (
	"io/fs"
	"log/slog"
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
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Add("Vary", "Cookie")
	page = []byte(strings.Replace(string(page), `data-theme="__TALLY_THEME__"`, `data-theme="`+s.requestTheme(r)+`"`, 1))
	_, _ = w.Write(page)
}

func (s *Server) requestTheme(r *http.Request) string {
	theme := s.browserTheme(r)
	if session, err := s.optionalSession(r); err == nil && session.Profile != "" {
		if preferences, prefErr := s.readPreferences(r, session.Profile); prefErr == nil {
			if value, ok := preferences["theme"].(string); ok {
				theme = value
			}
		} else {
			slog.Warn("read profile theme for first paint", "profile_id", session.Profile, "error", prefErr)
		}
	} else if err != nil {
		slog.Warn("resolve session for first-paint theme", "error", err)
	}
	if theme != "light" && theme != "dark" && theme != "system" {
		return "system"
	}
	return theme
}
