package api

import (
	"context"
	"database/sql"
	"errors"
)

func (s *Server) profileIsAdmin(ctx context.Context, profileID string) (bool, error) {
	var admin bool
	err := s.DB.QueryRowContext(ctx, "SELECT is_admin FROM profile_roles WHERE profile_id=?", profileID).Scan(&admin)
	if errors.Is(err, sql.ErrNoRows) {
		return false, apiError{404, "profile not found"}
	}
	if err != nil {
		return false, err
	}
	return admin, nil
}
