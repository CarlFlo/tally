package api

import (
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/torrent"
)

func (s *Server) saveDownloader(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if e := s.operator(session); e != nil {
		return e
	}
	var in torrent.ClientUpdate
	if e := decode(r, &in); e != nil {
		return e
	}
	c, e := s.Clients.Save(r.Context(), in)
	if e != nil {
		return clientInputError(e)
	}
	jsonResponse(w, 200, c.View())
	return nil
}
