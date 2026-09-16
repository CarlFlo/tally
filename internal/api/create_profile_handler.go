package api

import (
	"errors"
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/profiles"
)

func (s *Server) createProfile(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if err := s.operator(session); err != nil {
		return err
	}
	if s.Config.AuthMode == "oidc" {
		return bad("OIDC profiles are created when their identity signs in")
	}
	var in struct{ Name, Avatar, Locale, Password string }
	if err := decode(r, &in); err != nil {
		return err
	}
	if err := profiles.ValidateName(in.Name); err != nil {
		return bad(err.Error())
	}
	if _, err := profiles.NormalizeAvatar(in.Avatar); err != nil {
		return bad(err.Error())
	}
	if in.Locale == "" {
		in.Locale = "en"
	}
	if s.Locales == nil || !s.Locales.Valid(in.Locale) {
		return bad("choose an available language")
	}
	hash := ""
	if s.Config.AuthMode == "local" && in.Password != "" {
		var err error
		hash, err = s.Auth.HashPassword(r.Context(), in.Password)
		if err != nil {
			return bad(err.Error())
		}
	}
	profile, err := (profiles.Repository{DB: s.DB, Limit: s.Config.MaxProfiles}).Create(r.Context(), in.Name, in.Avatar, in.Locale, hash, session.Profile)
	if errors.Is(err, profiles.ErrLimit) {
		return bad(err.Error())
	}
	if err != nil {
		return err
	}
	jsonResponse(w, 201, profile)
	return nil
}
