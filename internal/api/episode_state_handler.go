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
	episodeID := r.PathValue("id")
	var show string
	if s.DB.QueryRowContext(r.Context(), "SELECT show_id FROM episodes WHERE id=?", episodeID).Scan(&show) != nil || !s.follows(r, session.Profile, show) {
		return apiError{404, "episode is not in your library"}
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().Unix()
	if in.Watched != nil {
		if _, err = tx.ExecContext(r.Context(), `INSERT INTO profile_episode_state(profile_id,episode_id,watched,downloaded,updated_at)
			VALUES(?,?,?,0,?)
			ON CONFLICT(profile_id,episode_id) DO UPDATE SET watched=excluded.watched,updated_at=excluded.updated_at`,
			session.Profile, episodeID, *in.Watched, now); err != nil {
			return err
		}
	}
	if in.Downloaded != nil {
		if _, err = tx.ExecContext(r.Context(), "UPDATE episodes SET downloaded=? WHERE id=?", *in.Downloaded, episodeID); err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
