package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CarlFlo/tally/internal/settings"
	"github.com/CarlFlo/tally/internal/torrent"
)

func TestJackettSearchNormalizesResultsWithoutLeakingURL(t *testing.T) {
	s, handler, _ := testServer(t, "disabled")
	jackett := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("apikey") != "PRIVATE-KEY" || r.URL.Query().Get("q") != "Linux" {
			t.Error("Jackett query or API key missing")
		}
		fmt.Fprintf(w, `<rss xmlns:torznab="http://torznab.com/schemas/2015/feed"><channel><item><title>Linux release</title><guid>one</guid><jackettindexer>Fixture Indexer</jackettindexer><pubDate>Sun, 13 Sep 2026 10:00:00 +0000</pubDate><enclosure url="magnet:?xt=urn:btih:%s" length="1024"/><torznab:attr name="seeders" value="20"/><torznab:attr name="leechers" value="4"/></item></channel></rss>`, strings.Repeat("a", 40))
	}))
	defer jackett.Close()
	if err := s.settingsStore().Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, err := s.settingsStore().Save(context.Background(), "search", settings.Search{BaseURL: jackett.URL, APIKey: "PRIVATE-KEY", Enabled: true}, 1)
	if err != nil {
		t.Fatal(err)
	}
	response := request(t, handler, "POST", "/api/torrents/search", map[string]any{"query": "Linux"})
	expect(t, response, 200)
	var output struct{ Results []map[string]any }
	if err = json.Unmarshal(response.Body.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if len(output.Results) != 1 || output.Results[0]["provider"] != "Fixture Indexer" || output.Results[0]["download_type"] != "Magnet" || output.Results[0]["leechers"] != float64(4) {
		t.Fatalf("Jackett result was not normalized: %s", response.Body.String())
	}
	if strings.Contains(response.Body.String(), "PRIVATE-KEY") || strings.Contains(response.Body.String(), jackett.URL) {
		t.Fatal("private Jackett configuration leaked")
	}
}

func TestJackettConnectionTestUsesSavedSearchProtocol(t *testing.T) {
	_, handler, _ := testServer(t, "disabled")
	jackett := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2.0/indexers/all/results/torznab/api" || r.URL.Query().Get("t") != "caps" || r.URL.Query().Get("apikey") != "PRIVATE-KEY" {
			t.Errorf("unexpected connection test request: %s", r.URL.String())
		}
		w.Write([]byte(`<caps><searching><search available="yes"/></searching></caps>`))
	}))
	defer jackett.Close()
	body := map[string]any{"data": settings.Search{BaseURL: jackett.URL, APIKey: "PRIVATE-KEY", Enabled: true}}
	expect(t, request(t, handler, "POST", "/api/settings/search/test", body), 200)
	body = map[string]any{"data": settings.Search{BaseURL: "file:///unsafe", APIKey: "PRIVATE-KEY", Enabled: true}}
	expect(t, request(t, handler, "POST", "/api/settings/search/test", body), 400)
}

func TestJackettTorrentFileSelectionIsFetchedAndSent(t *testing.T) {
	s, handler, _ := testServer(t, "disabled")
	jackett := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/download" {
			w.Write([]byte("de"))
			return
		}
		fmt.Fprintf(w, `<rss><channel><item><title>Torrent file result</title><guid>two</guid><enclosure url="http://%s/download?apikey=PRIVATE-KEY" length="2048"/></item></channel></rss>`, r.Host)
	}))
	defer jackett.Close()
	client := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+fixtureClientKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/v2/torrents/categories":
			w.Write([]byte(`{"tally":{"name":"tally","savePath":""}}`))
		case "/api/v2/torrents/add":
			if e := r.ParseMultipartForm(1 << 20); e != nil {
				t.Error(e)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if r.FormValue("category") != torrent.TallyCategory {
				t.Error("torrent file was not assigned to the Tally category")
			}
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer client.Close()
	ctx := context.Background()
	if err := s.settingsStore().Ensure(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := s.settingsStore().Save(ctx, "search", settings.Search{BaseURL: jackett.URL, APIKey: "PRIVATE-KEY", Enabled: true}, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Clients.Save(ctx, torrent.ClientUpdate{Adapter: "qbittorrent", Fields: map[string]string{"url": client.URL, "api_key": fixtureClientKey}}); err != nil {
		t.Fatal(err)
	}
	search := request(t, handler, "POST", "/api/torrents/search", map[string]any{"query": "Linux"})
	expect(t, search, 200)
	var output struct {
		Results []struct {
			ID           string `json:"id"`
			DownloadType string `json:"download_type"`
		} `json:"results"`
	}
	if err := json.Unmarshal(search.Body.Bytes(), &output); err != nil || len(output.Results) != 1 || output.Results[0].DownloadType != "Torrent file" {
		t.Fatalf("unexpected search response: %s", search.Body.String())
	}
	sent := request(t, handler, "POST", "/api/torrents/send", map[string]string{"selection": output.Results[0].ID, "idempotency_key": "0123456789abcdef"})
	expect(t, sent, 200)
}
