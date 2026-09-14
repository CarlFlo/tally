package api

import (
	"net/http"
)

func (s *Server) follows(r *http.Request, profile, show string) bool {
	var exists int
	return s.DB.QueryRowContext(r.Context(), "SELECT 1 FROM profile_shows WHERE profile_id=? AND show_id=?", profile, show).Scan(&exists) == nil
}
