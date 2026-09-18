package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/CarlFlo/tally/internal/activity"
	"github.com/CarlFlo/tally/internal/auth"
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

	where := []string{"show_id=?"}
	args := []any{id}
	if in.Season != nil {
		where = append(where, "season=?")
		args = append(args, *in.Season)
	}
	// Bulk watched/downloaded state is always limited to released episodes.
	// A date-only episode airing today is still considered upcoming until an
	// exact airstamp is known, matching the frontend release-time rules.
	where = append(where, "((airstamp<>'' AND julianday(airstamp)<=julianday('now')) OR (airstamp='' AND airdate<>'' AND airdate<date('now')))")
	filter := strings.Join(where, " AND ")

	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var updated int
	if err = tx.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM episodes WHERE "+filter, args...).Scan(&updated); err != nil {
		return err
	}
	if in.Watched != nil {
		watchArgs := []any{session.Profile, *in.Watched, time.Now().Unix()}
		watchArgs = append(watchArgs, args...)
		query := `INSERT INTO profile_episode_state(profile_id,episode_id,watched,downloaded,updated_at)
			SELECT ?,id,?,0,? FROM episodes WHERE ` + filter + `
			ON CONFLICT(profile_id,episode_id) DO UPDATE SET watched=excluded.watched,updated_at=excluded.updated_at`
		if _, err = tx.ExecContext(r.Context(), query, watchArgs...); err != nil {
			return err
		}
	}
	if in.Downloaded != nil {
		downloadArgs := []any{*in.Downloaded}
		downloadArgs = append(downloadArgs, args...)
		if _, err = tx.ExecContext(r.Context(), "UPDATE episodes SET downloaded=? WHERE "+filter, downloadArgs...); err != nil {
			return err
		}
	}

	var name string
	if err = tx.QueryRowContext(r.Context(), "SELECT name FROM shows WHERE id=?", id).Scan(&name); err != nil {
		return err
	}
	if err = activity.Record(r.Context(), tx, activity.Event{
		Action: "episode_progress_updated", Profile: session.Profile, ShowID: id, ShowName: name,
		Message: "Updated episode progress for " + name,
	}); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if in.Downloaded != nil && s.Events != nil {
		s.Events.Publish("", "shows", "show", "calendar")
	}
	jsonResponse(w, 200, map[string]any{"updated": updated})
	return nil
}
