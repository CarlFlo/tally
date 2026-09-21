package api

import (
	"context"

	"github.com/CarlFlo/tally/internal/settings"
	"github.com/CarlFlo/tally/internal/torrent"
)

func (s *Server) jackett(ctx context.Context) (*torrent.Jackett, error) {
	var saved settings.Search
	if _, err := s.settingsStore().Load(ctx, "search", &saved); err != nil {
		return nil, err
	}
	configured := saved.Effective()
	if !configured.Configured() {
		return nil, nil
	}
	return &torrent.Jackett{Control: s.Control, BaseURL: configured.BaseURL, APIKey: configured.APIKey}, nil
}

func (s *Server) searchProvider(ctx context.Context, id string) (*torrent.Jackett, error) {
	if id != "" && id != "jackett" {
		return nil, nil
	}
	return s.jackett(ctx)
}

func (s *Server) jackettConfigured(ctx context.Context) (bool, error) {
	provider, err := s.jackett(ctx)
	return provider != nil, err
}
