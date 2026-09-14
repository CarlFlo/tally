package torrent

import (
	"encoding/xml"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type jackettFeed struct {
	XMLName xml.Name
	Code    string `xml:"code,attr"`
	Channel struct {
		Items []jackettItem `xml:"item"`
	} `xml:"channel"`
}

type jackettItem struct {
	Title     string `xml:"title"`
	GUID      string `xml:"guid"`
	Link      string `xml:"link"`
	Size      int64  `xml:"size"`
	PubDate   string `xml:"pubDate"`
	Indexer   string `xml:"jackettindexer"`
	Enclosure struct {
		URL    string `xml:"url,attr"`
		Length int64  `xml:"length,attr"`
	} `xml:"enclosure"`
	Attrs []struct {
		Name  string `xml:"name,attr"`
		Value string `xml:"value,attr"`
	} `xml:"attr"`
}

func parseJackettResults(body []byte) ([]SearchResult, error) {
	var feed jackettFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("Jackett returned invalid XML")
	}
	if feed.XMLName.Local == "error" {
		return nil, fmt.Errorf("Jackett rejected the search (code %s); check its configuration", feed.Code)
	}
	if feed.XMLName.Local != "rss" {
		return nil, fmt.Errorf("Jackett did not return a Torznab RSS feed")
	}
	results := make([]SearchResult, 0, len(feed.Channel.Items))
	for index, item := range feed.Channel.Items {
		if index == 100 {
			break
		}
		results = append(results, normalizeJackettItem(item))
	}
	return results, nil
}

func normalizeJackettItem(item jackettItem) SearchResult {
	result := SearchResult{ID: item.GUID, Name: item.Title, Size: item.Size, Provider: item.Indexer, Source: "jackett", Published: item.PubDate}
	if result.Provider == "" {
		result.Provider = "Jackett"
	}
	if result.Size == 0 {
		result.Size = item.Enclosure.Length
	}
	link := item.Enclosure.URL
	if link == "" {
		link = item.Link
	}
	if ValidMagnet(link) {
		result.Magnet, result.DownloadType = link, "Magnet"
	} else {
		result.URL, result.DownloadType = link, "Torrent file"
	}
	peers, explicitLeechers := 0, false
	for _, attribute := range item.Attrs {
		switch strings.ToLower(attribute.Name) {
		case "seeders":
			result.Seeders, _ = strconv.Atoi(attribute.Value)
		case "peers":
			peers, _ = strconv.Atoi(attribute.Value)
		case "leechers":
			result.Leechers, _ = strconv.Atoi(attribute.Value)
			explicitLeechers = true
		case "magneturl":
			if ValidMagnet(attribute.Value) {
				result.Magnet, result.DownloadType = attribute.Value, "Magnet"
			}
		case "indexer":
			if attribute.Value != "" {
				result.Provider = attribute.Value
			}
		}
	}
	if !explicitLeechers {
		result.Leechers = peers
		if peers >= result.Seeders {
			result.Leechers = peers - result.Seeders
		}
	}
	return result
}

func filterResults(results []SearchResult, query SearchQuery) []SearchResult {
	filtered := results[:0]
	for _, result := range results {
		if result.Seeders < query.MinSeeders || result.Size < query.MinSize || (query.MaxSize > 0 && result.Size > query.MaxSize) {
			continue
		}
		name, include := strings.ToLower(result.Name), true
		for _, word := range strings.Fields(strings.ToLower(query.Include)) {
			include = include && strings.Contains(name, word)
		}
		for _, word := range strings.Fields(strings.ToLower(query.Exclude)) {
			include = include && !strings.Contains(name, word)
		}
		if include {
			filtered = append(filtered, result)
		}
	}
	sort.SliceStable(filtered, func(i, k int) bool { return filtered[i].Seeders > filtered[k].Seeders })
	return filtered
}
