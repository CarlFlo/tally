package api

import (
	"net/http"
	"testing"
)

func TestLoginAppearanceIsStoredPerBrowser(t *testing.T) {
	s, h, _ := testServer(t, "local")
	response := request(t, h, "PATCH", "/api/browser/preferences", map[string]string{"theme": "light"})
	expect(t, response, 200)
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].Name != "tally_browser" {
		t.Fatal("missing private browser cookie")
	}
	var stored string
	if err := s.DB.QueryRow("SELECT theme FROM browser_preferences WHERE id=?", cookies[0].Value).Scan(&stored); err != nil || stored != "light" {
		t.Fatal("theme is not durable", err)
	}
	if got := value(t, request(t, h, "GET", "/api/bootstrap", nil, cookies...), "browser_theme"); got != "light" {
		t.Fatal(got)
	}
	if got := value(t, request(t, h, "GET", "/api/bootstrap", nil), "browser_theme"); got != "system" {
		t.Fatal("leaked between browsers", got)
	}
	expect(t, request(t, h, "PATCH", "/api/browser/preferences", map[string]string{"theme": "invalid"}, cookies...), 400)
	expect(t, request(t, h, "PATCH", "/api/browser/preferences", map[string]string{"theme": "dark"}, cookies...), 200)
	if got := value(t, request(t, h, "GET", "/api/bootstrap", nil, cookies...), "browser_theme"); got != "dark" {
		t.Fatal(got)
	}
	unknown := []*http.Cookie{{Name: "tally_browser", Value: "user-supplied"}}
	response = request(t, h, "PATCH", "/api/browser/preferences", map[string]string{"theme": "system"}, unknown...)
	if response.Result().Cookies()[0].Value == "user-supplied" {
		t.Fatal("accepted a client-selected browser identifier")
	}
}
