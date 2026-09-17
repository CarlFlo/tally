package api

import (
	"context"

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
	return s.Auth.Reauthenticate(ctx, profile, password)
}
