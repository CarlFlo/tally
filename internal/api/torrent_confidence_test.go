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

	"github.com/CarlFlo/tally/internal/metadata"
	"github.com/CarlFlo/tally/internal/settings"
	"github.com/CarlFlo/tally/internal/torrent"
)

type torrentTargetTV struct {
	results      []metadata.SearchResult
	episodes     []metadata.Episode
	searchErr    error
	episodeErr   error
	searchCalls  atomic.Int32
	episodeCalls atomic.Int32
}

func (f *torrentTargetTV) SearchShows(context.Context, string) ([]metadata.SearchResult, error) {
	f.searchCalls.Add(1)
	return f.results, f.searchErr
}

func (f *torrentTargetTV) GetShow(context.Context, string) (*metadata.Show, error) {
	return nil, fmt.Errorf("unexpected GetShow call")
}

func (f *torrentTargetTV) GetEpisodes(context.Context, string) ([]metadata.Episode, error) {
	f.episodeCalls.Add(1)
	return f.episodes, f.episodeErr
}

func seedTorrentEpisodeTarget(t *testing.T, s *Server) {
	t.Helper()
	if _, err := s.DB.Exec(`INSERT INTO shows(id,name,premiered) VALUES('show-confidence','Example Show','2026-01-01');
INSERT INTO episodes(id,show_id,season,number,name,runtime) VALUES('episode-confidence','show-confidence',1,2,'Second',45);
INSERT INTO profile_shows(profile_id,show_id,added_at) VALUES('profile-admin','show-confidence',1);
INSERT INTO external_ids(provider,kind,external_id,internal_id) VALUES('tvmaze','show','4242','show-confidence');`); err != nil {
		t.Fatal(err)
	}
}

func torrentFileFixture(name string) string {
	return fmt.Sprintf("d4:infod6:lengthi2048e4:name%d:%s12:piece lengthi16384e6:pieces20:aaaaaaaaaaaaaaaaaaaaee", len(name), name)
}

func TestTorrentSearchEvaluatesConfidenceOnlyWhenResultIsExpanded(t *testing.T) {
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
	if _, ok := exactOut.Results[0]["confidence"]; ok {
		t.Fatalf("discovery eagerly evaluated confidence: %s", exact.Body.String())
	}
	exactID, _ := exactOut.Results[0]["id"].(string)
	evaluation := request(t, handler, "GET", "/api/torrents/search/"+exactID+"/evaluation", nil)
	expect(t, evaluation, 200)
	var evaluated map[string]any
	if err := json.Unmarshal(evaluation.Body.Bytes(), &evaluated); err != nil {
		t.Fatal(err)
	}
	if evaluated["confidence"] != "high" || evaluated["verification"] != "unverified" {
		t.Fatalf("expanded result did not receive preliminary confidence: %s", evaluation.Body.String())
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
		t.Fatalf("arbitrary free-text discovery received authoritative confidence: %s", freeText.Body.String())
	}
	freeID, _ := freeOut.Results[0]["id"].(string)
	freeEvaluation := request(t, handler, "GET", "/api/torrents/search/"+freeID+"/evaluation", nil)
	expect(t, freeEvaluation, 200)
	if strings.Contains(freeEvaluation.Body.String(), "\"confidence\"") {
		t.Fatalf("arbitrary free-text evaluation received authoritative confidence: %s", freeEvaluation.Body.String())
	}
}

