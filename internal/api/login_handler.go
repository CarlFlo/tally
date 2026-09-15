package api

import (
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) login(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	if s.Config.AuthMode != "local" {
		return bad("local sign-in is not enabled")
	}
	if session, _ := s.Auth.Resolve(r); session.Profile != "" {
		return apiError{409, "sign out before signing in to another profile"}
	}
	var in struct{ Profile, Password string }
	if e := decode(r, &in); e != nil {
		return e
	}
	if e := s.Auth.Login(r.Context(), w, r, in.Profile, in.Password); e != nil {
		return apiError{401, e.Error()}
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
