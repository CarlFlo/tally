package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

func (s *Server) security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; font-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" {
			if r.Header.Get("X-Tally-CSRF") != "1" || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				jsonResponse(w, 403, map[string]string{"error": "request origin could not be verified"})
				return
			}
			if origin := r.Header.Get("Origin"); origin != "" {
				expected := "http://" + r.Host
				if r.TLS != nil {
					expected = "https://" + r.Host
				}
				allowed := origin == expected
				if s.Config.PublicURL != "" {
					u, _ := url.Parse(s.Config.PublicURL)
					allowed = allowed || origin == u.Scheme+"://"+u.Host
				}
				if !allowed {
					jsonResponse(w, 403, map[string]string{"error": "cross-origin request rejected"})
					return
				}
			}
		}
		r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
		defer func() {
			if v := recover(); v != nil {
				slog.Error("request panic", "path", r.URL.Path, "panic", fmt.Sprint(v))
				http.Error(w, "Internal server error", 500)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
