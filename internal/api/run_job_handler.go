package api

import (
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) runJob(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	kind := r.PathValue("kind")
	if kind == "backup" || kind == "maintenance" || kind == "torrent_automation" {
		if e := s.operator(session); e != nil {
			return e
		}
	}
	id, e := s.Jobs.Trigger(kind, "manual_refresh", "")
	if e != nil {
		return jobRequestError(e)
	}
	jsonResponse(w, 202, map[string]string{"id": id})
	return nil
}
