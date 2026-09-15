package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/CarlFlo/mediaManager/internal/torrent"
)

const fixtureClientKey = "qbt_0123456789abcdefghijklmnopqr"

func TestClientSetupTestSaveAndSecretRedaction(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	var adds atomic.Int32
	var versionMode atomic.Int32
	client := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") != "http://"+r.Host {
			t.Error("incorrect origin for reverse-proxy path")
		}
		if r.Header.Get("Cookie") != "" || r.URL.RawQuery != "" {
			t.Error("authentication must use only the Bearer header")
		}
		if r.Header.Get("Authorization") != "Bearer "+fixtureClientKey {
			http.Error(w, "invalid key", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/qb/api/v2/app/version":
			if versionMode.Load() == 1 {
				w.Write([]byte("<html>Login page</html>"))
				return
			}
			w.Write([]byte("v5.2.0"))
		case "/qb/api/v2/torrents/add":
			adds.Add(1)
			w.Write([]byte("Ok."))
		default:
			http.NotFound(w, r)
		}
	}))
	defer client.Close()
	fields := map[string]string{"url": client.URL + "/qb", "api_key": "qbt_aaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	in := torrent.ClientUpdate{Adapter: "qbittorrent", Fields: fields}
	expect(t, request(t, h, "POST", "/api/downloader/test", in), 502)
	fields["api_key"] = fixtureClientKey
	result := request(t, h, "POST", "/api/downloader/test", in)
	expect(t, result, 200)
	if !strings.Contains(result.Body.String(), "Connected to qBittorrent") {
		t.Fatal("test did not report success")
	}
	saved, _ := s.Clients.Load(context.Background())
	if saved.Adapter != "" || saved.Revision != 0 {
		t.Fatal("connection test saved the draft")
	}
	versionMode.Store(1)
	expect(t, request(t, h, "POST", "/api/downloader/test", in), 502)
	versionMode.Store(0)
	response := request(t, h, "PUT", "/api/downloader", in)
	expect(t, response, 200)
	for _, path := range []string{"/api/downloader", "/api/settings"} {
		response = request(t, h, "GET", path, nil)
		expect(t, response, 200)
		if strings.Contains(response.Body.String(), fixtureClientKey) {
			t.Fatal("saved API key leaked")
		}
	}
	if !strings.Contains(request(t, h, "GET", "/api/downloader", nil).Body.String(), `"api_key":true`) {
		t.Fatal("saved API key flag missing")
	}
	saved, _ = s.Clients.Load(context.Background())
	in.Revision = saved.Revision
	delete(fields, "api_key")
	expect(t, request(t, h, "POST", "/api/downloader/test", in), 200)
	fields["url"] = client.URL + "/another-target"
	// Changing the endpoint keeps the saved API key; validation reaches the
	// endpoint and reports its connection failure instead of requiring re-entry.
	expect(t, request(t, h, "POST", "/api/downloader/test", in), 502)
	if adds.Load() != 0 {
		t.Fatal("connection test sent a torrent")
	}
}

func TestClientManagementRequiresOperator(t *testing.T) {
	s, h, _ := testServer(t, "local")
	recorder := httptest.NewRecorder()
	if _, e := s.DB.Exec("INSERT INTO profiles VALUES('profile-member','Alex','mint',1)"); e != nil {
		t.Fatal(e)
	}
	if e := s.Auth.NewSession(context.Background(), recorder, httptest.NewRequest("GET", "/", nil), "profile-member", false); e != nil {
		t.Fatal(e)
	}
	for _, endpoint := range []struct{ method, path string }{{"GET", "/api/downloader"}, {"PUT", "/api/downloader"}, {"POST", "/api/downloader/test"}} {
		expect(t, request(t, h, endpoint.method, endpoint.path, map[string]any{}, recorder.Result().Cookies()...), 403)
	}
	expect(t, request(t, h, "GET", "/api/settings", nil, recorder.Result().Cookies()...), 403)
	response := request(t, h, "GET", "/api/capabilities", nil, recorder.Result().Cookies()...)
	expect(t, response, 200)
	var safe map[string]any
	if e := json.Unmarshal(response.Body.Bytes(), &safe); e != nil {
		t.Fatal(e)
	}
	if safe["operator"] != false {
		t.Fatal("ordinary profile received management permission")
	}
}

func TestOldClientSettingsDisableSendingUntilAPIKeyIsSaved(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	if _, e := s.DB.Exec(`UPDATE download_client_settings SET adapter='qbittorrent',fields=?,revision=1 WHERE id=1`, `{"url":"http://unused.example","username":"old-user","password":"old-password"}`); e != nil {
		t.Fatal(e)
	}
	response := request(t, h, "GET", "/api/settings", nil)
	expect(t, response, 200)
	if !strings.Contains(response.Body.String(), `"downloader_configured":false`) {
		t.Fatal("old credentials made sending available")
	}
	response = request(t, h, "POST", "/api/torrents/send", map[string]string{})
	expect(t, response, 400)
	if !strings.Contains(response.Body.String(), "API key") {
		t.Fatal("missing key was not explained")
	}
}
