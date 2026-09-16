package api

import (
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) torrentHistory(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	searches, e := s.DB.Rows(r.Context(), "SELECT query,provider,results,created_at FROM torrent_search_history WHERE profile_id=? ORDER BY created_at DESC LIMIT 20", session.Profile)
	if e != nil {
		return e
	}
	sends, e := s.DB.Rows(r.Context(), "SELECT name,status,error,created_at FROM torrent_send_history WHERE profile_id=? ORDER BY created_at DESC LIMIT 20", session.Profile)
	if e != nil {
		return e
	}
	jsonResponse(w, 200, map[string]any{"searches": searches, "sends": sends})
	return nil
}

func (s *Server) clearTorrentHistory(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	var table string
	switch r.PathValue("kind") {
	case "searches":
		table = "torrent_search_history"
	case "submissions":
		table = "torrent_send_history"
	default:
		return apiError{404, "unknown torrent history"}
	}
	if _, e := s.DB.ExecContext(r.Context(), "DELETE FROM "+table+" WHERE profile_id=?", session.Profile); e != nil {
		return e
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
