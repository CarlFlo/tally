package api

import (
	"net/http"
	"time"

	"github.com/CarlFlo/mediaManager/internal/activity"
	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) bulkState(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	var in struct {
		Season     *int  `json:"season"`
		AiredOnly  bool  `json:"aired_only"`
		Watched    *bool `json:"watched"`
		Downloaded *bool `json:"downloaded"`
	}
	if e := decode(r, &in); e != nil {
		return e
	}
	if in.Watched == nil && in.Downloaded == nil {
		return bad("choose watched or downloaded state")
	}
	id := r.PathValue("id")
	if !s.follows(r, session.Profile, id) {
		return apiError{404, "show is not in your library"}
	}
	var watched, downloaded any
	if in.Watched != nil {
		watched = *in.Watched
	}
	if in.Downloaded != nil {
		downloaded = *in.Downloaded
	}
	query := `INSERT INTO profile_episode_state SELECT ?,id,COALESCE(?,0),COALESCE(?,0),? FROM episodes WHERE show_id=?`
	args := []any{session.Profile, watched, downloaded, time.Now().Unix(), id}
	if in.Season != nil {
		query += " AND season=?"
		args = append(args, *in.Season)
	}
	if in.AiredOnly {
		query += " AND ((airstamp<>'' AND julianday(airstamp)<=julianday('now')) OR (airstamp='' AND airdate<>'' AND airdate<=date('now')))"
	}
	query += ` ON CONFLICT(profile_id,episode_id) DO UPDATE SET watched=COALESCE(?,watched),downloaded=COALESCE(?,downloaded),updated_at=excluded.updated_at`
	args = append(args, watched, downloaded)
	tx, e := s.DB.BeginTx(r.Context(), nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	res, e := tx.ExecContext(r.Context(), query, args...)
	if e != nil {
		return e
	}
	n, _ := res.RowsAffected()
	var name string
	if e = tx.QueryRowContext(r.Context(), "SELECT name FROM shows WHERE id=?", id).Scan(&name); e != nil {
		return e
	}
	if e = activity.Record(r.Context(), tx, activity.Event{Action: "episode_progress_updated", Profile: session.Profile, ShowID: id, ShowName: name, Message: "Updated episode progress for " + name}); e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	jsonResponse(w, 200, map[string]any{"updated": n})
	return nil
}
