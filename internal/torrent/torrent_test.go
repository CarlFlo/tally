package torrent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/providers"
)

func TestJackettSearchConnectionAndFilters(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, e := database.Open(ctx, dir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	control, e := providers.New(ctx, db, dir, 2, 0)
	if e != nil {
		t.Fatal(e)
	}
	defer control.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != jackettTorznabPath {
			t.Errorf("unexpected Jackett path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("apikey") != "secret" {
			t.Error("key missing")
		}
		if r.URL.Query().Get("t") == "caps" {
			w.Write([]byte(`<caps><searching><search available="yes"/></searching></caps>`))
			return
		}
		w.Write([]byte(`<rss xmlns:torznab="http://torznab.com/schemas/2015/feed"><channel><item><title>Linux 1080p HEVC</title><guid>one</guid><jackettindexer>Example Indexer</jackettindexer><pubDate>Sun, 13 Sep 2026 10:00:00 +0000</pubDate><enclosure url="magnet:?xt=urn:btih:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" length="2048"/><torznab:attr name="seeders" value="50"/><torznab:attr name="peers" value="62"/><torznab:attr name="grabs" value="87"/><torznab:attr name="uploader" value="trusted-user"/><torznab:attr name="infohash" value="aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"/><torznab:attr name="category" value="5000"/><torznab:attr name="category" value="5030"/><torznab:attr name="tvdbid" value="123"/><torznab:attr name="tmdbid" value="456"/><torznab:attr name="imdbid" value="tt7654321"/><torznab:attr name="tvmazeid" value="789"/><torznab:attr name="downloadvolumefactor" value="0"/><torznab:attr name="uploadvolumefactor" value="2"/></item><item><title>Other release</title><guid>two</guid><size>1024</size><torznab:attr name="seeders" value="5"/></item></channel></rss>`))
	}))
	defer server.Close()
	p := Jackett{Control: control, BaseURL: server.URL, APIKey: "secret"}
	if e = p.TestConnection(ctx); e != nil {
		t.Fatal(e)
	}
	results, e := p.Search(ctx, SearchQuery{Query: "Linux", Include: "HEVC", MinSeeders: 10})
	if e != nil || len(results) != 1 {
		t.Fatalf("parse/filter failure: %+v %v", results, e)
	}
	result := results[0]
	if result.Seeders != 50 || result.Leechers != 12 || result.Grabs != 87 || result.Provider != "Example Indexer" || result.Uploader != "trusted-user" || result.DownloadType != "Magnet" {
		t.Fatalf("basic metadata parse failure: %+v", result)
	}
	if result.InfoHash != strings.Repeat("a", 40) || result.TVDBID != "123" || result.TMDBID != "456" || result.IMDBID != "tt7654321" || result.TVMazeID != "789" {
		t.Fatalf("identity metadata parse failure: %+v", result)
	}
	if len(result.Categories) != 2 || result.Categories[0] != 5000 || result.Categories[1] != 5030 {
		t.Fatalf("category metadata parse failure: %+v", result.Categories)
	}
	if result.DownloadVolumeFactor == nil || *result.DownloadVolumeFactor != 0 || result.UploadVolumeFactor == nil || *result.UploadVolumeFactor != 2 {
		t.Fatalf("ratio metadata parse failure: download=%v upload=%v", result.DownloadVolumeFactor, result.UploadVolumeFactor)
	}
	if p.validTorrentURL("https://evil.example/secret.torrent") {
		t.Fatal("foreign torrent URL accepted")
	}
}

func TestJackettUploaderFallsBackToAuthor(t *testing.T) {
	results, err := parseJackettResults([]byte(`<rss><channel><item><title>Example.Show.S01E02</title><guid>one</guid><author>author-user</author><enclosure url="magnet:?xt=urn:btih:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" length="2048"/></item></channel></rss>`))
	if err != nil || len(results) != 1 {
		t.Fatalf("parse failure: %+v %v", results, err)
	}
	if results[0].Uploader != "author-user" {
		t.Fatalf("author was not normalized as uploader: %+v", results[0])
	}
}

func TestMagnetValidation(t *testing.T) {
	for _, v := range []string{"javascript:alert(1)", "http://example.com", "magnet:?xt=anything", "magnet:?xt=urn:btih:" + strings.Repeat("z", 40)} {
		if ValidMagnet(v) {
			t.Fatalf("invalid magnet accepted: %s", v)
		}
	}
	if !ValidMagnet("magnet:?xt=urn:btih:" + strings.Repeat("a", 40)) {
		t.Fatal("valid magnet rejected")
	}
}
