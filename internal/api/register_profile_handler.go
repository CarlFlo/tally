package api

import (
	"errors"
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/profiles"
)

func (s *Server) registerProfile(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	if session, _ := s.Auth.Resolve(r); session.Profile != "" {
		return apiError{409, "sign out before creating a profile here"}
	}
	var in struct {
		Name       string `json:"name"`
		Avatar     string `json:"avatar"`
		Locale     string `json:"locale"`
		Password   string `json:"password"`
		AuthMethod string `json:"auth_method"`
	}
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
		return badCode("profile_locale_invalid", "choose an available language")
	}
	if in.AuthMethod == "" {
		if in.Password == "" {
			in.AuthMethod = auth.ProfileAuthNone
		} else {
			in.AuthMethod = auth.ProfileAuthPassword
		}
	}
	if in.AuthMethod != auth.ProfileAuthPassword && in.AuthMethod != auth.ProfileAuthNone {
		return bad("choose Password or No authentication")
	}
	hash := ""
	if in.AuthMethod == auth.ProfileAuthPassword {
		if in.Password == "" {
			return bad("password is required")
		}
		if err := s.Auth.Policy(in.Password); err != nil {
			return bad(err.Error())
		}
		var err error
		hash, err = s.Auth.HashPassword(r.Context(), in.Password)
		if err != nil {
			return err
		}
	}
	profile, err := (profiles.Repository{DB: s.DB, Limit: s.Config.MaxProfiles}).Create(r.Context(), in.Name, in.Avatar, in.Locale, hash, "")
	if errors.Is(err, profiles.ErrLimit) {
		return bad(err.Error())
	}
	if err != nil {
		return err
	}
	if err = s.Auth.NewSession(r.Context(), w, r, profile.ID, false); err != nil {
		return err
	}
	jsonResponse(w, 201, profile)
	return nil
}
