package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/torrent"
)

func TestTorrentAutomationRunAccessIsScopedToFollowedShows(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	ctx := context.Background()

	if _, err := s.DB.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('profile-member','Member','mint',?)", time.Now().Unix()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec("INSERT INTO shows(id,name) VALUES('show-a','Private Show')"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec("INSERT INTO episodes(id,show_id,season,number,name) VALUES('episode-a','show-a',1,1,'Pilot')"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec("INSERT INTO profile_shows(profile_id,show_id,added_at) VALUES('profile-admin','show-a',?)", time.Now().Unix()); err != nil {
		t.Fatal(err)
	}
	store := torrent.AutomationStore{DB: s.DB}
	runID, err := store.StartRun(ctx, torrent.AutomationRun{
		ShowID: "show-a", EpisodeID: "episode-a", ShowName: "Private Show",
		Season: 1, Episode: 1, Query: "Private Show S01E01",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = store.FinishRun(ctx, runID, torrent.RunSkipped, torrent.ReleaseAssessment{}, ""); err != nil {
		t.Fatal(err)
	}

	member := &http.Cookie{Name: "tally_profile", Value: "profile-member"}
	list := request(t, h, "GET", "/api/torrents/automation/runs", nil, member)
	expect(t, list, http.StatusOK)
	if strings.Contains(list.Body.String(), runID) {
		t.Fatal("automation run leaked to a profile that does not follow the show")
	}
	expect(t, request(t, h, "GET", "/api/torrents/automation/runs/"+runID, nil, member), http.StatusNotFound)
	expect(t, request(t, h, "POST", "/api/torrents/automation/runs/"+runID+"/bad", map[string]string{"reason": "other"}, member), http.StatusNotFound)

	if _, err = s.DB.Exec("INSERT INTO profile_shows(profile_id,show_id,added_at) VALUES('profile-member','show-a',?)", time.Now().Unix()); err != nil {
		t.Fatal(err)
	}
	list = request(t, h, "GET", "/api/torrents/automation/runs", nil, member)
	expect(t, list, http.StatusOK)
	if !strings.Contains(list.Body.String(), runID) {
		t.Fatal("followed show automation run was not visible")
	}
	expect(t, request(t, h, "GET", "/api/torrents/automation/runs/"+runID, nil, member), http.StatusOK)
}
