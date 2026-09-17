package torrent

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/providers"
	"github.com/CarlFlo/tally/internal/settings"
)

type automationRequester struct {
	magnetOnly    bool
	searchCalls   *int
	torrentCalls  *int
	beforeTorrent func() error
}

func (r automationRequester) Do(_ context.Context, request providers.Request) (providers.Response, error) {
	u, err := url.Parse(request.URL)
	if err != nil {
		return providers.Response{}, err
	}
	if u.Path == "/download" {
		if r.torrentCalls != nil {
			*r.torrentCalls++
		}
		if r.beforeTorrent != nil {
			if err := r.beforeTorrent(); err != nil {
				return providers.Response{}, err
			}
		}
		body := []byte(automationTorrentFixture("Example.Show.S01E02.1080p.WEB-DL.mkv"))
		return providers.Response{Body: body, Status: 200}, nil
	}
	if !strings.HasSuffix(u.Path, jackettTorznabPath) {
		return providers.Response{}, fmt.Errorf("unexpected provider request %s", request.URL)
	}
	if r.searchCalls != nil {
		*r.searchCalls++
	}
	var enclosure string
	if r.magnetOnly {
		enclosure = `magnet:?xt=urn:btih:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`
	} else {
		enclosure = `http://jackett.test/download`
	}
	feed := fmt.Sprintf(`<rss xmlns:torznab="http://torznab.com/schemas/2015/feed"><channel><item><title>Example.Show.S01E02.1080p.WEB-DL.H264-GROUP</title><guid>one</guid><enclosure url="%s" length="2048"/><torznab:attr name="seeders" value="50"/></item></channel></rss>`, enclosure)
	return providers.Response{Body: []byte(feed), Status: 200}, nil
}

type automationClientSource struct{ client *automationClient }

func (s automationClientSource) Current(context.Context) (DownloadClient, error) {
	return s.client, nil
}

type automationClient struct {
	added          int
	data           []byte
	addErr         error
	reconcile      bool
	lastHash       string
	downloadsCalls int
}

func (c *automationClient) Name() string                             { return "test" }
func (c *automationClient) TestConnection(context.Context) error    { return nil }
func (c *automationClient) AddMagnet(context.Context, string) error { return fmt.Errorf("automation must not submit magnets") }
func (c *automationClient) AddTorrent(_ context.Context, data []byte) error {
	c.added++
	c.data = append([]byte(nil), data...)
	if metadata, err := ParseTorrentMetadata(data); err == nil {
		c.lastHash = metadata.InfoHashV1
	}
	return c.addErr
}
func (c *automationClient) Downloads(context.Context, string) (DownloadSnapshot, error) {
	c.downloadsCalls++
	if c.reconcile && c.lastHash != "" {
		return DownloadSnapshot{Torrents: []Download{{Hash: c.lastHash, Category: TallyCategory}}}, nil
	}
	return DownloadSnapshot{}, nil
}
func (c *automationClient) Stop(context.Context, string) error         { return nil }
func (c *automationClient) Start(context.Context, string) error        { return nil }
func (c *automationClient) Remove(context.Context, string, bool) error { return nil }

func automationTorrentFixture(name string) string {
	return fmt.Sprintf("d4:infod6:lengthi2048e4:name%d:%s12:piece lengthi16384e6:pieces20:aaaaaaaaaaaaaaaaaaaaee", len(name), name)
}

func automationTestStore(t *testing.T, now time.Time) *database.Store {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err = (settings.Store{DB: db}).Ensure(ctx); err != nil {
		t.Fatal(err)
	}
	setAutomationSettings(t, db, true, true, true)
	airstamp := now.Add(-time.Hour).UTC().Format(time.RFC3339)
	if _, err = db.ExecContext(ctx, `INSERT INTO profiles(id,display_name,avatar,created_at,locale,auth_method) VALUES('profile-a','Alex','mint',1,'en','none');
INSERT INTO shows(id,name,premiered) VALUES('show-a','Example Show','2026-01-01');
INSERT INTO episodes(id,show_id,season,number,name,airstamp) VALUES('episode-a','show-a',1,2,'Second',?);
INSERT INTO profile_shows(profile_id,show_id,added_at) VALUES('profile-a','show-a',1);`, airstamp); err != nil {
		t.Fatal(err)
	}
	return db
}

