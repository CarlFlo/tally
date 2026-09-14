package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestPasswordlessProfileSetupCannotReplaceExistingCredentials(t *testing.T) {
	s, h, _ := testServer(t, "local")
	if _, err := s.DB.Exec("INSERT INTO profiles VALUES('user1','Alex','mint',1)"); err != nil {
		t.Fatal(err)
	}
	boot := request(t, h, "GET", "/api/bootstrap", nil)
	expect(t, boot, 200)
	// SQLite EXISTS values are numeric in the generic profile response.
	var raw map[string]any
	if err := json.Unmarshal(boot.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	for _, p := range raw["profiles"].([]any) {
		if p.(map[string]any)["has_password"].(float64) != 0 {
			t.Fatal("unexpected password")
		}
	}
	password := " Spaces,Symbols! 1234 "
	result := request(t, h, "POST", "/api/auth/setup", map[string]any{"profile": "user1", "password": password})
	expect(t, result, 200)
	cookies := result.Result().Cookies()
	expect(t, request(t, h, "GET", "/api/shows", nil, cookies...), 200)
	expect(t, request(t, h, "POST", "/api/auth/setup", map[string]any{"profile": "user0", "password": "secret"}, cookies...), 409)
	expect(t, request(t, h, "POST", "/api/auth/setup", map[string]any{"profile": "user1", "password": "replacement"}), 400)
	expect(t, request(t, h, "POST", "/api/auth/login", map[string]any{"profile": "user1", "password": password}), 200)
	var count int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM local_credentials WHERE profile_id='user0'").Scan(&count); err != nil || count != 0 {
		t.Fatal("other profile was claimed", err)
	}
	expect(t, request(t, h, "POST", "/api/auth/setup", map[string]any{"profile": "missing", "password": "1234"}), 400)
	expect(t, request(t, h, "POST", "/api/auth/setup", map[string]any{"profile": "user0", "password": "test\npassword"}), 400)
}

func TestPublicProfileRegistrationIsBoundedAndRequiresPasswordInLocalMode(t *testing.T) {
	for _, mode := range []string{"local", "disabled"} {
		t.Run(mode, func(t *testing.T) {
			s, h, _ := testServer(t, mode)
			signedOut := &http.Cookie{Name: "tally_profile", Value: "signed-out"}
			if mode == "local" {
				expect(t, request(t, h, "POST", "/api/auth/register", map[string]any{"name": "Alex"}, signedOut), 400)
			}
			expect(t, request(t, h, "POST", "/api/auth/register", map[string]any{"name": "bad\nname", "password": "Pass123!"}, signedOut), 400)
			first := request(t, h, "POST", "/api/auth/register", map[string]any{"name": "  Alex  ", "password": "Pass123!"}, signedOut)
			expect(t, first, 201)
			if value(t, first, "id") != "user1" || value(t, first, "display_name") != "Alex" {
				t.Fatal("invalid profile identity")
			}
			expect(t, request(t, h, "POST", "/api/auth/register", map[string]any{"name": "Switch", "password": "Pass123!"}, first.Result().Cookies()...), 409)
			second := request(t, h, "POST", "/api/auth/register", map[string]any{"name": "Sam", "password": "Pass123!"}, signedOut)
			expect(t, second, 201)
			full := request(t, h, "POST", "/api/auth/register", map[string]any{"name": "Overflow", "password": "Pass123!"}, signedOut)
			expect(t, full, 400)
			if !strings.Contains(full.Body.String(), "maximum") {
				t.Fatal("missing profile limit message")
			}
			if _, err := s.DB.Exec("DELETE FROM profiles WHERE id='user1'"); err != nil {
				t.Fatal(err)
			}
			next := request(t, h, "POST", "/api/auth/register", map[string]any{"name": "New", "password": "Pass123!"}, signedOut)
			expect(t, next, 201)
			if value(t, next, "id") != "user3" {
				t.Fatal("reused immutable ID")
			}
		})
	}
}
