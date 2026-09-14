package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) dismissAlert(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if e := s.operator(session); e != nil {
		return e
	}
	if _, e := s.DB.ExecContext(r.Context(), "UPDATE alerts SET active=0 WHERE id=?", r.PathValue("id")); e != nil {
		return e
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