func setAutomationSettings(t *testing.T, db *database.Store, automationEnabled, searchEnabled, downloadsEnabled bool) {
	t.Helper()
	search := fmt.Sprintf(`{"base_url":"http://jackett.test","api_key":"key","enabled":%t}`, searchEnabled)
	downloads := fmt.Sprintf(`{"enabled":%t}`, downloadsEnabled)
	automation := fmt.Sprintf(`{"enabled":%t,"preferred_quality":"1080p","min_seeders":5,"high_confidence_only":true,"prefer_smaller":false,"release_delay_minutes":20,"retry_window_hours":24,"max_candidates":5}`, automationEnabled)
	if _, err := db.ExecContext(context.Background(), `UPDATE application_settings SET data=? WHERE key='search';
UPDATE application_settings SET data=? WHERE key='torrent';
UPDATE application_settings SET data=? WHERE key='torrent_automation';`, search, downloads, automation); err != nil {
		t.Fatal(err)
	}
}

func automationService(db *database.Store, requester automationRequester, client *automationClient, now time.Time) *AutomationService {
	return &AutomationService{
		DB:      db,
		Control: requester,
		Clients: automationClientSource{client: client},
		Now:     func() time.Time { return now },
	}
}

func TestAutomationDownloadsOnlyAfterVerifiedTorrentInspection(t *testing.T) {
	now := time.Date(2026, 9, 17, 19, 0, 0, 0, time.UTC)
	db := automationTestStore(t, now)
	client := &automationClient{}
	processed, err := automationService(db, automationRequester{}, client, now).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 || client.added != 1 {
		t.Fatalf("expected one verified automated submission, processed=%d added=%d", processed, client.added)
	}
	if len(client.data) == 0 {
		t.Fatal("torrent payload was not submitted")
	}
	runs, err := (AutomationStore{DB: db}).ListRuns(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].Status != RunDownloaded || runs[0].Confidence != ConfidenceHigh || runs[0].Verification != VerificationVerified {
		t.Fatalf("unexpected automation history: %+v", runs)
	}
	if runs[0].SelectedInfoHash == "" {
		t.Fatal("verified run did not persist the derived infohash")
	}
	if len(runs[0].DecisionLog) < 4 {
		t.Fatalf("decision flow was not preserved: %+v", runs[0].DecisionLog)
	}
}

func TestAutomationNeverDownloadsMagnetOnlyCandidate(t *testing.T) {
	now := time.Date(2026, 9, 17, 19, 0, 0, 0, time.UTC)
	db := automationTestStore(t, now)
	client := &automationClient{}
	processed, err := automationService(db, automationRequester{magnetOnly: true}, client, now).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 || client.added != 0 {
		t.Fatalf("magnet-only candidate reached automatic download, processed=%d added=%d", processed, client.added)
	}
	runs, err := (AutomationStore{DB: db}).ListRuns(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].Status != RunNoVerifiedCandidate {
		t.Fatalf("unexpected magnet-only outcome: %+v", runs)
	}
	foundMagnetReason := false
	for _, step := range runs[0].DecisionLog {
		if step.Stage == "filter" {
			if count, ok := step.Data["magnet_only"].(float64); ok && count == 1 {
				foundMagnetReason = true
			}
		}
	}
	if !foundMagnetReason {
		t.Fatalf("magnet exclusion was not recorded in decision flow: %+v", runs[0].DecisionLog)
	}
}

func TestAutomationNoOpsWhenRequiredCapabilityIsDisabled(t *testing.T) {
	now := time.Date(2026, 9, 17, 19, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name                         string
		automation, search, downloads bool
	}{
		{name: "automation", automation: false, search: true, downloads: true},
		{name: "search", automation: true, search: false, downloads: true},
		{name: "downloads", automation: true, search: true, downloads: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := automationTestStore(t, now)
			setAutomationSettings(t, db, test.automation, test.search, test.downloads)
			searchCalls := 0
			client := &automationClient{}
			processed, err := automationService(db, automationRequester{searchCalls: &searchCalls}, client, now).Run(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if processed != 0 || searchCalls != 0 || client.added != 0 {
				t.Fatalf("disabled %s capability performed work: processed=%d searches=%d added=%d", test.name, processed, searchCalls, client.added)
			}
		})
	}
}

