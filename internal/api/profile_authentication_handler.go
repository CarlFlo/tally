package api

import (
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) updateProfileAuthentication(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if err := s.operator(session); err != nil {
		return err
	}
	id := r.PathValue("id")
	var in struct {
		Method        string `json:"method"`
		Password      string `json:"password"`
		ActorPassword string `json:"actor_password"`
	}
	if err := decode(r, &in); err != nil {
		return err
	}
	if in.Method != auth.ProfileAuthPassword && in.Method != auth.ProfileAuthNone {
		return bad("choose Password or No authentication")
	}
	actorMethod, err := s.Auth.ProfileAuthMethod(r.Context(), session.Profile)
	if err != nil {
		return err
	}
	if actorMethod == auth.ProfileAuthPassword {
		if err := s.Auth.Reauthenticate(r.Context(), session.Profile, in.ActorPassword); err != nil {
			return apiError{401, err.Error()}
		}
	}
	if err := s.Auth.SetProfileAuthentication(r.Context(), id, in.Method, in.Password); err != nil {
		if err.Error() == "profile not found" {
			return apiError{404, err.Error()}
		}
		return bad(err.Error())
	}
	if id == session.Profile {
		if err := s.Auth.NewSession(r.Context(), w, r, id, false); err != nil {
			return err
		}
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
