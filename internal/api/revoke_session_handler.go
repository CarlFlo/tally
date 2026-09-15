package api

import (
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) revokeSession(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	res, e := s.DB.ExecContext(r.Context(), "DELETE FROM sessions WHERE id=? AND profile_id=?", r.PathValue("id"), session.Profile)
	if e != nil {
		return e
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apiError{404, "session not found"}
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
