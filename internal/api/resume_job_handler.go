package api

import (
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) resumeJob(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if e := s.operator(session); e != nil {
		return e
	}
	if e := s.Jobs.Resume(r.Context(), r.PathValue("kind")); e != nil {
		return bad(e.Error())
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
