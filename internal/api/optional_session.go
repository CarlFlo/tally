package api

import (
	"errors"
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) optionalSession(r *http.Request) (auth.Session, error) {
	session, err := s.Auth.Resolve(r)
	if err == nil {
		return session, nil
	}
	if errors.Is(err, auth.ErrSignInRequired) || errors.Is(err, auth.ErrSessionExpired) {
		return auth.Session{}, nil
	}
	return auth.Session{}, err
}
