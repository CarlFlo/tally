package api

import (
	"context"

	"github.com/CarlFlo/tally/internal/settings"
)

func (s *Server) torrentSearchEnabled(ctx context.Context) (bool, error) {
	var saved settings.Search
	if _, err := s.settingsStore().Load(ctx, "search", &saved); err != nil {
		return false, err
	}
	return saved.Effective().Enabled, nil
}

func (s *Server) torrentDownloadsEnabled(ctx context.Context) (bool, error) {
	var saved settings.Torrent
	if _, err := s.settingsStore().Load(ctx, "torrent", &saved); err != nil {
		return false, err
	}
	return saved.Enabled, nil
}
