package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) runJob(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	kind := r.PathValue("kind")
	if kind == "backup" || kind == "maintenance" {
		if e := s.operator(session); e != nil {
			return e
		}
	}
	id, e := s.Jobs.Trigger(kind, "manual_refresh", "")
	if e != nil {
		return bad(e.Error())
	}
	jsonResponse(w, 202, map[string]string{"id": id})
	return nil
}
