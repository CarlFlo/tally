package api

import (
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) favoriteShow(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	var in struct {
		Favorite *bool `json:"favorite"`
	}
	if e := decode(r, &in); e != nil {
		return e
	}
	if in.Favorite == nil {
		return bad("choose a favorite state")
	}
	res, e := s.DB.ExecContext(r.Context(), "UPDATE profile_shows SET favorite=? WHERE profile_id=? AND show_id=?", *in.Favorite, session.Profile, r.PathValue("id"))
	if e != nil {
		return e
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return apiError{404, "show is not in your library"}
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
