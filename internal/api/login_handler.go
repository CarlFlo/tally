package api

import (
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) login(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	if session, _ := s.Auth.Resolve(r); session.Profile != "" {
		return apiError{409, "sign out before signing in to another profile"}
	}
	var in struct{ Profile, Password string }
	if err := decode(r, &in); err != nil {
		return err
	}
	method, err := s.Auth.ProfileAuthMethod(r.Context(), in.Profile)
	if err != nil || method != auth.ProfileAuthPassword {
		return apiError{401, "password sign-in is not available for this profile"}
	}
	if err = s.Auth.Login(r.Context(), w, r, in.Profile, in.Password); err != nil {
		return apiError{401, err.Error()}
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
