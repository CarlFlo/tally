package api

import (
	"net/http"
	"time"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) calendar(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	from, e := time.Parse("2006-01-02", r.URL.Query().Get("from"))
	if e != nil {
		return bad("from must be YYYY-MM-DD")
	}
	to, e := time.Parse("2006-01-02", r.URL.Query().Get("to"))
	if e != nil || !to.After(from) || to.Sub(from) > 100*24*time.Hour {
		return bad("choose a calendar range of 1–100 days")
	}
	rows, e := s.DB.Rows(r.Context(), episodeSelect+"WHERE e.airdate>=? AND e.airdate<? ORDER BY e.airdate,e.airstamp,s.name,e.season,e.number", session.Profile, from.AddDate(0, 0, -1).Format("2006-01-02"), to.AddDate(0, 0, 1).Format("2006-01-02"))
	if e != nil {
		return e
	}
	jsonResponse(w, 200, rows)
	return nil
}
