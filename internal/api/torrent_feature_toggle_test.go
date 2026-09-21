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
	"time"

	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/settings"
	"github.com/CarlFlo/tally/internal/torrent"
)

func TestTorrentFeatureTogglesPreserveConnectionsAndEnforceActions(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	ctx := context.Background()
	if err := s.settingsStore().Ensure(ctx); err != nil {
		t.Fatal(err)
	}

	search := settings.Search{
		BaseURL: "http://unused.invalid",
		APIKey:  "saved-jackett-key",
		Enabled: false,
	}
	response := request(t, h, "PUT", "/api/settings/search", map[string]any{
		"data": search, "revision": 1,
	})
	expect(t, response, 200)
	configured, err := s.jackettConfigured(ctx)
	if err != nil {
		t.Fatal(err)
	}
	enabled, err := s.torrentSearchEnabled(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !configured || enabled {
		t.Fatal("search toggle changed Jackett configuration state")
	}
	response = request(t, h, "POST", "/api/torrents/search", map[string]any{"query": "Example"})
	expect(t, response, 400)
	if !strings.Contains(response.Body.String(), "disabled") {
		t.Fatal("disabled torrent search was not explained")
	}

	hash := strings.Repeat("b", 40)
	var adds atomic.Int32
	var deletedFiles atomic.Bool
	client := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+fixtureClientKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/torrents/categories":
			fmt.Fprint(w, `{"tally":{"name":"tally","savePath":""}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/torrents/add":
			adds.Add(1)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/torrents/info":
			fmt.Fprintf(w, `[{"hash":"%s","name":"Example","state":"downloading","progress":0.5,"size":1000,"downloaded":500,"dlspeed":10,"upspeed":2,"ratio":0.1,"added_on":1,"category":"tally"}]`, hash)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/torrents/delete":
			if err := r.ParseForm(); err != nil {
				t.Error(err)
				return
			}
			deletedFiles.Store(r.PostForm.Get("deleteFiles") == "true")
		default:
			http.NotFound(w, r)
		}
	}))
	defer client.Close()
	if _, err := s.Clients.Save(ctx, torrent.ClientUpdate{
		Adapter: "qbittorrent",
		Fields:  map[string]string{"url": client.URL, "api_key": fixtureClientKey},
	}); err != nil {
		t.Fatal(err)
	}

	magnet := "magnet:?xt=urn:btih:" + strings.Repeat("a", 40)
	selected, _ := json.Marshal(torrent.SearchResult{Name: "Example", Magnet: magnet})
	s.selections.Store("choice", selection{
		Profile: "profile-admin",
		Data:    selected,
		Expires: time.Now().Add(time.Minute),
	})
	defer s.selections.Delete("choice")

	response = request(t, h, "PUT", "/api/settings/torrent", map[string]any{
		"data": settings.Torrent{Enabled: false}, "revision": 1,
	})
	expect(t, response, 200)
	savedClient, err := s.Clients.Load(ctx)
	if err != nil || !savedClient.Configured() {
		t.Fatal("torrent toggle changed qBittorrent configuration state")
	}
	body := map[string]string{"selection": "choice", "idempotency_key": "0123456789abcdef"}
	response = request(t, h, "POST", "/api/torrents/send", body)
	expect(t, response, 400)
	if adds.Load() != 0 || !strings.Contains(response.Body.String(), "disabled") {
		t.Fatal("disabled torrent downloads still submitted a torrent")
	}
	expect(t, request(t, h, "GET", "/api/torrents/downloads", nil), 400)

	response = request(t, h, "PUT", "/api/settings/torrent", map[string]any{
		"data": settings.Torrent{Enabled: true}, "revision": 2,
	})
	expect(t, response, 200)
	expect(t, request(t, h, "POST", "/api/torrents/send", body), 200)
	if adds.Load() != 1 {
		t.Fatal("enabled torrent downloads did not submit the torrent")
	}
	expect(t, request(t, h, "DELETE", "/api/torrents/downloads/"+hash+"?delete_files=true", nil), 200)
	if !deletedFiles.Load() {
		t.Fatal("delete_files=true was not forwarded to qBittorrent")
	}
}

func TestTorrentFeatureReadsFailClosedOnStorageFailure(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{DB: db}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	if enabled, readErr := s.torrentSearchEnabled(ctx); readErr == nil || enabled {
		t.Fatalf("search settings failure did not fail closed: enabled=%t err=%v", enabled, readErr)
	}
	if enabled, readErr := s.torrentDownloadsEnabled(ctx); readErr == nil || enabled {
		t.Fatalf("download settings failure did not fail closed: enabled=%t err=%v", enabled, readErr)
	}
}
