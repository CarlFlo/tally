package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
	"github.com/CarlFlo/mediaManager/internal/library"
)

func (s *Server) removeShow(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	ctx, id := r.Context(), r.PathValue("id")
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Supersede an older queued add so it cannot silently restore the follow.
	_, err = tx.ExecContext(ctx, `UPDATE show_actions SET desired=0,status='done',revision=revision+1,error='' WHERE profile_id=? AND (show_id=? OR external_id IN (SELECT external_id FROM external_ids WHERE provider='tvmaze' AND kind='show' AND internal_id=?))`, session.Profile, id, id)
	if err != nil {
		return err
	}
	if err = library.SetFollow(ctx, tx, session.Profile, id, false); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
