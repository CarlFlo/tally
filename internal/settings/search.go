package settings

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/CarlFlo/tally/internal/config"
)

// Indexer is retained only to read older multi-provider settings. New saves use
// the single Jackett fields on Search.
type Indexer struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	URL     string `json:"url"`
	APIKey  string `json:"api_key"`
	Enabled bool   `json:"enabled"`
}

type Search struct {
	BaseURL   string    `json:"base_url"`
	APIKey    string    `json:"api_key"`
	Enabled   bool      `json:"enabled"`
	Providers []Indexer `json:"providers,omitempty"`
}

func (s Search) Effective() Search {
	if s.BaseURL != "" || s.APIKey != "" {
		s.Providers = nil
		return s
	}
	for _, provider := range s.Providers {
		if base, ok := legacyJackettBase(provider.URL); ok && provider.Enabled {
			return Search{BaseURL: base, APIKey: provider.APIKey, Enabled: true}
		}
	}
	return Search{}
}

func (s Search) Configured() bool {
	effective := s.Effective()
	if strings.TrimSpace(effective.BaseURL) == "" || effective.APIKey == "" {
		return false
	}
	effective.Enabled = true
	return ValidateSearch(effective) == nil
}

func ValidateSearch(s Search) error {
	if len(s.Providers) > 0 {
		return fmt.Errorf("save one Jackett connection instead of search providers")
	}
	if !s.Enabled && s.APIKey == "" {
		if s.BaseURL == "" {
			return nil
		}
		u, err := url.Parse(s.BaseURL)
		if err != nil || config.ValidateURL(s.BaseURL) != nil || u.RawQuery != "" || u.ForceQuery {
			return fmt.Errorf("enter a Jackett HTTP(S) base URL without a query, credentials, or fragment")
		}
		return nil
	}
	if strings.TrimSpace(s.BaseURL) == "" {
		return fmt.Errorf("Jackett base URL is required")
	}
	u, err := url.Parse(s.BaseURL)
	if err != nil || config.ValidateURL(s.BaseURL) != nil || u.RawQuery != "" || u.ForceQuery {
		return fmt.Errorf("enter a Jackett HTTP(S) base URL without a query, credentials, or fragment")
	}
	if s.APIKey == "" {
		return fmt.Errorf("Jackett API key is required")
	}
	if len(s.APIKey) > 4096 || strings.ContainsAny(s.APIKey, "\r\n") {
		return fmt.Errorf("invalid Jackett API key")
	}
	return nil
}

func legacyJackettBase(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	const marker = "/api/v2.0/indexers/"
	index := strings.Index(strings.ToLower(u.Path), marker)
	if index < 0 {
		return "", false
	}
	u.Path = strings.TrimRight(u.Path[:index], "/")
	u.RawQuery = ""
	return strings.TrimRight(u.String(), "/"), true
}
