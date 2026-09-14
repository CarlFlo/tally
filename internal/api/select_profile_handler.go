package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) selectProfile(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	if s.Config.AuthMode != "disabled" {
		return apiError{403, "sign in as that profile to switch accounts"}
	}
	var in struct{ Profile string }
	if e := decode(r, &in); e != nil {
		return e
	}
	var id string
	if session, _ := s.Auth.Resolve(r); session.Profile != "" && session.Profile != in.Profile {
		return apiError{409, "sign out before choosing another profile"}
	}
	if s.DB.QueryRowContext(r.Context(), "SELECT id FROM profiles WHERE id=?", in.Profile).Scan(&id) != nil {
		return apiError{404, "profile no longer exists"}
	}
	s.Auth.Cookie(w, r, "tally_profile", id, 365*24*3600)
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
