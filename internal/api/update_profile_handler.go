package api

import (
	"net/http"
	"strings"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/profiles"
)

func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	var in struct{ Name, Avatar string }
	if err := decode(r, &in); err != nil {
		return err
	}
	name := strings.TrimSpace(in.Name)
	if err := profiles.ValidateName(name); err != nil {
		return bad(err.Error())
	}
	avatar := in.Avatar
	if avatar != "" {
		var err error
		avatar, err = profiles.NormalizeAvatar(avatar)
		if err != nil {
			return bad(err.Error())
		}
	}
	var err error
	if avatar == "" {
		_, err = s.DB.ExecContext(r.Context(), "UPDATE profiles SET display_name=? WHERE id=?", name, session.Profile)
	} else {
		_, err = s.DB.ExecContext(r.Context(), "UPDATE profiles SET display_name=?,avatar=? WHERE id=?", name, avatar, session.Profile)
	}
	if err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
