package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/activity"
	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) deleteProfile(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	id := r.PathValue("id")
	if id == "user0" {
		return bad("the administrator account is permanent and cannot be deleted")
	}
	if session.Profile != "user0" && id != session.Profile {
		return apiError{403, "you can only delete your own profile"}
	}
	tx, e := s.DB.BeginTx(r.Context(), nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var name string
	if e = tx.QueryRowContext(r.Context(), "SELECT display_name FROM profiles WHERE id=?", id).Scan(&name); e != nil {
		return apiError{404, "profile not found"}
	}
	res, e := tx.ExecContext(r.Context(), "DELETE FROM profiles WHERE id=?", id)
	if e != nil {
		return e
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apiError{404, "profile not found"}
	}
	if e = activity.Record(r.Context(), tx, activity.Event{Action: "profile_deleted", Profile: session.Profile, Message: "Deleted profile " + name}); e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	if id == session.Profile {
		return s.logout(w, r, session)
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
