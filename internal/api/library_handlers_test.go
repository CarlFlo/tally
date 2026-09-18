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


func TestEpisodeStateAndSeasonBulkIgnoreUnreleasedEpisodes(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	if _, err := s.DB.Exec(`INSERT INTO shows(id,name) VALUES('release-guard','Release Guard');
	INSERT INTO episodes(id,show_id,season,number,name,airdate,airstamp) VALUES
		('released','release-guard',1,1,'Released','2000-01-01',''),
		('today-date-only','release-guard',1,2,'Today',date('now'),''),
		('future','release-guard',1,3,'Future','2099-01-01','');
	INSERT INTO profile_shows(profile_id,show_id,added_at) VALUES('profile-admin','release-guard',1);`); err != nil {
		t.Fatal(err)
	}

	expect(t, request(t, h, "PATCH", "/api/episodes/future", map[string]bool{"watched": true}), http.StatusBadRequest)
	expect(t, request(t, h, "PATCH", "/api/episodes/future", map[string]bool{"downloaded": true}), http.StatusBadRequest)
	expect(t, request(t, h, "PATCH", "/api/episodes/today-date-only", map[string]bool{"watched": true}), http.StatusBadRequest)
	// Clearing stale future state must remain possible.
	expect(t, request(t, h, "PATCH", "/api/episodes/future", map[string]bool{"watched": false}), http.StatusOK)
	expect(t, request(t, h, "PATCH", "/api/episodes/future", map[string]bool{"downloaded": false}), http.StatusOK)

	response := request(t, h, "POST", "/api/shows/release-guard/bulk", map[string]any{
		"season": 1, "watched": true, "downloaded": true,
	})
	expect(t, response, http.StatusOK)
	if !strings.Contains(response.Body.String(), `"updated":1`) {
		t.Fatalf("bulk update included unreleased episodes: %s", response.Body.String())
	}

	var releasedWatched, futureWatched, todayWatched, releasedDownloaded, futureDownloaded, todayDownloaded int
	if err := s.DB.QueryRow(`SELECT
		COALESCE((SELECT watched FROM profile_episode_state WHERE profile_id='profile-admin' AND episode_id='released'),0),
		COALESCE((SELECT watched FROM profile_episode_state WHERE profile_id='profile-admin' AND episode_id='future'),0),
		COALESCE((SELECT watched FROM profile_episode_state WHERE profile_id='profile-admin' AND episode_id='today-date-only'),0),
		(SELECT downloaded FROM episodes WHERE id='released'),
		(SELECT downloaded FROM episodes WHERE id='future'),
		(SELECT downloaded FROM episodes WHERE id='today-date-only')`).Scan(
		&releasedWatched, &futureWatched, &todayWatched,
		&releasedDownloaded, &futureDownloaded, &todayDownloaded,
	); err != nil {
		t.Fatal(err)
	}
	if releasedWatched != 1 || releasedDownloaded != 1 {
		t.Fatal("released episode was not marked by season bulk action")
	}
	if futureWatched != 0 || todayWatched != 0 || futureDownloaded != 0 || todayDownloaded != 0 {
		t.Fatalf("unreleased state changed: future watched=%d today watched=%d future downloaded=%d today downloaded=%d",
			futureWatched, todayWatched, futureDownloaded, todayDownloaded)
	}

	response = request(t, h, "POST", "/api/shows/release-guard/bulk", map[string]any{
		"season": 1, "watched": false, "downloaded": false,
	})
	expect(t, response, http.StatusOK)
	if err := s.DB.QueryRow(`SELECT
		COALESCE((SELECT watched FROM profile_episode_state WHERE profile_id='profile-admin' AND episode_id='released'),0),
		(SELECT downloaded FROM episodes WHERE id='released')`).Scan(&releasedWatched, &releasedDownloaded); err != nil {
		t.Fatal(err)
	}
	if releasedWatched != 0 || releasedDownloaded != 0 {
		t.Fatal("second season bulk action did not unmark released episodes")
	}
}
