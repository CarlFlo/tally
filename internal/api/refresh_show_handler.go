package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/activity"
	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) refreshShow(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if !s.follows(r, session.Profile, r.PathValue("id")) {
		return apiError{404, "show is not in your library"}
	}
	id, e := s.Jobs.Trigger("metadata", "manual_refresh", r.PathValue("id"))
	if e != nil {
		return bad(e.Error())
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
