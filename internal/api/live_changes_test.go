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
		name      string
		method    string
		path      string
		profile   string
		wantScope string
		want      []string
	}{
		{"episode state", http.MethodPatch, "/api/episodes/123", "user1", "user1", []string{"shows", "show", "calendar"}},
		{"show action", http.MethodPost, "/api/show-actions", "user1", "user1", []string{"show-actions"}},
		{"shared search settings", http.MethodPut, "/api/settings/search", "user0", "", []string{"editable-settings", "settings", "capabilities"}},
		{"job start", http.MethodPost, "/api/jobs/metadata", "user0", "", []string{"jobs", "schedules", "statistics"}},
		{"test connection", http.MethodPost, "/api/downloader/test", "user0", "", nil},
		{"read", http.MethodGet, "/api/calendar", "user1", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.path, nil)
			scope, got := liveChanges(r, auth.Session{Profile: tt.profile})
			if scope != tt.wantScope || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got scope=%q changes=%v, want scope=%q changes=%v", scope, got, tt.wantScope, tt.want)
			}
		})
	}
}
