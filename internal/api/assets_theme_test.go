package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func TestAssetsUseActiveProfileThemeBeforePagePaint(t *testing.T) {
	s, _, _ := testServer(t, "local")
	s.Assets = fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte(`<html data-theme="__TALLY_THEME__"></html>`)}}
	if _, err := s.DB.Exec(`INSERT INTO profile_preferences(profile_id,data) VALUES('profile-admin','{"theme":"dark"}')`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec(`INSERT INTO browser_preferences(id,theme,updated_at) VALUES('browser-id','light',?)`, time.Now().Unix()); err != nil {
		t.Fatal(err)
	}
	issued := httptest.NewRecorder()
	if err := s.Auth.NewSession(context.Background(), issued, httptest.NewRequest("GET", "/", nil), "profile-admin", false); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/calendar", nil)
	for _, cookie := range issued.Result().Cookies() {
		request.AddCookie(cookie)
	}
	request.AddCookie(&http.Cookie{Name: "tally_browser", Value: "browser-id"})
	response := httptest.NewRecorder()
	s.assets(response, request)
	if !strings.Contains(response.Body.String(), `data-theme="dark"`) {
		t.Fatalf("page did not start with the active profile theme: %q", response.Body.String())
	}
}
