package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) logout(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	// Logout is idempotent, including after session expiry or another tab signing out.
	if cookie, e := r.Cookie("tally_session"); e == nil {
		if _, e := s.DB.ExecContext(r.Context(), "DELETE FROM sessions WHERE id=?", auth.Digest(cookie.Value)); e != nil {
			return e
		}
	}
	s.Auth.Cookie(w, r, "tally_session", "", -1)
	if s.Config.AuthMode == "disabled" {
		// Retain an explicit choice to leave, even on a single-profile installation.
		s.Auth.Cookie(w, r, "tally_profile", "signed-out", 365*24*3600)
	} else {
		s.Auth.Cookie(w, r, "tally_profile", "", -1)
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
