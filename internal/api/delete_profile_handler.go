package api

import (
	"errors"
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/profiles"
)

func (s *Server) deleteProfile(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	id := r.PathValue("id")
	if !session.Admin && id != session.Profile {
		return apiError{403, "you can only delete your own profile"}
	}

	var targetAdmin bool
	if err := s.DB.QueryRowContext(r.Context(), "SELECT is_admin FROM profile_roles WHERE profile_id=?", id).Scan(&targetAdmin); err != nil {
		return apiError{404, "profile not found"}
	}

	var in struct{ Password string }
	if r.ContentLength > 0 {
		if err := decode(r, &in); err != nil {
			return err
		}
	}
	if targetAdmin {
		if err := s.reauthenticateIfProtected(r.Context(), session.Profile, in.Password); err != nil {
			return apiError{401, err.Error()}
		}
	}

	_, _, err := (profiles.Repository{DB: s.DB, Limit: s.Config.MaxProfiles}).Delete(r.Context(), session.Profile, id)
	if errors.Is(err, profiles.ErrNotFound) {
		return apiError{404, err.Error()}
	}
	if errors.Is(err, profiles.ErrLastAdmin) {
		return bad(err.Error())
	}
	if err != nil {
		return err
	}
	if id == session.Profile {
		return s.logout(w, r, session)
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
