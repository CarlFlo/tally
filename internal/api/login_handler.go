package api

import (
	"errors"
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) login(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	session, err := s.optionalSession(r)
	if err != nil {
		return err
	}
	if session.Profile != "" {
		return apiError{409, "sign out before signing in to another profile"}
	}
	var in struct{ Profile, Password string }
	if err := decode(r, &in); err != nil {
		return err
	}
	method, err := s.Auth.ProfileAuthMethod(r.Context(), in.Profile)
	if errors.Is(err, auth.ErrProfileNotFound) || (err == nil && method != auth.ProfileAuthPassword) {
		return apiError{401, "password sign-in is not available for this profile"}
	}
	if err != nil {
		return err
	}
	if err = s.Auth.Login(r.Context(), w, r, in.Profile, in.Password); err != nil {
		if errors.Is(err, auth.ErrIncorrectPassword) || errors.Is(err, auth.ErrPasswordTooLong) || errors.Is(err, auth.ErrAuthenticationThrottled) {
			return apiError{401, err.Error()}
		}
		return err
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
