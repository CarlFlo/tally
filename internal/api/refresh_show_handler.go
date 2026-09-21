package api

import (
	"net/http"

	"github.com/CarlFlo/tally/internal/activity"
	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) refreshShow(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if err := s.requireFollow(r.Context(), session.Profile, r.PathValue("id")); err != nil {
		return err
	}
	id, e := s.Jobs.Trigger("metadata", "manual_refresh", r.PathValue("id"))
	if e != nil {
		return jobRequestError(e)
	}
	var name string
	if e = s.DB.QueryRowContext(r.Context(), "SELECT name FROM shows WHERE id=?", r.PathValue("id")).Scan(&name); e != nil {
		return e
	}
	if e = activity.Record(r.Context(), s.DB, activity.Event{Action: "show_refresh_requested", Profile: session.Profile, ShowID: r.PathValue("id"), ShowName: name, Message: "Requested metadata refresh for " + name}); e != nil {
		return e
	}
	jsonResponse(w, 202, map[string]string{"job_id": id})
	return nil
}
