package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/torrent"
)

func TestTorrentSubmissionIdempotency(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	var submissions atomic.Int32
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+fixtureClientKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/v2/torrents/categories":
			w.Write([]byte(`{"tally":{"name":"tally","savePath":""}}`))
		case "/api/v2/torrents/add":
			if e := r.ParseForm(); e != nil || r.PostForm.Get("category") != torrent.TallyCategory {
				t.Error("submission was not assigned to the Tally category")
			}
			submissions.Add(1)
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer fake.Close()
	if _, e := s.Clients.Save(context.Background(), torrent.ClientUpdate{Adapter: "qbittorrent", Fields: map[string]string{"url": fake.URL, "api_key": fixtureClientKey}}); e != nil {
		t.Fatal(e)
	}
	magnet := "magnet:?xt=urn:btih:" + strings.Repeat("a", 40)
	data, _ := json.Marshal(torrent.SearchResult{Name: "Example", Magnet: magnet})
	s.selections.Store("choice", selection{Profile: "profile-admin", Data: data, Expires: time.Now().Add(time.Minute)})
	defer s.selections.Delete("choice")
	body := map[string]string{"selection": "choice", "idempotency_key": "0123456789abcdef"}
	expect(t, request(t, h, "POST", "/api/torrents/send", body), 200)
	expect(t, request(t, h, "POST", "/api/torrents/send", body), 200)
	if submissions.Load() != 1 {
		t.Fatal("submission replayed")
	}
	var downloaded int
	_ = s.DB.QueryRow("SELECT COUNT(*) FROM profile_episode_state WHERE downloaded=1").Scan(&downloaded)
	if downloaded != 0 {
		t.Fatal("send changed downloaded state")
	}
}
