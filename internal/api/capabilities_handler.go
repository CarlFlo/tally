package api

import (
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) capabilities(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	client, err := s.Clients.Load(r.Context())
	if err != nil {
		return err
	}
	jackettConfigured, err := s.jackettConfigured(r.Context())
	if err != nil {
		return err
	}
	searchEnabled, err := s.torrentSearchEnabled(r.Context())
	if err != nil {
		return err
	}
	downloadsEnabled, err := s.torrentDownloadsEnabled(r.Context())
	if err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]any{"operator": session.Admin, "downloader": client.Adapter, "downloader_configured": client.Configured(), "jackett_configured": jackettConfigured, "jackett_enabled": searchEnabled, "torrent_search_enabled": searchEnabled, "torrent_downloads_enabled": downloadsEnabled})
	return nil
}
