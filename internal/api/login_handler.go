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
	if e := decode(r, &in); e != nil {
		return e
	}
	var method string
	if err := s.DB.QueryRowContext(r.Context(), "SELECT auth_method FROM profiles WHERE id=?", in.Profile).Scan(&method); err != nil || method != auth.ProfileAuthPassword {
		return apiError{401, "password sign-in is not available for this profile"}
	}
	if e := s.Auth.Login(r.Context(), w, r, in.Profile, in.Password); e != nil {
		return apiError{401, e.Error()}
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
