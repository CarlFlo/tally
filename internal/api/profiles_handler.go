package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) profiles(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	rows, e := s.DB.Rows(r.Context(), "SELECT id,display_name,avatar FROM profiles ORDER BY created_at,id")
	if e != nil {
		return e
	}
	jsonResponse(w, 200, rows)
	return nil
}
