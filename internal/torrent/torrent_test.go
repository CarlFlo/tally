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
		w.Write([]byte(`<rss xmlns:torznab="http://torznab.com/schemas/2015/feed"><channel><item><title>Linux 1080p HEVC</title><guid>one</guid><jackettindexer>Example Indexer</jackettindexer><pubDate>Sun, 13 Sep 2026 10:00:00 +0000</pubDate><enclosure url="magnet:?xt=urn:btih:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" length="2048"/><torznab:attr name="seeders" value="50"/><torznab:attr name="peers" value="62"/></item><item><title>Other release</title><guid>two</guid><size>1024</size><torznab:attr name="seeders" value="5"/></item></channel></rss>`))
	}))
	defer server.Close()
	p := Jackett{Control: control, BaseURL: server.URL, APIKey: "secret"}
	if e = p.TestConnection(ctx); e != nil {
		t.Fatal(e)
	}
	results, e := p.Search(ctx, SearchQuery{Query: "Linux", Include: "HEVC", MinSeeders: 10})
	if e != nil || len(results) != 1 || results[0].Seeders != 50 || results[0].Leechers != 12 || results[0].Provider != "Example Indexer" || results[0].DownloadType != "Magnet" {
		t.Fatalf("parse/filter failure: %+v %v", results, e)
	}
	if p.validTorrentURL("https://evil.example/secret.torrent") {
		t.Fatal("foreign torrent URL accepted")
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
