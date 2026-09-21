package api

import (
	"database/sql"
	"errors"
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
	var isReleased int
	err := s.DB.QueryRowContext(r.Context(), `SELECT show_id,
		CASE WHEN
			(airstamp<>'' AND julianday(airstamp)<=julianday('now'))
			OR (airstamp='' AND airdate<>'' AND airdate<date('now'))
		THEN 1 ELSE 0 END
		FROM episodes WHERE id=?`, episodeID).Scan(&show, &isReleased)
	if errors.Is(err, sql.ErrNoRows) {
		return apiError{404, "episode is not in your library"}
	}
	if err != nil {
		return err
	}
	if err = s.requireFollow(r.Context(), session.Profile, show); err != nil {
		if apiErr, ok := err.(apiError); ok && apiErr.Status == http.StatusNotFound {
			return apiError{404, "episode is not in your library"}
		}
		return err
	}
	if isReleased == 0 && ((in.Watched != nil && *in.Watched) || (in.Downloaded != nil && *in.Downloaded)) {
		return bad("episode has not been released yet")
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
	if in.Downloaded != nil && s.Events != nil {
		s.Events.Publish("", "shows", "show", "calendar")
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
