package torrent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/CarlFlo/tally/internal/providers"
)

const testAPIKey = "qbt_0123456789abcdefghijklmnopqr"

func qbtControl(t *testing.T) *providers.Coordinator {
	t.Helper()
	store := clientStore(t)
	control, e := providers.New(context.Background(), store.DB, t.TempDir(), 2, 2)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { control.Close() })
	return control
}

func TestQBittorrentBearerForTestsMagnetsAndFiles(t *testing.T) {
	magnet := "magnet:?xt=urn:btih:" + strings.Repeat("a", 40)
	file := "synthetic torrent file for protocol verification"
	var magnets, files atomic.Int32
	var categoryCreated atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+testAPIKey {
			t.Error("missing Bearer API key")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Header.Get("Cookie") != "" {
			t.Error("cookie used for authentication")
		}
		http.SetCookie(w, &http.Cookie{Name: "SID", Value: "must-not-be-reused"})
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/proxy/api/v2/app/version":
			fmt.Fprint(w, "v5.2.0")
		case r.Method == http.MethodGet && r.URL.Path == "/proxy/api/v2/torrents/categories":
			if categoryCreated.Load() {
				fmt.Fprint(w, `{"tally":{"name":"tally","savePath":""}}`)
			} else {
				fmt.Fprint(w, `{}`)
			}
		case r.Method == http.MethodPost && r.URL.Path == "/proxy/api/v2/torrents/createCategory":
			if e := r.ParseForm(); e != nil || r.PostForm.Get("category") != TallyCategory {
				t.Error("Tally category was not created")
			}
			categoryCreated.Store(true)
		case r.Method == http.MethodPost && r.URL.Path == "/proxy/api/v2/torrents/add":
			if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
				f, header, e := r.FormFile("torrents")
				if e != nil {
					t.Error(e)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				defer f.Close()
				data, e := io.ReadAll(f)
				if e != nil || string(data) != file || header.Filename != "selected.torrent" {
					t.Error("torrent file was changed")
				}
				if values := r.MultipartForm.Value["category"]; len(values) != 1 || values[0] != TallyCategory {
					t.Error("torrent file missing Tally category")
				}
				files.Add(1)
			} else {
				if e := r.ParseForm(); e != nil || r.PostForm.Get("urls") != magnet || r.PostForm.Get("category") != TallyCategory {
					t.Error("magnet or category was changed")
				}
				magnets.Add(1)
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &QBittorrent{Control: qbtControl(t), URL: server.URL + "/proxy/", APIKey: testAPIKey}
	ctx := context.Background()
	if e := client.TestConnection(ctx); e != nil {
		t.Fatal(e)
	}
	if e := client.AddMagnet(ctx, magnet); e != nil {
		t.Fatal(e)
	}
	if e := client.AddTorrent(ctx, []byte(file)); e != nil {
		t.Fatal(e)
	}
	if magnets.Load() != 1 || files.Load() != 1 || !categoryCreated.Load() {
		t.Fatal("expected both Tally-categorized submissions")
	}
}

func TestQBittorrentListsAndControlsTallyDownloads(t *testing.T) {
	hash := strings.Repeat("b", 40)
	var actions atomic.Int32
	var deletedFiles atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+testAPIKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/torrents/info":
			if r.URL.Query().Get("category") != TallyCategory {
				t.Error("downloads were not scoped to the Tally category")
			}
			fmt.Fprintf(w, `[{"hash":"%s","name":"Example","state":"downloading","progress":0.5,"size":1000,"downloaded":500,"dlspeed":120,"upspeed":30,"ratio":0.25,"added_on":42,"category":"tally"}]`, hash)
		case r.Method == http.MethodPost && (r.URL.Path == "/api/v2/torrents/stop" || r.URL.Path == "/api/v2/torrents/start" || r.URL.Path == "/api/v2/torrents/delete"):
			if e := r.ParseForm(); e != nil || r.PostForm.Get("hashes") != hash {
				t.Error("torrent control used the wrong hash")
			}
			if r.URL.Path == "/api/v2/torrents/delete" {
				switch r.PostForm.Get("deleteFiles") {
				case "false":
				case "true":
					deletedFiles.Store(true)
				default:
					t.Error("torrent delete did not specify file handling")
				}
			}
			actions.Add(1)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &QBittorrent{Control: qbtControl(t), URL: server.URL, APIKey: testAPIKey}
	snapshot, e := client.Downloads(context.Background(), TallyCategory)
	if e != nil {
		t.Fatal(e)
	}
	if len(snapshot.Torrents) != 1 || snapshot.Torrents[0].Hash != hash {
		t.Fatal("Tally torrent was not listed")
	}
	if snapshot.Stats.Total != 1 || snapshot.Stats.Active != 1 || snapshot.Stats.DownloadSpeed != 120 || snapshot.Stats.UploadSpeed != 30 {
		t.Fatalf("unexpected download stats: %#v", snapshot.Stats)
	}
	if e := client.Stop(context.Background(), hash); e != nil {
		t.Fatal(e)
	}
	if e := client.Start(context.Background(), hash); e != nil {
		t.Fatal(e)
	}
	if e := client.Remove(context.Background(), hash, false); e != nil {
		t.Fatal(e)
	}
	if e := client.Remove(context.Background(), hash, true); e != nil {
		t.Fatal(e)
	}
	if actions.Load() != 4 || !deletedFiles.Load() {
		t.Fatal("expected stop, start, and both remove modes")
	}
}

func TestQBittorrentRejectsAuthenticationAndRedirectWithoutRetry(t *testing.T) {
	for _, status := range []int{401, 403, 302} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var requests, forwarded atomic.Int32
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { forwarded.Add(1) }))
			defer target.Close()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.Header().Set("Location", target.URL)
				w.WriteHeader(status)
				fmt.Fprint(w, testAPIKey)
			}))
			defer server.Close()
			client := &QBittorrent{Control: qbtControl(t), URL: server.URL, APIKey: testAPIKey}
			e := client.TestConnection(context.Background())
			if e == nil || !strings.Contains(e.Error(), fmt.Sprint(status)) || strings.Contains(e.Error(), testAPIKey) {
				t.Fatal("unsafe or missing HTTP error")
			}
			if requests.Load() != 1 || forwarded.Load() != 0 {
				t.Fatal("authentication retried or credentials followed redirect")
			}
		})
	}
}

func TestQBittorrentMissingKeyDoesNotMakeRequests(t *testing.T) {
	client := &QBittorrent{URL: "http://unused.example"}
	if e := client.TestConnection(context.Background()); e == nil {
		t.Fatal("missing key accepted")
	}
	if e := client.AddMagnet(context.Background(), "magnet:?xt=urn:btih:"+strings.Repeat("a", 40)); e == nil {
		t.Fatal("missing key accepted")
	}
	if e := client.AddTorrent(context.Background(), []byte("fixture")); e == nil {
		t.Fatal("missing key accepted")
	}
}
