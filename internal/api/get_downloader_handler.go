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
	view := c.View()
	if r.URL.Query().Get("reveal") == "1" {
		w.Header().Set("Cache-Control", "no-store")
		for key, configured := range view.SecretsConfigured {
			if configured {
				view.Fields[key] = c.Fields[key]
			}
		}
	}
	jsonResponse(w, 200, map[string]any{"adapters": torrent.ClientDefinitions(), "settings": view})
	return nil
}
