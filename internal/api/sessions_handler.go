package api

import (
	"net/http"
	"time"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) sessions(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	rows, e := s.DB.Rows(r.Context(), "SELECT id,created_at,last_seen,expires_at,user_agent FROM sessions WHERE profile_id=? AND expires_at>? AND last_seen>? ORDER BY last_seen DESC", session.Profile, time.Now().Unix(), time.Now().Add(-s.Config.SessionIdle).Unix())
	if e != nil {
		return e
	}
	for _, row := range rows {
		row["current"] = row["id"] == session.ID
	}
	jsonResponse(w, 200, rows)
	return nil
}
