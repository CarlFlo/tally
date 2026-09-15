package api

import (
	"context"

	"github.com/CarlFlo/tally/internal/settings"
	"github.com/CarlFlo/tally/internal/torrent"
)

func (s *Server) jackett(ctx context.Context) *torrent.Jackett {
	var saved settings.Search
	if _, err := s.settingsStore().Load(ctx, "search", &saved); err != nil {
		return nil
	}
	configured := saved.Effective()
	if !configured.Configured() {
		return nil
	}
	return &torrent.Jackett{Control: s.Control, BaseURL: configured.BaseURL, APIKey: configured.APIKey}
}

func (s *Server) searchProvider(id string) *torrent.Jackett {
	if id != "" && id != "jackett" {
		return nil
	}
	return s.jackett(context.Background())
}

func (s *Server) jackettConfigured() bool {
	return s.searchProvider("jackett") != nil
}
