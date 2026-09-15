package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) jobs(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	schedule, e := s.DB.Rows(r.Context(), `SELECT j.*,
(SELECT COUNT(*) FROM job_runs r WHERE (r.job_key=j.key OR r.job_key LIKE j.key||':%') AND started_at>=unixepoch()-604800 AND status='success') AS successes_7d,
(SELECT COUNT(*) FROM job_runs r WHERE (r.job_key=j.key OR r.job_key LIKE j.key||':%') AND started_at>=unixepoch()-604800 AND status='failed') AS failures_7d,
COALESCE((SELECT status FROM job_runs r WHERE r.job_key=j.key ORDER BY started_at DESC,id DESC LIMIT 1),'idle') AS last_status
FROM jobs j ORDER BY key`)
	if e != nil {
		return e
	}
	addSchedulePreviews(schedule, s.Config.Timezone)
	kind, status := r.URL.Query().Get("kind"), r.URL.Query().Get("status")
	if kind != "" && kind != "all" && kind != "metadata" && kind != "maintenance" && kind != "backup" {
		return bad("invalid job filter")
	}
	if status != "" && status != "all" && status != "success" && status != "failed" && status != "running" && status != "cancelled" && status != "interrupted" {
		return bad("invalid status filter")
	}
	query := "SELECT r.*,COALESCE(s.name,'') AS show_name FROM job_runs r LEFT JOIN shows s ON r.job_key='metadata:tvmaze:show:'||s.id WHERE 1=1"
	args := []any{}
	if kind != "" && kind != "all" {
		query += " AND (job_key=? OR job_key LIKE ?)"
		args = append(args, kind, kind+":%")
	}
	if status != "" && status != "all" {
		query += " AND r.status=?"
		args = append(args, status)
	}
	query += " ORDER BY started_at DESC,r.id DESC LIMIT 100"
	runs, e := s.DB.Rows(r.Context(), query, args...)
	if e != nil {
		return e
	}
	jsonResponse(w, 200, map[string]any{"schedules": schedule, "runs": runs})
	return nil
}
