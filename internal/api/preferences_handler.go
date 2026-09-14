package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) readPreferences(r *http.Request, id string) map[string]any {
	p := map[string]any{"theme": s.Config.Theme, "timezone": s.Config.Timezone, "date_format": "d MMM yyyy", "time_format": "24h", "calendar_view": "month", "week_start": 1, "debug_mode": false, "debug_job_state": "normal", "request_limit": 20, "scan_limit": 20, "job_type_filter": "all", "job_status_filter": "all", "bell_categories": []string{"scheduled_job_failures", "backup_failures", "episode_releases", "provider_api_failures", "torrent_client_failures"}}
	var raw string
	if s.DB.QueryRowContext(r.Context(), "SELECT data FROM profile_preferences WHERE profile_id=?", id).Scan(&raw) == nil {
		_ = json.Unmarshal([]byte(raw), &p)
	}
	delete(p, "torrent_providers")
	return p
}

func (s *Server) preferences(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	var in map[string]any
	if e := decode(r, &in); e != nil {
		return e
	}
	p := s.readPreferences(r, session.Profile)
	for k, v := range in {
		switch k {
		case "theme":
			if v != "system" && v != "light" && v != "dark" {
				return bad("invalid theme")
			}
		case "timezone":
			tz, ok := v.(string)
			if !ok {
				return bad("invalid timezone")
			}
			if _, e := time.LoadLocation(tz); e != nil {
				return bad("use an IANA timezone")
			}
		case "calendar_view":
			if v != "month" && v != "week" && v != "agenda" {
				return bad("invalid calendar view")
			}
		case "week_start":
			if v != float64(0) && v != float64(1) {
				return bad("week starts on Sunday or Monday")
			}
		case "time_format":
			if v != "12h" && v != "24h" {
				return bad("invalid time format")
			}
		case "date_format":
			if v != "d MMM yyyy" && v != "yyyy-MM-dd" && v != "MM/dd/yyyy" {
				return bad("invalid date format")
			}
		case "debug_mode":
			if _, ok := v.(bool); !ok {
				return bad("invalid debug mode")
			}
		case "debug_job_state":
			if v != "normal" && v != "failed" && v != "paused" && v != "disabled" {
				return bad("invalid debug preview")
			}
		case "request_limit", "scan_limit":
			if v != float64(20) && v != float64(50) && v != float64(100) {
				return bad("choose 20, 50, or 100 rows")
			}
		case "job_type_filter":
			if v != "all" && v != "metadata" && v != "maintenance" && v != "backup" {
				return bad("invalid job filter")
			}
		case "job_status_filter":
			if v != "all" && v != "success" && v != "failed" && v != "running" && v != "cancelled" && v != "interrupted" {
				return bad("invalid status filter")
			}
		case "bell_categories":
			bellCategories, ok := v.([]any)
			if !ok || len(bellCategories) > 7 {
				return bad("invalid bell notification categories")
			}
			allowed := map[string]bool{"scheduled_job_failures": true, "backup_failures": true, "backup_successes": true, "episode_releases": true, "provider_api_failures": true, "torrent_client_failures": true, "routine_background": true}
			seen := map[string]bool{}
			for _, category := range bellCategories {
				name, ok := category.(string)
				if !ok || !allowed[name] || seen[name] {
					return bad("invalid bell notification category")
				}
				seen[name] = true
			}
		default:
			return bad("unknown preference")
		}
		p[k] = v
	}
	data, _ := json.Marshal(in)
	if _, e := s.DB.ExecContext(r.Context(), "INSERT INTO profile_preferences VALUES(?,?) ON CONFLICT(profile_id) DO UPDATE SET data=json_patch(profile_preferences.data,excluded.data)", session.Profile, string(data)); e != nil {
		return e
	}
	jsonResponse(w, 200, p)
	return nil
}
