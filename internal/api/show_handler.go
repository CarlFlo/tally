package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) show(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	id := r.PathValue("id")
	if !s.follows(r, session.Profile, id) {
		return apiError{404, "show is not in your library"}
	}
	shows, e := s.DB.Rows(r.Context(), "SELECT s.*,f.favorite FROM shows s JOIN profile_shows f ON f.show_id=s.id WHERE s.id=? AND f.profile_id=?", id, session.Profile)
	if e != nil {
		return e
	}
	episodes, e := s.DB.Rows(r.Context(), episodeSelect+"WHERE e.show_id=? ORDER BY e.season,e.number,e.airdate", session.Profile, id)
	if e != nil {
		return e
	}
	seasons, e := s.DB.Rows(r.Context(), "SELECT * FROM seasons WHERE show_id=? ORDER BY number", id)
	if e != nil {
		return e
	}
	ids, e := s.DB.Rows(r.Context(), "SELECT provider,external_id FROM external_ids WHERE kind='show' AND internal_id=?", id)
	if e != nil {
		return e
	}
	jsonResponse(w, 200, map[string]any{"show": shows[0], "episodes": episodes, "seasons": seasons, "external_ids": ids})
	return nil
}
