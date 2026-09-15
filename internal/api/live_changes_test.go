package api

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/CarlFlo/mediaManager/internal/auth"
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
			"shared search settings",
			http.MethodPut,
			"/api/settings/search",
			"profile-admin",
			[]liveUpdate{update("", "editable-settings", "settings", "capabilities")},
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
