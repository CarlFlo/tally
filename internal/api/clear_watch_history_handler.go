package api

import (
	"net/http"

	"github.com/CarlFlo/tally/internal/activity"
	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) clearWatchHistory(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	ctx, id := r.Context(), r.PathValue("id")
	if err := s.requireFollow(r.Context(), session.Profile, id); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE profile_episode_state SET watched=0,updated_at=unixepoch() WHERE profile_id=? AND watched=1 AND episode_id IN (SELECT id FROM episodes WHERE show_id=?)`, session.Profile, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	var name string
	if err = tx.QueryRowContext(ctx, "SELECT name FROM shows WHERE id=?", id).Scan(&name); err != nil {
		return err
	}
	if count > 0 {
		err = activity.Record(ctx, tx, activity.Event{Action: "watch_history_cleared", Profile: session.Profile, ShowID: id, ShowName: name, Message: "Cleared watch history for " + name})
		if err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]any{"updated": count})
	return nil
}
