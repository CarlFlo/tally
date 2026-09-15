package api

import "github.com/CarlFlo/mediaManager/internal/auth"

func (s *Server) operator(session auth.Session) error {
	if !session.Admin {
		return apiError{403, "this operation is reserved for administrators"}
	}
	return nil
}
