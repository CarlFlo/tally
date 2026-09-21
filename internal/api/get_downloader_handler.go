package api

import (
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/torrent"
)

func (s *Server) getDownloader(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if e := s.operator(session); e != nil {
		return e
	}
	c, e := s.Clients.Load(r.Context())
	if e != nil {
		return e
	}
	w.Header().Set("Cache-Control", "no-store")
	view := c.View()
	jsonResponse(w, 200, map[string]any{"adapters": torrent.ClientDefinitions(), "settings": view})
	return nil
}
