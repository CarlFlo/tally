package torrent

import (
	"context"
)

type SearchQuery struct {
	Query            string
	MinSeeders       int
	MinSize, MaxSize int64
	Include, Exclude string
}

type SearchResult struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	Seeders      int    `json:"seeders"`
	Leechers     int    `json:"leechers"`
	Provider     string `json:"provider"`
	Source       string `json:"source"`
	Magnet       string `json:"magnet"`
	URL          string `json:"url"`
	Published    string `json:"published"`
	DownloadType string `json:"download_type"`
}

type SearchProvider interface {
	ID() string
	Name() string
	Search(context.Context, SearchQuery) ([]SearchResult, error)
}

type DownloadClient interface {
	Name() string
	TestConnection(context.Context) error
	AddMagnet(context.Context, string) error
	AddTorrent(context.Context, []byte) error
}
