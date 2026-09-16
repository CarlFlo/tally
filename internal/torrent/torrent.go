package torrent

import (
	"context"
)

const TallyCategory = "tally"

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

type Download struct {
	Hash          string  `json:"hash"`
	Name          string  `json:"name"`
	State         string  `json:"state"`
	Progress      float64 `json:"progress"`
	Size          int64   `json:"size"`
	Downloaded    int64   `json:"downloaded"`
	DownloadSpeed int64   `json:"download_speed"`
	UploadSpeed   int64   `json:"upload_speed"`
	Ratio         float64 `json:"ratio"`
	AddedOn       int64   `json:"added_on"`
	Category      string  `json:"category"`
}

type DownloadStats struct {
	Total         int   `json:"total"`
	Active        int   `json:"active"`
	DownloadSpeed int64 `json:"download_speed"`
	UploadSpeed   int64 `json:"upload_speed"`
}

type DownloadSnapshot struct {
	Torrents []Download    `json:"torrents"`
	Stats    DownloadStats `json:"stats"`
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
	Downloads(context.Context, string) (DownloadSnapshot, error)
	Stop(context.Context, string) error
	Start(context.Context, string) error
	Remove(context.Context, string, bool) error
}
