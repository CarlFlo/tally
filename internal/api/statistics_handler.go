package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) statistics(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	summary, e := s.DB.Rows(r.Context(), `SELECT provider,SUM(requests) AS requests,SUM(successes) AS successes,SUM(failures) AS failures,SUM(retries) AS retries,SUM(rate_limited) AS rate_limited,SUM(cache_hits) AS cache_hits,SUM(conditional_hits) AS conditional_hits,SUM(avoided) AS avoided,CASE WHEN SUM(requests)>0 THEN SUM(latency_ms)/SUM(requests) ELSE 0 END AS average_latency FROM provider_request_aggregates WHERE day>=date('now','-30 days') GROUP BY provider`)
	if e != nil {
		return e
	}
	daily, e := s.DB.Rows(r.Context(), "SELECT day,SUM(requests) AS requests,SUM(avoided) AS avoided FROM provider_request_aggregates WHERE day>=date('now','-30 days') GROUP BY day ORDER BY day")
	if e != nil {
		return e
	}
	states, e := s.DB.Rows(r.Context(), "SELECT * FROM provider_state ORDER BY provider")
	if e != nil {
		return e
	}
	requestLimit, e := rowLimit(r.URL.Query().Get("request_limit"))
	if e != nil {
		return e
	}
	scanLimit, e := rowLimit(r.URL.Query().Get("scan_limit"))
	if e != nil {
		return e
	}
	requests, e := s.DB.Rows(r.Context(), "SELECT * FROM provider_requests ORDER BY id DESC LIMIT ?", requestLimit)
	if e != nil {
		return e
	}
	due, e := s.DB.Rows(r.Context(), "SELECT id,name,next_check_at FROM shows WHERE EXISTS(SELECT 1 FROM profile_shows f WHERE f.show_id=shows.id) ORDER BY next_check_at LIMIT ?", scanLimit)
	if e != nil {
		return e
	}
	jsonResponse(w, 200, map[string]any{"summary": summary, "daily": daily, "states": states, "requests": requests, "next_scans": due})
	return nil
}
