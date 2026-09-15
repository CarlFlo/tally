package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) createManualBackup(w http.ResponseWriter, _ *http.Request, session auth.Session) error {
	if err := s.operator(session); err != nil {
		return err
	}
	id, err := s.Jobs.Trigger("backup", "manual_backup", "")
	if err != nil {
		return bad(err.Error())
	}
	jsonResponse(w, http.StatusAccepted, map[string]string{"id": id})
	return nil
}
