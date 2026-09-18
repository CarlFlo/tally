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
	Author    string `xml:"author"`
	Creator   string `xml:"creator"`
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
	uploader := strings.TrimSpace(item.Author)
	if uploader == "" {
		uploader = strings.TrimSpace(item.Creator)
	}
	result := SearchResult{ID: item.GUID, Name: item.Title, Size: item.Size, Provider: item.Indexer, Uploader: uploader, Source: "jackett", Published: item.PubDate}
	if result.Provider == "" {
		result.Provider = "Jackett"
	}
	if result.Size == 0 {
		result.Size = item.Enclosure.Length
	}
	for _, link := range []string{item.Enclosure.URL, item.Link} {
		link = strings.TrimSpace(link)
		if link == "" {
			continue
		}
		if ValidMagnet(link) {
			if result.Magnet == "" {
				result.Magnet = link
			}
			continue
		}
		if result.URL == "" {
			result.URL = link
		}
	}
	if result.URL != "" {
		result.DownloadType = "Torrent file"
	} else if result.Magnet != "" {
		result.DownloadType = "Magnet"
	}
	peers, explicitLeechers := 0, false
	categorySeen := map[int]bool{}
	for _, attribute := range item.Attrs {
		name := strings.ToLower(strings.TrimSpace(attribute.Name))
		value := strings.TrimSpace(attribute.Value)
		switch name {
		case "seeders":
			result.Seeders, _ = strconv.Atoi(value)
		case "peers":
			peers, _ = strconv.Atoi(value)
		case "leechers":
			result.Leechers, _ = strconv.Atoi(value)
			explicitLeechers = true
		case "grabs":
			result.Grabs, _ = strconv.Atoi(value)
		case "magneturl":
			if ValidMagnet(value) {
				result.Magnet = value
				if result.URL == "" {
					result.DownloadType = "Magnet"
				}
			}
		case "indexer":
			if value != "" {
				result.Provider = value
			}
		case "uploader", "author":
			if value != "" {
				result.Uploader = value
			}
		case "infohash":
			result.InfoHash = value
		case "category":
			if category, err := strconv.Atoi(value); err == nil && !categorySeen[category] {
				categorySeen[category] = true
				result.Categories = append(result.Categories, category)
			}
		case "tvdbid":
			result.TVDBID = value
		case "tmdbid":
			result.TMDBID = value
		case "imdbid":
			result.IMDBID = value
		case "tvmazeid":
			result.TVMazeID = value
		case "downloadvolumefactor":
			if factor, err := strconv.ParseFloat(value, 64); err == nil {
				result.DownloadVolumeFactor = &factor
			}
		case "uploadvolumefactor":
			if factor, err := strconv.ParseFloat(value, 64); err == nil {
				result.UploadVolumeFactor = &factor
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
