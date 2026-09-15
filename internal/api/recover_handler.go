package api

import (
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) recover(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	if s.Config.AuthMode != "local" {
		return bad("local sign-in is not enabled")
	}
	var in struct{ Profile string }
	if e := decode(r, &in); e != nil {
		return e
	}
	if e := s.Auth.Recover(r.Context(), in.Profile); e != nil {
		return apiError{429, e.Error()}
	}
	jsonResponse(w, 200, map[string]string{"message": "Temporary password written to the server logs. It expires in 30 minutes."})
	return nil
}
