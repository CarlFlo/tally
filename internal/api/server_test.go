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

	"github.com/CarlFlo/mediaManager/internal/auth"
	"github.com/CarlFlo/mediaManager/internal/backup"
	"github.com/CarlFlo/mediaManager/internal/config"
	"github.com/CarlFlo/mediaManager/internal/database"
	"github.com/CarlFlo/mediaManager/internal/live"
	"github.com/CarlFlo/mediaManager/internal/metadata"
	"github.com/CarlFlo/mediaManager/internal/providers"
	"github.com/CarlFlo/mediaManager/internal/torrent"
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
	c := config.Config{DataDir: dir, AuthMode: mode, MaxProfiles: 3, PasswordMin: 4, PasswordMax: 128, Timezone: "UTC", Theme: "system", SessionIdle: 30 * 24 * time.Hour, SessionAbsolute: 180 * 24 * time.Hour, ResetCooldown: time.Minute}
	p, e := providers.New(context.Background(), db, dir, 2, 0)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { p.Close() })
	tv := &fakeTV{}
	hub := live.New()
	b := &backup.Service{DB: db, DataDir: dir, Path: filepath.Join(dir, "backups"), Keep: 2}
	s := &Server{DB: db, Backup: b, Config: c, Auth: auth.New(db, c), Metadata: &metadata.Service{DB: db, Provider: tv}, Control: p, Events: hub}
	s.Metadata.OnChange = hub.Publish
	s.Clients = &torrent.ClientStore{DB: db, Control: p}
	return s, s.Handler(), tv
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
