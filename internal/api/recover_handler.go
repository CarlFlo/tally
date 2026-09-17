package api

import (
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) recover(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	var in struct{ Profile string }
	if err := decode(r, &in); err != nil {
		return err
	}
	method, err := s.Auth.ProfileAuthMethod(r.Context(), in.Profile)
	if err != nil || method != auth.ProfileAuthPassword {
		return bad("password recovery is not available for this profile")
	}
	if err = s.Auth.Recover(r.Context(), in.Profile); err != nil {
		return apiError{429, err.Error()}
	}
	jsonResponse(w, 200, map[string]string{"message": "Temporary password written to the server logs. It expires in 30 minutes."})
	return nil
}
