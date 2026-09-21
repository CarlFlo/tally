package api

import (
	"context"
	"database/sql"
	"errors"
)

func (s *Server) follows(ctx context.Context, profile, show string) (bool, error) {
	var exists int
	err := s.DB.QueryRowContext(ctx, "SELECT 1 FROM profile_shows WHERE profile_id=? AND show_id=?", profile, show).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Server) requireFollow(ctx context.Context, profile, show string) error {
	follows, err := s.follows(ctx, profile, show)
	if err != nil {
		return err
	}
	if !follows {
		return apiError{404, "show is not in your library"}
	}
	return nil
}
