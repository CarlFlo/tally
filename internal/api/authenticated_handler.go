package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

type handler func(http.ResponseWriter, *http.Request, auth.Session) error

func (s *Server) wrap(fn handler, public bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var session auth.Session
		var e error
		if !public {
			session, e = s.Auth.Resolve(r)
			if e != nil {
				jsonResponse(w, 401, map[string]string{"error": e.Error()})
				return
			}
			if session.Restricted && r.URL.Path != "/api/auth/password" && r.URL.Path != "/api/auth/logout" {
				jsonResponse(w, 403, map[string]string{"error": "replace the temporary password before continuing", "code": "password_change_required"})
				return
			}
			if adminPath(r.URL.Path) && session.Profile != "user0" {
				jsonResponse(w, 403, map[string]string{"error": "Only the administrator can access this page."})
				return
			}
		}
		if e = fn(w, r, session); e != nil {
			var ae apiError
			if errors.As(e, &ae) {
				jsonResponse(w, ae.Status, map[string]string{"error": ae.Message})
			} else {
				slog.Error("request failed", "path", r.URL.Path, "error", e)
				jsonResponse(w, 500, map[string]string{"error": "The local operation failed. Check server logs for details."})
			}
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.URL.Path != "/api/settings/scheduling/preview" {
			s.Events.Publish()
		}
	}
}
