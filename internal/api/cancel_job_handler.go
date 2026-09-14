package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) cancelJob(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if e := s.operator(session); e != nil {
		return e
	}
	if !s.Jobs.Cancel(r.PathValue("id")) {
		return bad("job is no longer running")
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
