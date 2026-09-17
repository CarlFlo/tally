package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/CarlFlo/tally/internal/settings"
	"github.com/CarlFlo/tally/internal/torrent"
)

func seedTorrentEpisodeTarget(t *testing.T, s *Server) {
	t.Helper()
	if _, err := s.DB.Exec(`INSERT INTO shows(id,name,premiered) VALUES('show-confidence','Example Show','2026-01-01');
INSERT INTO episodes(id,show_id,season,number,name) VALUES('episode-confidence','show-confidence',1,2,'Second');
INSERT INTO profile_shows(profile_id,show_id,added_at) VALUES('profile-admin','show-confidence',1);
INSERT INTO external_ids(provider,kind,external_id,internal_id) VALUES('tvmaze','show','4242','show-confidence');`); err != nil {
		t.Fatal(err)
	}
}

func torrentFileFixture(name string) string {
	return fmt.Sprintf("d4:infod6:lengthi2048e4:name%d:%s12:piece lengthi16384e6:pieces20:aaaaaaaaaaaaaaaaaaaaee", len(name), name)
}

func TestTorrentSearchAddsConfidenceOnlyForAuthoritativeEpisodeContext(t *testing.T) {
	s, handler, _ := testServer(t, "disabled")
	seedTorrentEpisodeTarget(t, s)
	jackett := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<rss xmlns:torznab="http://torznab.com/schemas/2015/feed"><channel><item><title>Example.Show.S01E02.1080p.WEB-DL.H264-GROUP</title><guid>one</guid><enclosure url="magnet:?xt=urn:btih:%s" length="2048"/><torznab:attr name="seeders" value="50"/><torznab:attr name="tvmazeid" value="4242"/></item></channel></rss>`, strings.Repeat("a", 40))
	}))
	defer jackett.Close()
	if err := s.settingsStore().Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.settingsStore().Save(context.Background(), "search", settings.Search{BaseURL: jackett.URL, APIKey: "PRIVATE-KEY", Enabled: true}, 1); err != nil {
		t.Fatal(err)
	}

	exact := request(t, handler, "POST", "/api/torrents/search", map[string]any{"query": "Example Show S01E02"})
	expect(t, exact, 200)
	var exactOut struct {
		Results []map[string]any `json:"results"`
	}
	if err := json.Unmarshal(exact.Body.Bytes(), &exactOut); err != nil || len(exactOut.Results) != 1 {
		t.Fatalf("unexpected exact search response: %s", exact.Body.String())
	}
	if exactOut.Results[0]["confidence"] != "high" || exactOut.Results[0]["verification"] != "unverified" {
		t.Fatalf("exact followed episode did not receive preliminary confidence: %s", exact.Body.String())
	}

	freeText := request(t, handler, "POST", "/api/torrents/search", map[string]any{"query": "Example"})
	expect(t, freeText, 200)
	var freeOut struct {
		Results []map[string]any `json:"results"`
	}
	if err := json.Unmarshal(freeText.Body.Bytes(), &freeOut); err != nil || len(freeOut.Results) != 1 {
		t.Fatalf("unexpected free-text search response: %s", freeText.Body.String())
	}
	if _, ok := freeOut.Results[0]["confidence"]; ok {
		t.Fatalf("arbitrary free-text search received authoritative confidence: %s", freeText.Body.String())
	}
}

func TestTorrentPayloadEpisodeMismatchIsRejectedBeforeQBittorrent(t *testing.T) {
	s, handler, _ := testServer(t, "disabled")
	seedTorrentEpisodeTarget(t, s)
	jackett := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/download" {
			w.Write([]byte(torrentFileFixture("Example.Show.S01E03.mkv")))
			return
		}
		fmt.Fprintf(w, `<rss><channel><item><title>Example.Show.S01E02.1080p.WEB-DL.H264-GROUP</title><guid>one</guid><enclosure url="http://%s/download?apikey=PRIVATE-KEY" length="2048"/></item></channel></rss>`, r.Host)
	}))
	defer jackett.Close()
	var clientRequests atomic.Int32
	client := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientRequests.Add(1)
		w.WriteHeader(http.StatusOK)
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
	search := request(t, handler, "POST", "/api/torrents/search", map[string]any{"query": "Example Show S01E02", "episode_id": "episode-confidence"})
	expect(t, search, 200)
	var output struct {
		Results []struct {
			ID         string `json:"id"`
			Confidence string `json:"confidence"`
		} `json:"results"`
	}
	if err := json.Unmarshal(search.Body.Bytes(), &output); err != nil || len(output.Results) != 1 || output.Results[0].Confidence != "high" {
		t.Fatalf("unexpected search response: %s", search.Body.String())
	}
	response := request(t, handler, "POST", "/api/torrents/send", map[string]string{"selection": output.Results[0].ID, "idempotency_key": "0123456789abcdef"})
	expect(t, response, 400)
	if !strings.Contains(response.Body.String(), "different episode") {
		t.Fatalf("payload mismatch was not explained: %s", response.Body.String())
	}
	if clientRequests.Load() != 0 {
		t.Fatal("episode-mismatched torrent reached qBittorrent")
	}
}
