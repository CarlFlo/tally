package api

import (
	"net/http"
	"strconv"

	"github.com/CarlFlo/mediaManager/internal/auth"
	"github.com/CarlFlo/mediaManager/internal/library"
	"github.com/CarlFlo/mediaManager/internal/metadata"
)

func (s *Server) addShow(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	var in struct {
		TVMazeID int `json:"tvmaze_id"`
	}
	if e := decode(r, &in); e != nil {
		return e
	}
	if in.TVMazeID <= 0 {
		return bad("choose a valid show")
	}
	var id string
	e := s.DB.QueryRowContext(r.Context(), "SELECT internal_id FROM external_ids WHERE provider='tvmaze' AND kind='show' AND external_id=?", strconv.Itoa(in.TVMazeID)).Scan(&id)
	if e != nil {
		id, e = s.Metadata.Sync(metadata.WithInfo(r.Context(), metadata.Info{Trigger: "user_add"}), strconv.Itoa(in.TVMazeID))
		if e != nil {
			return remote(e)
		}
	} else {
		s.Control.Avoid("tvmaze", "user_add", id, "shared metadata")
	}
	tx, e := s.DB.BeginTx(r.Context(), nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = library.SetFollow(r.Context(), tx, session.Profile, id, true); e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	jsonResponse(w, 201, map[string]string{"id": id})
	return nil
}
