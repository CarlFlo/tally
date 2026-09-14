package metadata

import (
	"context"
	"fmt"
)

func (s *Service) sync(ctx context.Context, external string) (string, error) {
	show, e := s.Provider.GetShow(ctx, external)
	if e != nil {
		return "", e
	}
	episodes, e := s.Provider.GetEpisodes(ctx, external)
	if e != nil {
		return "", e
	}
	if len(episodes) > 30000 {
		return "", fmt.Errorf("show exceeds 30000 episode safety limit")
	}
	seasons := []Season{}
	if p, ok := s.Provider.(interface {
		GetSeasons(context.Context, string) ([]Season, error)
	}); ok {
		seasons, e = p.GetSeasons(ctx, external)
		if e != nil {
			return "", e
		}
	}
	return (Repository{DB: s.DB}).Save(ctx, external, show, episodes, seasons)
}
