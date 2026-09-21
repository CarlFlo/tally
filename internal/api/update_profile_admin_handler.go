package api

import (
	"errors"
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/profiles"
)

func (s *Server) updateProfileAdmin(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if err := s.operator(session); err != nil {
		return err
	}
	id := r.PathValue("id")
	var in struct {
		Admin    bool   `json:"is_admin"`
		Password string `json:"password"`
	}
	if err := decode(r, &in); err != nil {
		return err
	}

	current, err := s.profileIsAdmin(r.Context(), id)
	if err != nil {
		return err
	}
	if current == in.Admin {
		jsonResponse(w, 200, map[string]bool{"ok": true, "is_admin": current})
		return nil
	}
	if current && !in.Admin {
		if err := s.reauthenticateIfProtected(r.Context(), session.Profile, in.Password); err != nil {
			return err
		}
	}

	err = (profiles.Repository{DB: s.DB, Limit: s.Config.MaxProfiles}).SetAdmin(r.Context(), session.Profile, id, in.Admin)
	if errors.Is(err, profiles.ErrNotFound) {
		return apiError{404, err.Error()}
	}
	if errors.Is(err, profiles.ErrLastAdmin) {
		return bad(err.Error())
	}
	if err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]bool{"ok": true, "is_admin": in.Admin})
	return nil
}
