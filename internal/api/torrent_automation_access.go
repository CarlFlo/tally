package api

import (
	"context"
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) requireTorrentShowAccess(ctx context.Context, session auth.Session, showID string) error {
	if !session.Admin {
		return s.requireFollow(ctx, session.Profile, showID)
	}
	var exists int
	if err := s.DB.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM shows WHERE id=?)", showID).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return apiError{http.StatusNotFound, "show not found"}
	}
	return nil
}