func TestTorrentFreeTextEpisodeUsesSharedUnfollowedMetadata(t *testing.T) {
	s, handler, tv := testServer(t, "disabled")
	if _, err := s.DB.Exec(`INSERT INTO shows(id,name,premiered,runtime) VALUES('show-shared','Shared Show','2025-01-01',50);
INSERT INTO episodes(id,show_id,season,number,name,runtime) VALUES('episode-shared','show-shared',1,3,'Third',45);
INSERT INTO external_ids(provider,kind,external_id,internal_id) VALUES('tvmaze','show','5151','show-shared');`); err != nil {
		t.Fatal(err)
	}
	size := int64(45 * 50 * 1024 * 1024)
	jackett := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<rss xmlns:torznab="http://torznab.com/schemas/2015/feed"><channel><item><title>Shared.Show.S01E03.1080p.WEB-DL-GROUP</title><guid>shared</guid><enclosure url="magnet:?xt=urn:btih:%s" length="%d"/><torznab:attr name="seeders" value="40"/></item></channel></rss>`, strings.Repeat("c", 40), size)
	}))
	defer jackett.Close()
	if err := s.settingsStore().Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.settingsStore().Save(context.Background(), "search", settings.Search{BaseURL: jackett.URL, APIKey: "PRIVATE-KEY", Enabled: true}, 1); err != nil {
		t.Fatal(err)
	}

	response := request(t, handler, "POST", "/api/torrents/search", map[string]any{"query": "Shared Show S01E03"})
	expect(t, response, 200)
	var out struct {
		Results []map[string]any `json:"results"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil || len(out.Results) != 1 {
		t.Fatalf("unexpected response: %s", response.Body.String())
	}
	if _, ok := out.Results[0]["confidence"]; ok {
		t.Fatalf("shared metadata was evaluated during discovery: %s", response.Body.String())
	}
	selectionID, _ := out.Results[0]["id"].(string)
	evaluation := request(t, handler, "GET", "/api/torrents/search/"+selectionID+"/evaluation", nil)
	expect(t, evaluation, 200)
	var evaluated map[string]any
	if err := json.Unmarshal(evaluation.Body.Bytes(), &evaluated); err != nil {
		t.Fatal(err)
	}
	if evaluated["confidence"] != "high" {
		t.Fatalf("shared unfollowed metadata did not produce confidence: %s", evaluation.Body.String())
	}
	if ratio, ok := evaluated["mb_per_minute"].(float64); !ok || ratio < 49.9 || ratio > 50.1 {
		t.Fatalf("unexpected MB/min ratio: %v", evaluated["mb_per_minute"])
	}
	if tv.calls.Load() != 0 {
		t.Fatalf("local shared metadata unnecessarily called TVMaze %d times", tv.calls.Load())
	}
	var follows int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM profile_shows WHERE show_id='show-shared'").Scan(&follows); err != nil {
		t.Fatal(err)
	}
	if follows != 0 {
		t.Fatal("free-text metadata resolution followed the local show")
	}
}

func TestTorrentFreeTextEpisodeResolvesTVMazeWithoutPersistingOrFollowing(t *testing.T) {
	s, handler, _ := testServer(t, "disabled")
	tv := &torrentTargetTV{
		results: []metadata.SearchResult{{Score: 1, Show: metadata.Show{ID: 9090, Name: "Remote Show", Premiered: "2024-02-01", Runtime: 50}}},
		episodes: []metadata.Episode{{ID: 9904, Season: 2, Number: 4, Name: "Fourth", Runtime: 42}},
	}
	s.Metadata.Provider = tv
	size := int64(42 * 60 * 1024 * 1024)
	jackett := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<rss xmlns:torznab="http://torznab.com/schemas/2015/feed"><channel><item><title>Remote.Show.S02E04.1080p.WEB-DL-GROUP</title><guid>remote</guid><enclosure url="magnet:?xt=urn:btih:%s" length="%d"/><torznab:attr name="seeders" value="60"/></item></channel></rss>`, strings.Repeat("d", 40), size)
	}))
	defer jackett.Close()
	if err := s.settingsStore().Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.settingsStore().Save(context.Background(), "search", settings.Search{BaseURL: jackett.URL, APIKey: "PRIVATE-KEY", Enabled: true}, 1); err != nil {
		t.Fatal(err)
	}

	response := request(t, handler, "POST", "/api/torrents/search", map[string]any{"query": "Remote Show S02E04"})
	expect(t, response, 200)
	var out struct {
		Results []map[string]any `json:"results"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil || len(out.Results) != 1 {
		t.Fatalf("unexpected response: %s", response.Body.String())
	}
	if tv.searchCalls.Load() != 0 || tv.episodeCalls.Load() != 0 {
		t.Fatalf("manual discovery called TVMaze before expansion: search=%d episodes=%d", tv.searchCalls.Load(), tv.episodeCalls.Load())
	}
	selectionID, _ := out.Results[0]["id"].(string)
	evaluation := request(t, handler, "GET", "/api/torrents/search/"+selectionID+"/evaluation", nil)
	expect(t, evaluation, 200)
	var evaluated map[string]any
	if err := json.Unmarshal(evaluation.Body.Bytes(), &evaluated); err != nil {
		t.Fatal(err)
	}
	if evaluated["confidence"] != "high" {
		t.Fatalf("TVMaze-resolved evaluation did not receive confidence: %s", evaluation.Body.String())
	}
	if ratio, ok := evaluated["mb_per_minute"].(float64); !ok || ratio < 59.9 || ratio > 60.1 {
		t.Fatalf("unexpected TVMaze MB/min ratio: %v", evaluated["mb_per_minute"])
	}
	if tv.searchCalls.Load() != 1 || tv.episodeCalls.Load() != 1 {
		t.Fatalf("unexpected TVMaze calls: search=%d episodes=%d", tv.searchCalls.Load(), tv.episodeCalls.Load())
	}
	var persisted, followed int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM shows WHERE name='Remote Show'").Scan(&persisted); err != nil {
		t.Fatal(err)
	}
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM profile_shows ps JOIN shows s ON s.id=ps.show_id WHERE s.name='Remote Show'").Scan(&followed); err != nil {
		t.Fatal(err)
	}
	if persisted != 0 || followed != 0 {
		t.Fatalf("TVMaze search context changed library state: persisted=%d followed=%d", persisted, followed)
	}
}

