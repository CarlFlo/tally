package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) logs(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 200 {
		return bad("search must be 200 characters or fewer")
	}
	action := r.URL.Query().Get("action")
	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if r.URL.Query().Get("offset") == "" {
		offset, err = 0, nil
	}
	if err != nil || offset < 0 {
		return bad("invalid page")
	}
	where, args := " WHERE 1=1", []any{}
	if session.Profile != "user0" {
		where += " AND l.profile_id=?"
		args = append(args, session.Profile)
	}
	if q != "" {
		where += " AND (instr(lower(l.message),lower(?))>0 OR instr(lower(l.show_name),lower(?))>0 OR instr(lower(COALESCE(p.display_name,CASE WHEN l.profile_id='' THEN 'System' ELSE 'Deleted profile' END)),lower(?))>0)"
		args = append(args, q, q, q)
	}
	if action != "" {
		where += " AND l.action=?"
		args = append(args, action)
	}
	var total int
	join := " FROM activity_log l LEFT JOIN profiles p ON p.id=l.profile_id"
	if err = s.DB.QueryRowContext(r.Context(), "SELECT COUNT(*)"+join+where, args...).Scan(&total); err != nil {
		return err
	}
	rows, err := s.DB.Rows(r.Context(), "SELECT l.*,COALESCE(p.display_name,CASE WHEN l.profile_id='' THEN 'System' ELSE 'Deleted profile' END) AS profile_name"+join+where+" ORDER BY l.id DESC LIMIT 50 OFFSET ?", append(args, offset)...)
	if err != nil {
		return err
	}
	actionQuery, actionArgs := "SELECT DISTINCT action FROM activity_log", []any{}
	if session.Profile != "user0" {
		actionQuery += " WHERE profile_id=?"
		actionArgs = append(actionArgs, session.Profile)
	}
	actions, err := s.DB.Rows(r.Context(), actionQuery+" ORDER BY action", actionArgs...)
	if err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]any{"entries": rows, "actions": actions, "total": total})
	return nil
}
