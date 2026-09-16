package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/backup"
	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/jobs"
	"github.com/CarlFlo/tally/internal/metadata"
	"github.com/CarlFlo/tally/internal/settings"
	"github.com/CarlFlo/tally/internal/torrent"
	"github.com/CarlFlo/tally/web"
)

type browserTV struct{ fakeTV }

func (f *browserTV) SuggestShows(ctx context.Context) ([]metadata.SearchResult, error) {
	return f.SearchShows(ctx, "")
}

func (f *browserTV) GetShow(ctx context.Context, external string) (*metadata.Show, error) {
	if external == "7" {
		return f.fakeTV.GetShow(ctx, external)
	}
	n, _ := strconv.Atoi(external)
	return &metadata.Show{ID: n, Name: "Fixture show " + external, Status: "Running", Premiered: "2026-01-01"}, nil
}
func (f *browserTV) SearchShows(context.Context, string) ([]metadata.SearchResult, error) {
	return []metadata.SearchResult{{Score: 1, Show: metadata.Show{ID: 7, Name: "Example Show", Summary: "An original story for the browser test fixture.", Status: "Running", Premiered: "2026-01-01"}}}, nil
}
func (f *browserTV) GetEpisodes(_ context.Context, external string) ([]metadata.Episode, error) {
	episodes := []metadata.Episode{}
	offset := 0
	if external != "7" {
		n, _ := strconv.Atoi(external)
		offset = n * 1000
	}
	now := time.Now().UTC()
	for i := 0; i < 8; i++ {
		date := now.AddDate(0, 0, i-2)
		episodes = append(episodes, metadata.Episode{ID: offset + 100 + i, Season: 1, Number: i + 1, Name: "Chapter " + string(rune('A'+i)), Airdate: date.Format("2006-01-02"), Airstamp: date.Format("2006-01-02") + "T20:00:00Z", Runtime: 45, Summary: "A deterministic episode used only in automated browser tests."})
	}
	return episodes, nil
}

// Only started by Playwright. The fixture uses a temporary DB and never production data.
func TestBrowserServer(t *testing.T) {
	if os.Getenv("TALLY_BROWSER_TEST") != "1" {
		t.Skip("browser fixture is opt-in")
	}
	mode, port := "disabled", "18080"
	if os.Getenv("TALLY_BROWSER_AUTH") == "local" {
		mode, port = "local", "18082"
	}
	s, _, _ := testServer(t, mode)
	browserLocale := []byte(`{
		"_meta":{"locale":"sv","name":"Svenska","direction":"ltr","catalogVersion":1},
		"settings":{"myProfile":"Min profil"},
		"calendar":{"yourCalendar":"Din kalender"},
		"nav":{"calendar":"Kalender"},
		"common":{"save":"Spara"}
	}`)
	if err := os.WriteFile(filepath.Join(s.Config.DataDir, "locales", "sv.json"), browserLocale, 0o644); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for !s.Locales.Valid("sv") && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if !s.Locales.Valid("sv") {
		t.Fatal("browser fixture Swedish locale did not load")
	}
	// Browser tests deliberately avoid legacy userN IDs so UI authorization
	// cannot accidentally pass by coupling administrator access to an ID.
	if _, err := s.DB.Exec("DELETE FROM profiles WHERE id='profile-admin'"); err != nil {
		t.Fatal(err)
	}
	adminID := database.ID()
	if _, err := s.DB.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES(?,?,?,?)", adminID, "My profile", "violet", time.Now().Unix()); err != nil {
		t.Fatal(err)
	}
	if mode == "local" {
		hash, err := auth.Hash("1234")
		if err != nil {
			t.Fatal(err)
		}
		userID := database.ID()
		if _, err = s.DB.Exec("INSERT INTO local_credentials VALUES(?,?,0)", adminID, hash); err != nil {
			t.Fatal(err)
		}
		if _, err = s.DB.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES(?,?,?,?)", userID, "Alex", "mint", time.Now().Unix()); err != nil {
			t.Fatal(err)
		}
		if _, err = s.DB.Exec("INSERT INTO local_credentials VALUES(?,?,0)", userID, hash); err != nil {
			t.Fatal(err)
		}
	}
	s.Config.JobConcurrency = 2
	s.Config.JobRuntime = time.Minute
	s.Config.BatchSize = 20
	s.Config.RawRetention = 30
	s.Config.AggregateRetention = 365
	s.Metadata.Provider = &browserTV{}
	b := &backup.Service{DB: s.DB, DataDir: s.Config.DataDir, Path: filepath.Join(s.Config.DataDir, "backups"), Keep: 2}
	s.Backup = b
	s.Jobs = jobs.New(context.Background(), s.DB, s.Config, s.Metadata, s.Control, b)
	s.Jobs.OnChange = s.Events.Publish
	s.Control.Alert = s.Jobs.Alert
	if e := s.Jobs.Start(); e != nil {
		t.Fatal(e)
	}
	defer s.Jobs.Stop(context.Background())
	indexers := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("t") == "caps" {
			w.Write([]byte(`<caps><searching><search available="yes"/></searching></caps>`))
			return
		}
		fmt.Fprintf(w, `<rss xmlns:torznab="http://torznab.com/schemas/2015/feed"><channel><item><title>Example.Show.S01E01.1080p.WEB-DL.x265</title><guid>a</guid><jackettindexer>Fixture HD</jackettindexer><enclosure url="magnet:?xt=urn:btih:%s" length="1073741824"/><torznab:attr name="seeders" value="90"/></item><item><title>Example.Show.S01E01.720p.WEBRip.x264</title><guid>b</guid><jackettindexer>Fixture SD</jackettindexer><enclosure url="magnet:?xt=urn:btih:%s" length="1073741824"/><torznab:attr name="seeders" value="20"/></item></channel></rss>`, strings.Repeat("a", 40), strings.Repeat("b", 40))
	}))
	defer indexers.Close()
	configured := settings.Search{BaseURL: indexers.URL, APIKey: "fixture-key", Enabled: true}
	if _, e := (settings.Store{DB: s.DB}).Save(context.Background(), "search", configured, 1); e != nil {
		t.Fatal(e)
	}
	client := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+fixtureClientKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/api/v2/app/version" {
			w.Write([]byte("v5.2.0"))
			return
		}
		if r.URL.Path != "/api/v2/torrents/add" || r.Method != "POST" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte("Ok."))
	}))
	defer client.Close()
	// This opt-in fixture seeds only its temporary DB. Tests can edit and test the mock connection through the UI.
	if _, e := s.Clients.Save(context.Background(), torrent.ClientUpdate{Adapter: "qbittorrent", Fields: map[string]string{"url": client.URL, "api_key": fixtureClientKey}}); e != nil {
		t.Fatal(e)
	}
	s.Assets = web.Assets()
	server := http.Server{Addr: "127.0.0.1:" + port, Handler: notificationBrowserFixture(t, s.Handler()), ReadHeaderTimeout: 5 * time.Second}
	if e := server.ListenAndServe(); e != nil && e != http.ErrServerClosed {
		t.Fatal(e)
	}
}