func TestTorrentFreeTextEpisodeStillSearchesWhenTVMazeResolutionFails(t *testing.T) {
	s, handler, _ := testServer(t, "disabled")
	s.Metadata.Provider = &torrentTargetTV{searchErr: fmt.Errorf("tvmaze unavailable")}
	jackett := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<rss xmlns:torznab="http://torznab.com/schemas/2015/feed"><channel><item><title>Offline.Show.S01E01.1080p.WEB-DL-GROUP</title><guid>offline</guid><enclosure url="magnet:?xt=urn:btih:%s" length="1073741824"/><torznab:attr name="seeders" value="20"/></item></channel></rss>`, strings.Repeat("f", 40))
	}))
	defer jackett.Close()
	if err := s.settingsStore().Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.settingsStore().Save(context.Background(), "search", settings.Search{BaseURL: jackett.URL, APIKey: "PRIVATE-KEY", Enabled: true}, 1); err != nil {
		t.Fatal(err)
	}

	response := request(t, handler, "POST", "/api/torrents/search", map[string]any{"query": "Offline Show S01E01"})
	expect(t, response, 200)
	var out struct {
		Results []map[string]any `json:"results"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil || len(out.Results) != 1 {
		t.Fatalf("manual Jackett search did not survive TVMaze failure: %s", response.Body.String())
	}
	selectionID, _ := out.Results[0]["id"].(string)
	evaluation := request(t, handler, "GET", "/api/torrents/search/"+selectionID+"/evaluation", nil)
	expect(t, evaluation, 200)
	if strings.Contains(evaluation.Body.String(), "\"confidence\"") {
		t.Fatalf("failed metadata resolution should remain unscored: %s", evaluation.Body.String())
	}
}

