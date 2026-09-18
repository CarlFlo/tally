package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/CarlFlo/tally/internal/library"
)

func TestClearWatchHistoryPreservesDownloadsAndOtherProfiles(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	original := h
	h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := r.Cookie("tally_profile"); err != nil {
			r.AddCookie(&http.Cookie{Name: "tally_profile", Value: "profile-admin"})
		}
		original.ServeHTTP(w, r)
	})
	added := request(t, h, "POST", "/api/shows", map[string]int{"tvmaze_id": 7})
	expect(t, added, 201)
	id := value(t, added, "id")
	expect(t, request(t, h, "POST", "/api/shows/"+id+"/bulk", map[string]any{"watched": true, "downloaded": true}), 200)
	if _, err := s.DB.Exec(`INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('profile-member','Alex','mint',1); INSERT INTO profile_episode_state SELECT 'profile-member',id,1,0,1 FROM episodes`); err != nil {
		t.Fatal(err)
	}
	expect(t, request(t, h, "DELETE", "/api/shows/"+id+"/watch-history", nil, &http.Cookie{Name: "tally_profile", Value: "profile-member"}), 404)
	expect(t, request(t, h, "DELETE", "/api/shows/"+id+"/watch-history", nil), 200)
	var watched, downloaded, other int
	if err := s.DB.QueryRow("SELECT SUM(watched) FROM profile_episode_state WHERE profile_id='profile-admin'").Scan(&watched); err != nil {
		t.Fatal(err)
	}
	if err := s.DB.QueryRow("SELECT SUM(downloaded) FROM episodes WHERE show_id=?", id).Scan(&downloaded); err != nil {
		t.Fatal(err)
	}
	_ = s.DB.QueryRow("SELECT SUM(watched) FROM profile_episode_state WHERE profile_id='profile-member'").Scan(&other)
	if watched != 0 || downloaded != 2 || other != 2 {
		t.Fatal("state isolation failed", watched, downloaded, other)
	}
	expect(t, request(t, h, "DELETE", "/api/shows/"+id+"/watch-history", nil), 200)
	logs := request(t, h, "GET", "/api/logs?action=watch_history_cleared&q=example", nil)
	expect(t, logs, 200)
	if !strings.Contains(logs.Body.String(), `"total":1`) {
		t.Fatal(logs.Body.String())
	}
	expect(t, request(t, h, "DELETE", "/api/shows/"+id, nil), 200)
	// Re-following remains idempotent and emits exactly one additional event.
	for i := 0; i < 2; i++ {
		tx, err := s.DB.BeginTx(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		if err = library.SetFollow(context.Background(), tx, "profile-admin", id, true); err != nil {
			t.Fatal(err)
		}
		if err = tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
	logs = request(t, h, "GET", "/api/logs?action=show_added", nil)
	if !strings.Contains(logs.Body.String(), `"total":2`) {
		t.Fatal(logs.Body.String())
	}
	if _, err := s.DB.Exec("INSERT INTO job_runs(id,job_key,trigger,started_at,status) VALUES('named',?,'manual',1,'success')", "metadata:tvmaze:show:"+id); err != nil {
		t.Fatal(err)
	}
	result := request(t, h, "GET", "/api/jobs?kind=metadata&status=success", nil)
	expect(t, result, 200)
	if !strings.Contains(result.Body.String(), `"show_name":"Example Show"`) {
		t.Fatal(fmt.Sprint(result.Body))
	}
}
