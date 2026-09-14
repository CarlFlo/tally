package torrent

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/CarlFlo/mediaManager/internal/providers"
)

const jackettTorznabPath = "/api/v2.0/indexers/all/results/torznab/api"

type Jackett struct {
	Control providers.Requester
	BaseURL string
	APIKey  string
}

func (j *Jackett) ID() string   { return "jackett" }
func (j *Jackett) Name() string { return "Jackett" }

func (j *Jackett) Search(ctx context.Context, query SearchQuery) ([]SearchResult, error) {
	values := url.Values{"t": {"search"}, "q": {strings.Join(strings.Fields(query.Query), " ")}, "limit": {"100"}}
	endpoint, err := j.endpoint(values)
	if err != nil {
		return nil, err
	}
	response, err := j.Control.Do(ctx, providers.Request{Provider: j.ID(), URL: endpoint, Trigger: "torrent_search", Entity: "search", TTL: 2 * time.Minute})
	if err != nil {
		return nil, err
	}
	results, err := parseJackettResults(response.Body)
	if err != nil {
		return nil, err
	}
	for index := range results {
		if results[index].URL != "" && !j.validTorrentURL(results[index].URL) {
			results[index].URL = ""
			if results[index].Magnet == "" {
				results[index].DownloadType = "Unavailable"
			}
		}
	}
	return filterResults(results, query), nil
}

func (j *Jackett) TestConnection(ctx context.Context) error {
	endpoint, err := j.endpoint(url.Values{"t": {"caps"}})
	if err != nil {
		return err
	}
	response, err := j.Control.Do(ctx, providers.Request{Provider: j.ID(), URL: endpoint, Trigger: "connection_test", Force: true, NoRetry: true})
	if err != nil {
		return err
	}
	var root struct{ XMLName xml.Name }
	if err = xml.Unmarshal(response.Body, &root); err != nil || root.XMLName.Local != "caps" {
		return fmt.Errorf("Jackett did not return valid Torznab capabilities")
	}
	return nil
}

func (j *Jackett) endpoint(values url.Values) (string, error) {
	u, err := url.Parse(j.BaseURL)
	if err != nil || u.Host == "" || j.APIKey == "" {
		return "", fmt.Errorf("configure the Jackett base URL and API key")
	}
	u.Path = strings.TrimRight(u.Path, "/") + jackettTorznabPath
	values.Set("apikey", j.APIKey)
	u.RawQuery = values.Encode()
	return u.String(), nil
}
