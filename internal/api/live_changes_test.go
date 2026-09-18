package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/auth"
)

func TestLiveChangesScopesAndTargetsResources(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		path    string
		profile string
		want    []liveUpdate
	}{
		{
			"episode state",
			http.MethodPatch,
			"/api/episodes/123",
			"profile-member",
			[]liveUpdate{update("profile-member", "shows", "show", "calendar")},
		},
		{
			"show action",
			http.MethodPost,
			"/api/show-actions",
			"profile-member",
			[]liveUpdate{update("profile-member", "show-actions")},
		},
		{
			"profile activity also refreshes admin logs",
			http.MethodDelete,
			"/api/shows/abc/watch-history",
			"profile-member",
			[]liveUpdate{
				update("profile-member", "shows", "show", "calendar", "logs"),
				update("", "logs"),
			},
		},
		{
			"torrent activity also refreshes admin statistics",
			http.MethodPost,
			"/api/torrents/search",
			"profile-member",
			[]liveUpdate{
				update("profile-member", "torrent-history"),
				update("", "statistics"),
			},
		},
		{
			"torrent send refreshes history and shared downloads",
			http.MethodPost,
			"/api/torrents/send",
			"profile-member",
			[]liveUpdate{
				update("profile-member", "torrent-history"),
				update("", "downloads"),
				update("", "statistics"),
			},
		},
		{
			"shared search settings",
			http.MethodPut,
			"/api/settings/search",
			"profile-admin",
			[]liveUpdate{update("", "editable-settings", "settings", "capabilities", "bootstrap")},
		},
		{
			"shared torrent settings",
			http.MethodPut,
			"/api/settings/torrent",
			"profile-admin",
			[]liveUpdate{update("", "editable-settings", "settings", "capabilities", "bootstrap", "downloads")},
		},
		{
			"password sign-in refreshes sessions globally",
			http.MethodPost,
			"/api/auth/login",
			"",
			[]liveUpdate{update("", "sessions")},
		},
		{
			"passwordless sign-in refreshes sessions globally",
			http.MethodPost,
			"/api/profiles/select",
			"",
			[]liveUpdate{update("", "sessions")},
		},
		{
			"logout refreshes sessions for the profile",
			http.MethodPost,
			"/api/auth/logout",
			"profile-member",
			[]liveUpdate{update("profile-member", "sessions")},
		},
		{
			"torrent pause keeps optimistic download state",
			http.MethodPost,
			"/api/torrents/downloads/abc/stop",
			"profile-member",
			[]liveUpdate{update("", "statistics")},
		},
		{
			"torrent removal keeps optimistic download state",
			http.MethodDelete,
			"/api/torrents/downloads/abc",
			"profile-member",
			[]liveUpdate{update("", "statistics")},
		},
		{
			"job start",
			http.MethodPost,
			"/api/jobs/metadata",
			"profile-admin",
			[]liveUpdate{update("", "jobs", "schedules", "statistics")},
		},
		{"test connection", http.MethodPost, "/api/downloader/test", "profile-admin", nil},
		{"read", http.MethodGet, "/api/calendar", "profile-member", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.path, nil)
			got := liveChanges(r, auth.Session{Profile: tt.profile})
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got updates=%#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestDownloadedEpisodeMutationPublishesGlobalLibraryInvalidation(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	if _, err := s.DB.Exec(`INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('profile-member','Member','mint',2);
	INSERT INTO shows(id,name) VALUES('shared-show','Shared Show');
	INSERT INTO episodes(id,show_id,season,number,name) VALUES('shared-episode','shared-show',1,1,'Pilot');
	INSERT INTO profile_shows(profile_id,show_id,added_at) VALUES
		('profile-admin','shared-show',1),
		('profile-member','shared-show',1);`); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updates := s.Events.Subscribe(ctx, "profile-member")

	response := request(t, h, "PATCH", "/api/episodes/shared-episode", map[string]any{"downloaded": true})
	expect(t, response, http.StatusOK)

	select {
	case event := <-updates:
		found := map[string]bool{}
		for _, change := range event.Changes {
			found[change.Resource] = true
		}
		if !found["shows"] || !found["show"] || !found["calendar"] {
			t.Fatalf("global downloaded change published incomplete resources: %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("other profile did not receive global downloaded invalidation")
	}
}
