package api

import (
	"net/http"
	"time"

	"github.com/CarlFlo/mediaManager/internal/auth"
	"github.com/CarlFlo/mediaManager/internal/jobs"
)

func (s *Server) schedulePreview(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if err := s.operator(session); err != nil {
		return err
	}
	var input struct {
		Schedule string `json:"schedule"`
	}
	if err := decode(r, &input); err != nil {
		return err
	}
	preview, err := jobs.PreviewSchedule(input.Schedule, time.Now(), s.Config.Timezone)
	if err != nil {
		return bad(err.Error())
	}
	jsonResponse(w, http.StatusOK, preview)
	return nil
}
