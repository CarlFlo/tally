package api

import (
	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) operator(session auth.Session) error {
	if session.Profile != "user0" {
		return apiError{403, "this operation is reserved for the deployment's user0 profile"}
	}
	return nil
}
