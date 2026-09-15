package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CarlFlo/mediaManager/internal/settings"
	"github.com/CarlFlo/mediaManager/internal/torrent"
)

func TestEditableSettingsPersistAndIgnoreLaterEnvironment(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	s.Config.Timezone = "Europe/Stockholm"
	ctx := context.Background()
	if e := s.settingsStore().Ensure(ctx); e != nil {
		t.Fatal(e)
	}
	input := map[string]any{"revision": 1, "data": settings.Search{BaseURL: "http://fixture.invalid", APIKey: "test-only-secret", Enabled: true}}
	expect(t, request(t, h, "PUT", "/api/settings/search", input), 200)
	if s.searchProvider("jackett") == nil {
		t.Fatal("saved Jackett connection not applied")
	}
	expect(t, request(t, h, "PUT", "/api/settings/search", input), 409)
	if e := s.settingsStore().Ensure(ctx); e != nil {
		t.Fatal(e)
	}
	expect(t, request(t, h, "GET", "/api/settings/search", nil), 200)
	notificationDefaults := request(t, h, "GET", "/api/settings/notifications", nil)
	expect(t, notificationDefaults, 200)
	if !strings.Contains(notificationDefaults.Body.String(), `"timezone":"Europe/Stockholm"`) || !strings.Contains(notificationDefaults.Body.String(), `"server_timezone":"Europe/Stockholm"`) {
		t.Fatal("notification defaults did not use deployment timezone", notificationDefaults.Body.String())
	}
	preview := request(t, h, "POST", "/api/settings/notifications/preview", map[string]string{"delivery_time": "09:00"})
	expect(t, preview, 200)
	if !strings.Contains(preview.Body.String(), `"server_timezone":"Europe/Stockholm"`) || !strings.Contains(preview.Body.String(), `"next_delivery":`) {
		t.Fatal("notification time preview did not use deployment timezone", preview.Body.String())
	}
	if s.searchProvider("jackett") == nil || s.searchProvider("torznab") != nil {
		t.Fatal("restart overwrote UI settings")
	}
	safe := request(t, h, "GET", "/api/settings", nil)
	if strings.Contains(safe.Body.String(), "test-only-secret") {
		t.Fatal("general settings leaked key")
	}
	if !strings.Contains(safe.Body.String(), `"TZ":"Europe/Stockholm"`) || strings.Contains(safe.Body.String(), "OIDC_SECRET") {
		t.Fatal("safe environment view is missing or leaked secret metadata", safe.Body.String())
	}
	expect(t, request(t, h, "PUT", "/api/settings/notifications", map[string]any{"revision": 1, "data": settings.Webhook{Enabled: true, URL: "file:///unsafe"}}), 400)
	expect(t, request(t, h, "PUT", "/api/settings/notifications", map[string]any{"revision": 1, "data": settings.Webhook{Enabled: true, URL: "http://webhook.invalid/path"}}), 200)
	expect(t, request(t, h, "PUT", "/api/settings/notifications", map[string]any{"revision": 2, "data": settings.Webhook{Enabled: false, URL: "http://webhook.invalid/path"}}), 200)
	safe = request(t, h, "GET", "/api/settings", nil)
	if !strings.Contains(safe.Body.String(), `"webhook_configured":false`) {
		t.Fatal("disabled webhook still enabled")
	}
}
func TestConnectionSecretsVisibleOnlyInExplicitOperatorView(t *testing.T) {
	s, h, _ := testServer(t, "local")
	ctx := context.Background()
	s.settingsStore().Ensure(ctx)
	if _, e := s.Clients.Save(ctx, torrent.ClientUpdate{Adapter: "qbittorrent", Fields: map[string]string{"url": "http://unused.invalid", "api_key": fixtureClientKey}}); e != nil {
		t.Fatal(e)
	}
	owner := httptest.NewRecorder()
	s.Auth.NewSession(ctx, owner, httptest.NewRequest("GET", "/", nil), "profile-admin", false)
	response := request(t, h, "GET", "/api/downloader?reveal=1", nil, owner.Result().Cookies()...)
	expect(t, response, 200)
	if !strings.Contains(response.Body.String(), fixtureClientKey) || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("explicit reveal unavailable or cacheable")
	}
	s.DB.Exec("INSERT INTO profiles VALUES('profile-member','Other','mint',1)")
	ordinary := httptest.NewRecorder()
	s.Auth.NewSession(ctx, ordinary, httptest.NewRequest("GET", "/", nil), "profile-member", false)
	for _, path := range []string{"/api/downloader?reveal=1", "/api/settings/search", "/api/settings/notifications", "/api/settings/scheduling"} {
		expect(t, request(t, h, "GET", path, nil, ordinary.Result().Cookies()...), 403)
	}
}
func TestFavoritesEpisodeResetAndPersistentViewPreferences(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	response := request(t, h, "POST", "/api/shows", map[string]int{"tvmaze_id": 7})
	expect(t, response, 201)
	id := value(t, response, "id")
	expect(t, request(t, h, "PATCH", "/api/shows/"+id+"/favorite", map[string]bool{"favorite": true}), 200)
	s.DB.Exec("INSERT INTO profiles VALUES('profile-member','Other','mint',1)")
	other := &http.Cookie{Name: "tally_profile", Value: "profile-member"}
	owner := &http.Cookie{Name: "tally_profile", Value: "profile-admin"}
	expect(t, request(t, h, "POST", "/api/shows", map[string]int{"tvmaze_id": 7}, other), 201)
	if strings.Contains(request(t, h, "GET", "/api/shows", nil, other).Body.String(), `"favorite":1`) {
		t.Fatal("favorite leaked between profiles")
	}
	var ep string
	s.DB.QueryRow("SELECT id FROM episodes LIMIT 1").Scan(&ep)
	for _, v := range []bool{true, false} {
		expect(t, request(t, h, "PATCH", "/api/episodes/"+ep, map[string]bool{"downloaded": v, "watched": v}, owner), 200)
	}
	var downloaded, watched int
	s.DB.QueryRow("SELECT downloaded,watched FROM profile_episode_state WHERE profile_id='profile-admin' AND episode_id=?", ep).Scan(&downloaded, &watched)
	if downloaded != 0 || watched != 0 {
		t.Fatal("states could not be cleared")
	}
	expect(t, request(t, h, "PATCH", "/api/preferences", map[string]any{"debug_mode": true, "request_limit": 50, "job_status_filter": "failed"}, owner), 200)
	expect(t, request(t, h, "PATCH", "/api/preferences", map[string]any{"scan_limit": 100}, owner), 200)
	var prefs map[string]any
	var raw string
	s.DB.QueryRow("SELECT data FROM profile_preferences WHERE profile_id='profile-admin'").Scan(&raw)
	json.Unmarshal([]byte(raw), &prefs)
	if prefs["request_limit"] != float64(50) || prefs["scan_limit"] != float64(100) || prefs["debug_mode"] != true {
		t.Fatal("preferences were not merged persistently")
	}
	expect(t, request(t, h, "GET", "/api/statistics?request_limit=1000", nil, owner), 400)
}
