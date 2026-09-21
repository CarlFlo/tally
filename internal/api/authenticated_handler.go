package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
)

type handler func(http.ResponseWriter, *http.Request, auth.Session) error

type routeAccess uint8

const (
	authenticatedRoute routeAccess = iota
	publicRoute
	adminRoute
)

func (s *Server) wrap(fn handler, access routeAccess) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var session auth.Session
		var e error
		if access != publicRoute {
			session, e = s.Auth.Resolve(r)
			if e != nil {
				if errors.Is(e, auth.ErrSignInRequired) || errors.Is(e, auth.ErrSessionExpired) {
					jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": e.Error()})
				} else {
					slog.Error("session resolution failed", "path", r.URL.Path, "error", e)
					jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": internalErrorMessage})
				}
				return
			}
			if session.Restricted && r.URL.Path != "/api/auth/password" && r.URL.Path != "/api/auth/logout" {
				jsonResponse(w, http.StatusForbidden, map[string]string{"error": "replace the temporary password before continuing", "code": "password_change_required"})
				return
			}
			if access == adminRoute && !session.Admin {
				jsonResponse(w, http.StatusForbidden, map[string]string{"error": "Only administrators can access this page."})
				return
			}
		}
		if e = fn(w, r, session); e != nil {
			var remoteErr remoteAPIError
			if errors.As(e, &remoteErr) {
				slog.Warn("external request failed", "path", r.URL.Path, "error", remoteErr.Cause)
				jsonResponse(w, http.StatusBadGateway, map[string]string{"error": remoteErrorMessage})
				return
			}
			var coded codedAPIError
			if errors.As(e, &coded) {
				jsonResponse(w, coded.Status, map[string]string{"error": coded.Message, "code": coded.Code})
				return
			}
			var ae apiError
			if errors.As(e, &ae) {
				jsonResponse(w, ae.Status, map[string]string{"error": ae.Message})
			} else {
				slog.Error("request failed", "path", r.URL.Path, "error", e)
				jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": internalErrorMessage})
			}
			return
		}
		for _, live := range liveChanges(r, session) {
			s.Events.Publish(live.Profile, live.Resources...)
		}
	}
}
