package api

import (
	"context"

	"github.com/CarlFlo/tally/internal/settings"
)

func (s *Server) torrentSearchEnabled(ctx context.Context) bool {
	var saved settings.Search
	if _, err := s.settingsStore().Load(ctx, "search", &saved); err != nil {
		return false
	}
	return saved.Effective().Enabled
}

func (s *Server) torrentDownloadsEnabled(ctx context.Context) bool {
	var saved settings.Torrent
	if _, err := s.settingsStore().Load(ctx, "torrent", &saved); err != nil {
		return true
	}
	return saved.Enabled
}
