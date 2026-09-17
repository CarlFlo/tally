package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/auth"
)

func TestCSRFAndAuthenticatedIsolation(t *testing.T) {
	s, h, _ := testServer(t, "local")
	hash, _ := auth.Hash("1234")
	_, e := s.DB.Exec("INSERT INTO local_credentials VALUES('profile-admin',?,0); INSERT INTO profiles(id,display_name,avatar,created_at,auth_method) VALUES('profile-member','Second','mint',?,'password'); INSERT INTO local_credentials VALUES('profile-member',?,0)", hash, time.Now().Unix(), hash)
	if e != nil {
		t.Fatal(e)
	}
	login := request(t, h, "POST", "/api/auth/login", map[string]string{"Profile": "profile-member", "Password": "1234"})
	expect(t, login, 200)
	cookies := login.Result().Cookies()
	cookies = append(cookies, &http.Cookie{Name: "tally_profile", Value: "profile-admin"})
	expect(t, request(t, h, "POST", "/api/profiles", map[string]string{"Name": "Intruder"}, cookies...), 403)
	expect(t, request(t, h, "PATCH", "/api/profile", map[string]string{"Name": "Only mine"}, cookies...), 200)
	var name string
	_ = s.DB.QueryRow("SELECT display_name FROM profiles WHERE id='profile-admin'").Scan(&name)
	if name != "My profile" {
		t.Fatal("browser-controlled profile ID was trusted")
	}
	r := httptest.NewRequest("PATCH", "/api/profile", strings.NewReader(`{"Name":"CSRF"}`))
	for _, c := range cookies {
		r.AddCookie(c)
	}
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	expect(t, w, 403)
	r = httptest.NewRequest("POST", "/api/auth/logout", strings.NewReader("{}"))
	r.Header.Set("Origin", "https://evil.example")
	r.Header.Set("X-Tally-CSRF", "1")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	expect(t, w, 403)
	expect(t, request(t, h, "POST", "/api/profiles/select", map[string]string{"Profile": "profile-admin"}, cookies...), 409)
}

func TestRestrictedSessionAndRevocation(t *testing.T) {
	s, h, _ := testServer(t, "local")
	hash, _ := auth.Hash("temporary")
	_, _ = s.DB.Exec("UPDATE profiles SET auth_method='password' WHERE id='profile-admin'; INSERT INTO local_credentials VALUES('profile-admin',?,1)", hash)
	w := request(t, h, "POST", "/api/auth/login", map[string]string{"Profile": "profile-admin", "Password": "temporary"})
	expect(t, w, 200)
	cookie := w.Result().Cookies()
	expect(t, request(t, h, "GET", "/api/shows", nil, cookie...), 403)
	expect(t, request(t, h, "POST", "/api/auth/password", map[string]string{"Password": "5678"}, cookie...), 200)
	expect(t, request(t, h, "GET", "/api/shows", nil, cookie...), 401)
	var stored string
	_ = s.DB.QueryRow("SELECT hash FROM local_credentials WHERE profile_id='profile-admin'").Scan(&stored)
	if !auth.Verify(stored, "5678") || auth.Verify(stored, "temporary") {
		t.Fatal("password replacement failed")
	}
}
