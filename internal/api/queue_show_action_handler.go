package api

import (
	"errors"
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/metadata"
)

func (s *Server) queueShowAction(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	var in struct {
		ID     int    `json:"tvmaze_id"`
		Name   string `json:"name"`
		Follow *bool  `json:"follow"`
	}
	if e := decode(r, &in); e != nil {
		return e
	}
	if in.Follow == nil {
		return bad("choose whether to add or remove the show")
	}
	if e := s.Metadata.QueueFollow(r.Context(), session.Profile, in.ID, in.Name, *in.Follow); e != nil {
		var queueErr metadata.QueueError
		if errors.As(e, &queueErr) {
			return bad(queueErr.Error())
		}
		return e
	}
	jsonResponse(w, 202, map[string]bool{"queued": true})
	return nil
}
