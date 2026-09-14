package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) capabilities(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	client, err := s.Clients.Load(r.Context())
	if err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]any{"operator": session.Profile == "user0", "downloader": client.Adapter, "downloader_configured": client.Configured(), "jackett_configured": s.jackettConfigured()})
	return nil
}