func TestTorrentFreeTextEpisodeLeavesAmbiguousTVMazeMatchesUnscored(t *testing.T) {
	s, handler, _ := testServer(t, "disabled")
	tv := &torrentTargetTV{
		results: []metadata.SearchResult{
			{Score: 1, Show: metadata.Show{ID: 1001, Name: "Ambiguous Show", Premiered: "2020-01-01", Runtime: 45}},
			{Score: 0.9, Show: metadata.Show{ID: 1002, Name: "Ambiguous Show", Premiered: "2022-01-01", Runtime: 45}},
		},
	}
	s.Metadata.Provider = tv
	jackett := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<rss xmlns:torznab="http://torznab.com/schemas/2015/feed"><channel><item><title>Ambiguous.Show.S01E01.1080p.WEB-DL-GROUP</title><guid>ambiguous</guid><enclosure url="magnet:?xt=urn:btih:%s" length="1073741824"/><torznab:attr name="seeders" value="20"/></item></channel></rss>`, strings.Repeat("e", 40))
	}))
	defer jackett.Close()
	if err := s.settingsStore().Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.settingsStore().Save(context.Background(), "search", settings.Search{BaseURL: jackett.URL, APIKey: "PRIVATE-KEY", Enabled: true}, 1); err != nil {
		t.Fatal(err)
	}

	response := request(t, handler, "POST", "/api/torrents/search", map[string]any{"query": "Ambiguous Show S01E01"})
	expect(t, response, 200)
	var out struct {
		Results []map[string]any `json:"results"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil || len(out.Results) != 1 {
		t.Fatalf("unexpected response: %s", response.Body.String())
	}
	if tv.searchCalls.Load() != 0 || tv.episodeCalls.Load() != 0 {
		t.Fatalf("ambiguous discovery called TVMaze early: search=%d episodes=%d", tv.searchCalls.Load(), tv.episodeCalls.Load())
	}
	selectionID, _ := out.Results[0]["id"].(string)
	evaluation := request(t, handler, "GET", "/api/torrents/search/"+selectionID+"/evaluation", nil)
	expect(t, evaluation, 200)
	if strings.Contains(evaluation.Body.String(), "\"confidence\"") {
		t.Fatalf("ambiguous TVMaze match received authoritative confidence: %s", evaluation.Body.String())
	}
	if tv.episodeCalls.Load() != 0 {
		t.Fatalf("ambiguous show lookup should not fetch episodes, got %d calls", tv.episodeCalls.Load())
	}
}

func TestManualTorrentSubmissionCanOverridePreliminaryMetadataRejection(t *testing.T) {
	s, handler, _ := testServer(t, "disabled")
	seedTorrentEpisodeTarget(t, s)
	jackett := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/download" {
			w.Write([]byte(torrentFileFixture("Example.Show.S01E02.mkv")))
			return
		}
		fmt.Fprintf(w, `<rss xmlns:torznab="http://torznab.com/schemas/2015/feed"><channel><item><title>Example.Show.S01E02.1080p.WEB-DL.H264-GROUP</title><guid>one</guid><enclosure url="http://%s/download?apikey=PRIVATE-KEY" length="2048"/><torznab:attr name="tvmazeid" value="9999"/></item></channel></rss>`, r.Host)
	}))
	defer jackett.Close()
	var clientRequests atomic.Int32
	client := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/torrents/categories":
			fmt.Fprint(w, `{"tally":{}}`)
		case "/api/v2/torrents/add":
			clientRequests.Add(1)
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

	search := request(t, handler, "POST", "/api/torrents/search", map[string]any{
		"query": "Example Show S01E02", "episode_id": "episode-confidence",
	})
	expect(t, search, 200)
	var output struct {
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
	}
	if err := json.Unmarshal(search.Body.Bytes(), &output); err != nil || len(output.Results) != 1 {
		t.Fatalf("unexpected search response: %s", search.Body.String())
	}
	evaluation := request(t, handler, "GET", "/api/torrents/search/"+output.Results[0].ID+"/evaluation", nil)
	expect(t, evaluation, 200)
	if !strings.Contains(evaluation.Body.String(), `"confidence":"rejected"`) {
		t.Fatalf("fixture did not produce the intended preliminary rejection: %s", evaluation.Body.String())
	}

	response := request(t, handler, "POST", "/api/torrents/send", map[string]string{
		"selection": output.Results[0].ID, "idempotency_key": "manual-override-0001",
	})
	expect(t, response, 200)
	if clientRequests.Load() == 0 {
		t.Fatal("manual override did not reach qBittorrent after the actual payload matched")
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
			ID string `json:"id"`
		} `json:"results"`
	}
	if err := json.Unmarshal(search.Body.Bytes(), &output); err != nil || len(output.Results) != 1 || output.Results[0].ID == "" {
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
