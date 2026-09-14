package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) alerts(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	rows, e := s.DB.Rows(r.Context(), "SELECT * FROM alerts WHERE active=1 ORDER BY updated_at DESC LIMIT 50")
	if e != nil {
		return e
	}
	jsonResponse(w, 200, rows)
	return nil
}
