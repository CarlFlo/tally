package api

import (
	"context"
	"errors"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) reauthenticateIfProtected(ctx context.Context, profile, password string) error {
	method, err := s.Auth.ProfileAuthMethod(ctx, profile)
	if err != nil {
		return err
	}
	if method != auth.ProfileAuthPassword {
		return nil
	}
	if err = s.Auth.Reauthenticate(ctx, profile, password); err != nil {
		if errors.Is(err, auth.ErrIncorrectPassword) || errors.Is(err, auth.ErrAuthenticationThrottled) {
			return apiError{401, err.Error()}
		}
		return err
	}
	return nil
}
