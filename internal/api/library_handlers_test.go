package api

import (
	"net/http"
	"strings"
	"testing"
)

func TestSharedMetadataAndPersonalEpisodeState(t *testing.T) {
	s, h, tv := testServer(t, "disabled")
	zero := &http.Cookie{Name: "tally_profile", Value: "profile-admin"}
	w := request(t, h, "POST", "/api/shows", map[string]any{"tvmaze_id": 7}, zero)
	expect(t, w, 201)
	id := value(t, w, "id")
	created := request(t, h, "POST", "/api/profiles", map[string]any{"name": "Second"}, zero)
	expect(t, created, 201)
	one := &http.Cookie{Name: "tally_profile", Value: value(t, created, "id")}
	w = request(t, h, "POST", "/api/shows", map[string]any{"tvmaze_id": 7}, one)
	expect(t, w, 201)
	if value(t, w, "id") != id || tv.calls.Load() != 2 {
		t.Fatal("metadata was fetched again for a second profile")
	}
	var episode string
	_ = s.DB.QueryRow("SELECT id FROM episodes WHERE number=1").Scan(&episode)
	expect(t, request(t, h, "PATCH", "/api/episodes/"+episode, map[string]bool{"watched": true}, zero), 200)
	w = request(t, h, "GET", "/api/calendar?from=2026-01-01&to=2026-02-01", nil, one)
	expect(t, w, 200)
	if strings.Contains(w.Body.String(), `"watched":1`) {
		t.Fatal("personal episode state leaked")
	}
	expect(t, request(t, h, "POST", "/api/shows/"+id+"/bulk", map[string]any{"season": 1, "downloaded": true}, one), 200)
	w = request(t, h, "GET", "/api/calendar?from=2026-01-01&to=2026-02-01", nil, zero)
	expect(t, w, 200)
	if !strings.Contains(w.Body.String(), `"downloaded":1`) {
		t.Fatal("global downloaded state was not visible to another profile")
	}
	expect(t, request(t, h, "DELETE", "/api/shows/"+id, nil, zero), 200)
	expect(t, request(t, h, "GET", "/api/shows/"+id, nil, one), 200)
	if tv.calls.Load() != 2 {
		t.Fatal("local navigation called provider")
	}
	var specials int
	_ = s.DB.QueryRow("SELECT COUNT(*) FROM episodes WHERE number=0").Scan(&specials)
	if specials != 1 {
		t.Fatal("specials missing")
	}
}
