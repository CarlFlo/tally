package api

import (
	"net/http"
	"time"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) episodeState(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	var in struct {
		Watched    *bool `json:"watched"`
		Downloaded *bool `json:"downloaded"`
	}
	if e := decode(r, &in); e != nil {
		return e
	}
	if in.Watched == nil && in.Downloaded == nil {
		return bad("choose watched or downloaded state")
	}
	var show string
	if s.DB.QueryRowContext(r.Context(), "SELECT show_id FROM episodes WHERE id=?", r.PathValue("id")).Scan(&show) != nil || !s.follows(r, session.Profile, show) {
		return apiError{404, "episode is not in your library"}
	}
	var watched, downloaded any
	if in.Watched != nil {
		watched = *in.Watched
	}
	if in.Downloaded != nil {
		downloaded = *in.Downloaded
	}
	_, e := s.DB.ExecContext(r.Context(), `INSERT INTO profile_episode_state VALUES(?,?,COALESCE(?,0),COALESCE(?,0),?) ON CONFLICT(profile_id,episode_id) DO UPDATE SET watched=COALESCE(?,watched),downloaded=COALESCE(?,downloaded),updated_at=excluded.updated_at`, session.Profile, r.PathValue("id"), watched, downloaded, time.Now().Unix(), watched, downloaded)
	if e != nil {
		return e
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
