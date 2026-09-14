package api

import (
	"net/http"
	"strings"
	"testing"
)

func TestProfilesNeverReuseIDsAndProtectDefault(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	cookie := &http.Cookie{Name: "tally_profile", Value: "user0"}
	expect(t, request(t, h, "DELETE", "/api/profiles/user0", nil, cookie), 400)
	for _, want := range []string{"user1", "user2"} {
		w := request(t, h, "POST", "/api/profiles", map[string]any{"name": "Same name"}, cookie)
		expect(t, w, 201)
		if value(t, w, "id") != want {
			t.Fatal("unexpected profile ID")
		}
	}
	expect(t, request(t, h, "POST", "/api/profiles", map[string]any{"name": "Over limit"}, cookie), 400)
	expect(t, request(t, h, "DELETE", "/api/profiles/user1", nil, cookie), 200)
	w := request(t, h, "POST", "/api/profiles", map[string]any{"name": "Replacement"}, cookie)
	expect(t, w, 201)
	if value(t, w, "id") != "user3" {
		t.Fatal("deleted ID reused")
	}
	w = request(t, h, "GET", "/api/bootstrap", nil, &http.Cookie{Name: "tally_profile", Value: "user1"})
	if !strings.Contains(w.Body.String(), `"profile":null`) {
		t.Fatal("deleted remembered profile was trusted")
	}
	var count int
	_ = s.DB.QueryRow("SELECT COUNT(*) FROM profiles WHERE id='user0'").Scan(&count)
	if count != 1 {
		t.Fatal("default profile lost")
	}
}