func TestAutomationNeverPolicySuppressesGlobalAutomation(t *testing.T) {
	now := time.Date(2026, 9, 17, 19, 0, 0, 0, time.UTC)
	db := automationTestStore(t, now)
	if err := (AutomationStore{DB: db}).SetShowPolicy(context.Background(), "show-a", "never"); err != nil {
		t.Fatal(err)
	}
	searchCalls := 0
	client := &automationClient{}
	processed, err := automationService(db, automationRequester{searchCalls: &searchCalls}, client, now).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if processed != 0 || searchCalls != 0 || client.added != 0 {
		t.Fatalf("never policy did not suppress automation: processed=%d searches=%d added=%d", processed, searchCalls, client.added)
	}
}

func TestAutomationRetryBackoffGatesRepeatedNoCandidateRuns(t *testing.T) {
	now := time.Date(2026, 9, 17, 19, 0, 0, 0, time.UTC)
	db := automationTestStore(t, now)
	store := AutomationStore{DB: db}
	runID, err := store.StartRun(context.Background(), AutomationRun{
		ShowID: "show-a", EpisodeID: "episode-a", ShowName: "Example Show", Season: 1, Episode: 2,
		Query: "Example Show S01E02", StartedAt: now.Add(-10 * time.Minute).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = store.FinishRun(context.Background(), runID, RunNoVerifiedCandidate, ReleaseAssessment{Verification: VerificationUnverified}, ""); err != nil {
		t.Fatal(err)
	}
	searchCalls := 0
	client := &automationClient{}
	service := automationService(db, automationRequester{searchCalls: &searchCalls}, client, now)
	processed, err := service.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if processed != 0 || searchCalls != 0 {
		t.Fatalf("retry ran before 30 minute backoff: processed=%d searches=%d", processed, searchCalls)
	}
	if _, err = db.Exec("UPDATE torrent_automation_runs SET started_at=? WHERE id=?", now.Add(-31*time.Minute).Unix(), runID); err != nil {
		t.Fatal(err)
	}
	processed, err = service.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 || searchCalls != 1 || client.added != 1 {
		t.Fatalf("retry did not resume after backoff: processed=%d searches=%d added=%d", processed, searchCalls, client.added)
	}
}

func TestAutomationDownloadedEpisodeIsNotSubmittedTwice(t *testing.T) {
	now := time.Date(2026, 9, 17, 19, 0, 0, 0, time.UTC)
	db := automationTestStore(t, now)
	client := &automationClient{}
	service := automationService(db, automationRequester{}, client, now)
	if processed, err := service.Run(context.Background()); err != nil || processed != 1 {
		t.Fatalf("first run failed: processed=%d err=%v", processed, err)
	}
	if processed, err := service.Run(context.Background()); err != nil || processed != 0 {
		t.Fatalf("downloaded episode was reconsidered: processed=%d err=%v", processed, err)
	}
	if client.added != 1 {
		t.Fatalf("duplicate torrent submission occurred: %d", client.added)
	}
}

func TestAutomationRechecksCapabilityBeforeSubmission(t *testing.T) {
	now := time.Date(2026, 9, 17, 19, 0, 0, 0, time.UTC)
	db := automationTestStore(t, now)
	client := &automationClient{}
	requester := automationRequester{beforeTorrent: func() error {
		_, err := db.Exec(`UPDATE application_settings SET data='{"enabled":false}' WHERE key='torrent'`)
		return err
	}}
	processed, err := automationService(db, requester, client, now).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 || client.added != 0 {
		t.Fatalf("disabled downloader reached submission: processed=%d added=%d", processed, client.added)
	}
	runs, err := (AutomationStore{DB: db}).ListRuns(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].Status != RunSkipped {
		t.Fatalf("capability shutdown was not recorded as skipped: %+v", runs)
	}
}

func TestAutomationReconcilesAmbiguousClientFailureByInfoHash(t *testing.T) {
	now := time.Date(2026, 9, 17, 19, 0, 0, 0, time.UTC)
	db := automationTestStore(t, now)
	client := &automationClient{addErr: errors.New("ambiguous client response"), reconcile: true}
	processed, err := automationService(db, automationRequester{}, client, now).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 || client.added != 1 || client.downloadsCalls != 1 {
		t.Fatalf("client failure was not reconciled: processed=%d added=%d downloads=%d", processed, client.added, client.downloadsCalls)
	}
	runs, err := (AutomationStore{DB: db}).ListRuns(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].Status != RunDownloaded || runs[0].SelectedInfoHash == "" {
		t.Fatalf("reconciled submission was not persisted as downloaded: %+v", runs)
	}
}
