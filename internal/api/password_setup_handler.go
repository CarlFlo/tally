package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) setupPassword(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	if s.Config.AuthMode != "local" {
		return bad("local sign-in is not enabled")
	}
	if session, _ := s.Auth.Resolve(r); session.Profile != "" {
		return apiError{409, "sign out before setting up another profile"}
	}
	var in struct{ Profile, Password string }
	if err := decode(r, &in); err != nil {
		return err
	}
	if len(in.Profile) > 64 {
		return bad("invalid profile")
	}
	if err := s.Auth.Setup(r.Context(), w, r, in.Profile, in.Password); err != nil {
		return bad(err.Error())
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
