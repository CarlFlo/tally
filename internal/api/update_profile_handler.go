package api

import (
	"github.com/CarlFlo/mediaManager/internal/profiles"
	"net/http"
	"strings"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	var in struct{ Name, Avatar string }
	if e := decode(r, &in); e != nil {
		return e
	}
	name := strings.TrimSpace(in.Name)
	if err := profiles.ValidateName(in.Name); err != nil {
		return bad(err.Error())
	}
	if in.Avatar != "" && !builtinAvatar(in.Avatar) {
		return bad("choose a built-in avatar or upload an image")
	}
	var e error
	if in.Avatar == "" {
		_, e = s.DB.ExecContext(r.Context(), "UPDATE profiles SET display_name=? WHERE id=?", name, session.Profile)
	} else {
		_, e = s.DB.ExecContext(r.Context(), "UPDATE profiles SET display_name=?,avatar=? WHERE id=?", name, in.Avatar, session.Profile)
	}
	if e != nil {
		return e
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
