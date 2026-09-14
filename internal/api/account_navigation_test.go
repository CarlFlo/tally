package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func profileCookie(t *testing.T, response *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == "tally_profile" {
			return cookie
		}
	}
	t.Fatal("profile choice cookie missing")
	return nil
}

func TestExplicitSignOutPreventsSingleProfileAutoEntry(t *testing.T) {
	_, h, _ := testServer(t, "disabled")
	initial := request(t, h, "GET", "/api/bootstrap", nil)
	expect(t, initial, 200)
	current := profileCookie(t, initial)
	if current.Value != "user0" {
		t.Fatal("first visit should still enter the single profile")
	}
	logout := request(t, h, "POST", "/api/auth/logout", nil, current)
	expect(t, logout, 200)
	left := profileCookie(t, logout)
	if !left.HttpOnly || left.MaxAge < 365*24*3600 {
		t.Fatal("explicit sign-out must be remembered")
	}
	for range 2 {
		boot := request(t, h, "GET", "/api/bootstrap", nil, left)
		if !strings.Contains(boot.Body.String(), `"profile":null`) {
			t.Fatal("reload automatically re-entered the signed-out profile")
		}
		expect(t, request(t, h, "GET", "/api/shows", nil, left), 401)
	}
	expect(t, request(t, h, "POST", "/api/auth/logout", nil, left), 200)
	selected := request(t, h, "POST", "/api/profiles/select", map[string]string{"profile": "user0"}, left)
	expect(t, selected, 200)
	expect(t, request(t, h, "GET", "/api/shows", nil, profileCookie(t, selected)), 200)
	deleted := request(t, h, "GET", "/api/bootstrap", nil, &http.Cookie{Name: "tally_profile", Value: "user99"})
	if !strings.Contains(deleted.Body.String(), `"profile":null`) {
		t.Fatal("invalid remembered profile should open selection even with one remaining profile")
	}
}

func TestChoosingAnotherProfileRequiresSignOut(t *testing.T) {
	_, h, _ := testServer(t, "disabled")
	current := &http.Cookie{Name: "tally_profile", Value: "user0"}
	expect(t, request(t, h, "POST", "/api/profiles", map[string]string{"name": "Alex"}, current), 201)
	expect(t, request(t, h, "POST", "/api/profiles/select", map[string]string{"profile": "user1"}, current), 409)
	left := profileCookie(t, request(t, h, "POST", "/api/auth/logout", nil, current))
	selected := request(t, h, "POST", "/api/profiles/select", map[string]string{"profile": "user1"}, left)
	expect(t, selected, 200)
	if profileCookie(t, selected).Value != "user1" {
		t.Fatal("could not choose another profile after signing out")
	}
}

func TestLocalSignOutRevokesSessionBeforeAnotherSignIn(t *testing.T) {
	s, h, _ := testServer(t, "local")
	ctx := context.Background()
	hash, err := auth.Hash("1234")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec("INSERT INTO profiles VALUES('user1','Alex','mint',?)", time.Now().Unix()); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec("INSERT INTO local_credentials VALUES('user1',?,0)", hash); err != nil {
		t.Fatal(err)
	}
	created := httptest.NewRecorder()
	if err = s.Auth.NewSession(ctx, created, httptest.NewRequest("GET", "/", nil), "user0", false); err != nil {
		t.Fatal(err)
	}
	cookies := created.Result().Cookies()
	expect(t, request(t, h, "POST", "/api/auth/login", map[string]string{"profile": "user1", "password": "1234"}, cookies...), 409)
	expect(t, request(t, h, "POST", "/api/auth/logout", nil, cookies...), 200)
	expect(t, request(t, h, "GET", "/api/shows", nil, cookies...), 401)
	expect(t, request(t, h, "POST", "/api/auth/logout", nil, cookies...), 200)
	login := request(t, h, "POST", "/api/auth/login", map[string]string{"profile": "user1", "password": "1234"}, cookies...)
	expect(t, login, 200)
	expect(t, request(t, h, "GET", "/api/shows", nil, login.Result().Cookies()...), 200)
}

func TestOIDCDoesNotReplaceActiveAccount(t *testing.T) {
	s, h, _ := testServer(t, "oidc")
	created := httptest.NewRecorder()
	if err := s.Auth.NewSession(context.Background(), created, httptest.NewRequest("GET", "/", nil), "user0", false); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/auth/oidc/start", "/auth/oidc/callback?state=old"} {
		response := request(t, h, "GET", path, nil, created.Result().Cookies()...)
		expect(t, response, 303)
		if response.Header().Get("Location") != "/profile" {
			t.Fatal("active OIDC account should return to its settings")
		}
	}
}
