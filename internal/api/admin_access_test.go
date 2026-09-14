package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdministrationIsUserZeroOnlyInEveryAuthMode(t *testing.T) {
	for _, mode := range []string{"disabled", "local"} {
		t.Run(mode, func(t *testing.T) {
			s, h, _ := testServer(t, mode)
			if _, err := s.DB.Exec("INSERT INTO profiles VALUES('user1','Alex','mint',1); INSERT INTO profiles VALUES('user2','Sam','mint',2)"); err != nil {
				t.Fatal(err)
			}
			cookies := []*http.Cookie{{Name: "tally_profile", Value: "user1"}}
			if mode == "local" {
				rec := httptest.NewRecorder()
				if err := s.Auth.NewSession(context.Background(), rec, httptest.NewRequest("GET", "/", nil), "user1", false); err != nil {
					t.Fatal(err)
				}
				cookies = rec.Result().Cookies()
			}
			for _, path := range []string{"/api/settings", "/api/settings/notifications", "/api/settings/search", "/api/jobs", "/api/statistics", "/api/backups", "/api/settings/backups", "/api/backups/example/download", "/api/alerts", "/api/downloader"} {
				expect(t, request(t, h, "GET", path, nil, cookies...), 403)
			}
			expect(t, request(t, h, "POST", "/api/settings/notifications/test", map[string]any{}, cookies...), 403)
			expect(t, request(t, h, "GET", "/api/capabilities", nil, cookies...), 200)
			expect(t, request(t, h, "DELETE", "/api/profiles/user2", nil, cookies...), 403)
			expect(t, request(t, h, "DELETE", "/api/profiles/user0", nil, cookies...), 400)
			deleted := request(t, h, "DELETE", "/api/profiles/user1", nil, cookies...)
			expect(t, deleted, 200)
			var remaining int
			_ = s.DB.QueryRow("SELECT COUNT(*) FROM profiles WHERE id='user1'").Scan(&remaining)
			if remaining != 0 {
				t.Fatal("profile still exists")
			}
			var message string
			if err := s.DB.QueryRow("SELECT message FROM activity_log WHERE action='profile_deleted'").Scan(&message); err != nil || !strings.Contains(message, "Alex") {
				t.Fatal("deletion activity lost", err)
			}
			expect(t, request(t, h, "GET", "/api/shows", nil, cookies...), 401)
		})
	}
}
