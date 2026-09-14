package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) shows(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	rows, e := s.DB.Rows(r.Context(), `SELECT s.*,f.added_at,f.favorite,(SELECT COUNT(*) FROM episodes e WHERE e.show_id=s.id) AS episode_count,(SELECT COUNT(*) FROM episodes e JOIN profile_episode_state p ON p.episode_id=e.id AND p.profile_id=f.profile_id WHERE e.show_id=s.id AND p.watched=1) AS watched_count,(SELECT COUNT(*) FROM episodes e WHERE e.show_id=s.id AND ((e.airstamp<>'' AND julianday(e.airstamp)<=julianday('now')) OR (e.airstamp='' AND e.airdate<>'' AND e.airdate<date('now')))) AS aired_count,(SELECT COUNT(*) FROM episodes e LEFT JOIN profile_episode_state p ON p.episode_id=e.id AND p.profile_id=f.profile_id WHERE e.show_id=s.id AND COALESCE(p.watched,0)=0 AND ((e.airstamp<>'' AND julianday(e.airstamp)<=julianday('now')) OR (e.airstamp='' AND e.airdate<>'' AND e.airdate<date('now')))) AS aired_unwatched,(SELECT MIN(e.airdate) FROM episodes e WHERE e.show_id=s.id AND e.airdate>=date('now')) AS next_episode FROM shows s JOIN profile_shows f ON f.show_id=s.id WHERE f.profile_id=? ORDER BY s.name COLLATE NOCASE`, session.Profile)
	if e != nil {
		return e
	}
	jsonResponse(w, 200, rows)
	return nil
}
