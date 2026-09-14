package metadata

import (
	"context"
)

type TVProvider interface {
	SearchShows(context.Context, string) ([]SearchResult, error)
	GetShow(context.Context, string) (*Show, error)
	GetEpisodes(context.Context, string) ([]Episode, error)
}
