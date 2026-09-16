package api

import (
	"net/http"
	"strings"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/profiles"
)

func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	var in struct{ Name, Avatar, Locale string }
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
	if in.Locale != "" {
		var currentLocale string
		if err := s.DB.QueryRowContext(r.Context(), "SELECT locale FROM profiles WHERE id=?", session.Profile).Scan(&currentLocale); err != nil {
			return err
		}
		if in.Locale != currentLocale && (s.Locales == nil || !s.Locales.Valid(in.Locale)) {
			return badCode("profile_locale_invalid", "choose an available language")
		}
	}
	_, err := s.DB.ExecContext(
		r.Context(),
		"UPDATE profiles SET display_name=?,avatar=COALESCE(NULLIF(?,''),avatar),locale=COALESCE(NULLIF(?,''),locale) WHERE id=?",
		name, avatar, in.Locale, session.Profile,
	)
	if err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
