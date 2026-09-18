package torrent

import "testing"

func TestJackettPreservesTorrentAndMagnetAlternatives(t *testing.T) {
	body := []byte(`<rss xmlns:torznab="http://torznab.com/schemas/2015/feed"><channel><item>
		<title>Example.Show.S01E02.1080p.WEB-DL-GROUP</title>
		<guid>dual</guid>
		<link>http://jackett.test/download/dual</link>
		<enclosure url="magnet:?xt=urn:btih:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" length="2147483648"/>
		<torznab:attr name="seeders" value="50"/>
	</item></channel></rss>`)
	results, err := parseJackettResults(body)
	if err != nil || len(results) != 1 {
		t.Fatalf("parse failed: %+v %v", results, err)
	}
	result := results[0]
	if result.URL != "http://jackett.test/download/dual" {
		t.Fatalf("torrent URL was lost: %+v", result)
	}
	if result.Magnet == "" || !ValidMagnet(result.Magnet) {
		t.Fatalf("magnet fallback was lost: %+v", result)
	}
	if result.DownloadType != "Torrent file" {
		t.Fatalf("inspectable torrent should be preferred: %+v", result)
	}
}
