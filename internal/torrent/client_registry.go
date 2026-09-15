package torrent

import (
	"github.com/CarlFlo/tally/internal/providers"
)

type ClientField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Secret      bool   `json:"secret"`
	Placeholder string `json:"placeholder,omitempty"`
	Help        string `json:"help,omitempty"`
}

type ClientDefinition struct {
	ID     string        `json:"id"`
	Name   string        `json:"name"`
	Fields []ClientField `json:"fields"`
}

type clientAdapter struct {
	ClientDefinition
	create   func(providers.Requester, map[string]string) DownloadClient
	validate func(map[string]string) error
}

// Register compiled adapters here; the UI obtains its dropdown and fields from this list.
// Each adapter's implementation and field definition live in its own file.
func clientAdapters() []clientAdapter { return []clientAdapter{qBittorrentAdapter()} }

func ClientDefinitions() []ClientDefinition {
	adapters := clientAdapters()
	out := make([]ClientDefinition, 0, len(adapters))
	for _, adapter := range adapters {
		out = append(out, adapter.ClientDefinition)
	}
	return out
}

func findClient(id string) *clientAdapter {
	for _, adapter := range clientAdapters() {
		if adapter.ID == id {
			return &adapter
		}
	}
	return nil
}
