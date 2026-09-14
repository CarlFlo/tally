package api

import (
	"errors"
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
	"github.com/CarlFlo/mediaManager/internal/profiles"
)

func (s *Server) registerProfile(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	if s.Config.AuthMode == "oidc" {
		return bad("create your profile through single sign-on")
	}
	if session, _ := s.Auth.Resolve(r); session.Profile != "" {
		return apiError{409, "sign out before creating a profile here"}
	}
	var in struct{ Name, Avatar, Password string }
	if err := decode(r, &in); err != nil {
		return err
	}
	if err := profiles.ValidateName(in.Name); err != nil {
		return bad(err.Error())
	}
	if !builtinAvatar(in.Avatar) {
		in.Avatar = "violet"
	}
	hash := ""
	if s.Config.AuthMode == "local" {
		var err error
		hash, err = s.Auth.HashPassword(r.Context(), in.Password)
		if err != nil {
			return bad(err.Error())
		}
	}
	profile, err := (profiles.Repository{DB: s.DB, Limit: s.Config.MaxProfiles}).Create(r.Context(), in.Name, in.Avatar, hash, "")
	if errors.Is(err, profiles.ErrLimit) {
		return bad(err.Error())
	}
	if err != nil {
		return err
	}
	if s.Config.AuthMode == "local" {
		if err = s.Auth.NewSession(r.Context(), w, r, profile.ID, false); err != nil {
			return err
		}
	} else {
		s.Auth.Cookie(w, r, "tally_profile", profile.ID, 365*24*3600)
	}
	jsonResponse(w, 201, profile)
	return nil
}
