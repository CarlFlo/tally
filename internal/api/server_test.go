package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/backup"
	"github.com/CarlFlo/tally/internal/config"
	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/live"
	"github.com/CarlFlo/tally/internal/localization"
	"github.com/CarlFlo/tally/internal/metadata"
	"github.com/CarlFlo/tally/internal/providers"
	"github.com/CarlFlo/tally/internal/torrent"
)

type fakeTV struct{ calls atomic.Int32 }

func (f *fakeTV) SearchShows(context.Context, string) ([]metadata.SearchResult, error) {
	f.calls.Add(1)
	return []metadata.SearchResult{}, nil
}

func (f *fakeTV) GetShow(context.Context, string) (*metadata.Show, error) {
	f.calls.Add(1)
	return &metadata.Show{ID: 7, Name: "Example Show", Status: "Running", Updated: 1}, nil
}

func (f *fakeTV) GetEpisodes(context.Context, string) ([]metadata.Episode, error) {
	f.calls.Add(1)
	return []metadata.Episode{{ID: 10, Season: 1, Number: 1, Name: "Pilot", Airdate: "2026-01-02", Airstamp: "2026-01-02T20:00:00Z"}, {ID: 11, Season: 1, Number: 0, Name: "Special", Type: "significant_special", Airdate: "2026-01-03"}}, nil
}

func testServer(t *testing.T, mode string) (*Server, http.Handler, *fakeTV) {
	t.Helper()
	dir := t.TempDir()
	db, e := database.Open(context.Background(), dir)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	if _, e = db.Exec("INSERT INTO profiles(id,display_name,avatar,created_at,auth_method) VALUES('profile-admin','My profile','violet',?,'none')", time.Now().Unix()); e != nil {
		t.Fatal(e)
	}
	c := config.Config{DataDir: dir, MaxProfiles: 3, PasswordMin: 4, PasswordMax: 128, Timezone: "UTC", Theme: "system", SessionIdle: 30 * 24 * time.Hour, SessionAbsolute: 180 * 24 * time.Hour, ResetCooldown: time.Minute}
	p, e := providers.New(context.Background(), db, dir, 2, 0)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { p.Close() })
	tv := &fakeTV{}
	hub := live.New()
	locales, e := localization.New(dir, func() { hub.Publish("", "locales") })
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { locales.Close() })
	b := &backup.Service{DB: db, DataDir: dir, Path: filepath.Join(dir, "backups"), Keep: 2}
	s := &Server{DB: db, Backup: b, Config: c, Auth: auth.New(db, c), Metadata: &metadata.Service{DB: db, Provider: tv}, Control: p, Events: hub, Locales: locales}
	s.Metadata.OnChange = hub.Publish
	s.Clients = &torrent.ClientStore{DB: db, Control: p}
	defaultProfile := ""
	if mode == "disabled" {
		defaultProfile = "profile-admin"
	}
	return s, testProfileCookies(s, s.Handler(), defaultProfile), tv
}

func testProfileCookies(s *Server, next http.Handler, defaultProfile string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := r.Cookie("tally_session"); err != nil {
			profile := defaultProfile
			if legacy, legacyErr := r.Cookie("tally_profile"); legacyErr == nil && legacy.Value != "" {
				profile = legacy.Value
			}
			if profile != "" {
				token := auth.Token()
				now := time.Now()
				result, execErr := s.DB.ExecContext(r.Context(), `INSERT INTO sessions(id,profile_id,created_at,last_seen,expires_at,restricted,user_agent)
					SELECT ?,id,?,?,?,?,? FROM profiles WHERE id=?`, auth.Digest(token), now.Unix(), now.Unix(), now.Add(s.Config.SessionAbsolute).Unix(), false, "test fixture", profile)
				if execErr == nil {
					if affected, rowsErr := result.RowsAffected(); rowsErr == nil && affected == 1 {
						r.AddCookie(&http.Cookie{Name: "tally_session", Value: token})
					}
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func request(t *testing.T, h http.Handler, method, path string, body any, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var data []byte
	if body != nil {
		data, _ = json.Marshal(body)
	}
	r := httptest.NewRequest(method, path, bytes.NewReader(data))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Tally-CSRF", "1")
	for _, c := range cookies {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func expect(t *testing.T, w *httptest.ResponseRecorder, status int) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status %d, want %d: %s", w.Code, status, w.Body.String())
	}
}

func value(t *testing.T, w *httptest.ResponseRecorder, key string) string {
	t.Helper()
	var v map[string]any
	if e := json.Unmarshal(w.Body.Bytes(), &v); e != nil {
		t.Fatal(e)
	}
	return v[key].(string)
}
