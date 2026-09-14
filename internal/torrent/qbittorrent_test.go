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

	"github.com/CarlFlo/mediaManager/internal/providers"
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
	var requests, magnets, files atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Header.Get("Authorization") != "Bearer "+testAPIKey {
			t.Error("missing Bearer API key")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Header.Get("Cookie") != "" || r.URL.RawQuery != "" {
			t.Error("cookie or query used for authentication")
		}
		http.SetCookie(w, &http.Cookie{Name: "SID", Value: "must-not-be-reused"})
		switch {
		case r.Method == "GET" && r.URL.Path == "/proxy/api/v2/app/version":
			fmt.Fprint(w, "v5.2.0")
		case r.Method == "POST" && r.URL.Path == "/proxy/api/v2/torrents/add":
			if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
				f, header, e := r.FormFile("torrents")
				if e != nil {
					t.Error(e)
					w.WriteHeader(400)
					return
				}
				defer f.Close()
				data, e := io.ReadAll(f)
				if e != nil || string(data) != file || header.Filename != "selected.torrent" {
					t.Error("torrent file was changed")
				}
				if len(r.MultipartForm.Value) != 0 || len(r.MultipartForm.File) != 1 {
					t.Error("unexpected multipart fields")
				}
				files.Add(1)
			} else {
				if e := r.ParseForm(); e != nil || r.PostForm.Get("urls") != magnet || len(r.PostForm) != 1 {
					t.Error("magnet or form fields were changed")
				}
				magnets.Add(1)
			}
			fmt.Fprint(w, "Ok.")
		default:
			t.Error("unexpected request; API keys must not call login")
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := &QBittorrent{Control: qbtControl(t), URL: server.URL + "/proxy/", APIKey: testAPIKey}
	ctx := context.Background()
	if e := client.TestConnection(ctx); e != nil {
		t.Fatal(e)
	}
	if requests.Load() != 1 || magnets.Load() != 0 || files.Load() != 0 {
		t.Fatal("connection test logged in or sent a torrent")
	}
	if e := client.AddMagnet(ctx, magnet); e != nil {
		t.Fatal(e)
	}
	if e := client.AddTorrent(ctx, []byte(file)); e != nil {
		t.Fatal(e)
	}
	if requests.Load() != 3 || magnets.Load() != 1 || files.Load() != 1 {
		t.Fatal("expected one request per operation")
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
				fmt.Fprint(w, testAPIKey) // An upstream echo must not leak into errors.
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
	client := &QBittorrent{URL: "http://unused.example"} // No coordinator: validation must happen before outbound work.
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
