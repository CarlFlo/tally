package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/auth"
)

func sessionCookie(t *testing.T, response *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == "tally_session" && cookie.Value != "" {
			return cookie
		}
	}
	t.Fatal("session cookie missing")
	return nil
}

func TestNoAuthProfileRequiresExplicitSelectionAfterSignOut(t *testing.T) {
	_, h, _ := testServer(t, "local")
	initial := request(t, h, "GET", "/api/bootstrap", nil)
	expect(t, initial, 200)
	if !strings.Contains(initial.Body.String(), `"profile":null`) {
		t.Fatal("an unauthenticated visit should open profile selection")
	}

	selected := request(t, h, "POST", "/api/profiles/select", map[string]string{"profile": "profile-admin"})
	expect(t, selected, 200)
	current := sessionCookie(t, selected)
	expect(t, request(t, h, "GET", "/api/shows", nil, current), 200)

	logout := request(t, h, "POST", "/api/auth/logout", nil, current)
	expect(t, logout, 200)
	expect(t, request(t, h, "GET", "/api/shows", nil, current), 401)
	boot := request(t, h, "GET", "/api/bootstrap", nil, current)
	if !strings.Contains(boot.Body.String(), `"profile":null`) {
		t.Fatal("sign-out should return to profile selection")
	}
}

func TestChoosingAnotherProfileRequiresSignOut(t *testing.T) {
	_, h, _ := testServer(t, "local")
	adminLogin := request(t, h, "POST", "/api/profiles/select", map[string]string{"profile": "profile-admin"})
	expect(t, adminLogin, 200)
	admin := sessionCookie(t, adminLogin)
	created := request(t, h, "POST", "/api/profiles", map[string]string{"name": "Alex", "auth_method": "none"}, admin)
	expect(t, created, 201)
	profileID := value(t, created, "id")
	expect(t, request(t, h, "POST", "/api/profiles/select", map[string]string{"profile": profileID}, admin), 409)
	expect(t, request(t, h, "POST", "/api/auth/logout", nil, admin), 200)
	selected := request(t, h, "POST", "/api/profiles/select", map[string]string{"profile": profileID})
	expect(t, selected, 200)
	expect(t, request(t, h, "GET", "/api/shows", nil, sessionCookie(t, selected)), 200)
}

func TestPasswordSignOutRevokesSessionBeforeAnotherSignIn(t *testing.T) {
	s, h, _ := testServer(t, "local")
	ctx := context.Background()
	hash, err := auth.Hash("1234")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec("INSERT INTO profiles(id,display_name,avatar,created_at,auth_method) VALUES('profile-member','Alex','mint',?,'password')", time.Now().Unix()); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec("INSERT INTO local_credentials VALUES('profile-member',?,0)", hash); err != nil {
		t.Fatal(err)
	}
	created := httptest.NewRecorder()
	if err = s.Auth.NewSession(ctx, created, httptest.NewRequest("GET", "/", nil), "profile-admin", false); err != nil {
		t.Fatal(err)
	}
	cookies := created.Result().Cookies()
	expect(t, request(t, h, "POST", "/api/auth/login", map[string]string{"profile": "profile-member", "password": "1234"}, cookies...), 409)
	expect(t, request(t, h, "POST", "/api/auth/logout", nil, cookies...), 200)
	expect(t, request(t, h, "GET", "/api/shows", nil, cookies...), 401)
	login := request(t, h, "POST", "/api/auth/login", map[string]string{"profile": "profile-member", "password": "1234"})
	expect(t, login, 200)
	expect(t, request(t, h, "GET", "/api/shows", nil, login.Result().Cookies()...), 200)
}
