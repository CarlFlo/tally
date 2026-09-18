package api

import (
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) createManualBackup(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if err := s.operator(session); err != nil {
		return err
	}
	id, err := s.Jobs.TriggerAndWait(r.Context(), "backup", "manual_backup", "")
	if err != nil {
		return bad(err.Error())
	}
	jsonResponse(w, http.StatusCreated, map[string]string{"id": id})
	return nil
}
