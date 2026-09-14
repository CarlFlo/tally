package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) showActions(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	rows, e := s.DB.Rows(r.Context(), `SELECT a.*,EXISTS(SELECT 1 FROM external_ids x JOIN profile_shows f ON f.show_id=x.internal_id WHERE x.provider='tvmaze' AND x.kind='show' AND x.external_id=CAST(a.external_id AS TEXT) AND f.profile_id=a.profile_id) AS followed FROM show_actions a WHERE profile_id=? ORDER BY updated_at DESC LIMIT 1000`, session.Profile)
	if e != nil {
		return e
	}
	jsonResponse(w, 200, rows)
	return nil
}
