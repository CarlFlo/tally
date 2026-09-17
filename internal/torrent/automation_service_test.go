package torrent

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/providers"
)

type automationRequester struct {
	magnetOnly bool
}

func (r automationRequester) Do(_ context.Context, request providers.Request) (providers.Response, error) {
	u, err := url.Parse(request.URL)
	if err != nil {
		return providers.Response{}, err
	}
	if u.Path == "/download" {
		body := []byte(automationTorrentFixture("Example.Show.S01E02.1080p.WEB-DL.mkv"))
		return providers.Response{Body: body, Status: 200}, nil
	}
	if !strings.HasSuffix(u.Path, jackettTorznabPath) {
		return providers.Response{}, fmt.Errorf("unexpected provider request %s", request.URL)
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
	added int
	data  []byte
}

func (c *automationClient) Name() string                              { return "test" }
func (c *automationClient) TestConnection(context.Context) error     { return nil }
func (c *automationClient) AddMagnet(context.Context, string) error  { return fmt.Errorf("automation must not submit magnets") }
func (c *automationClient) AddTorrent(_ context.Context, data []byte) error {
	c.added++
	c.data = append([]byte(nil), data...)
	return nil
}
func (c *automationClient) Downloads(context.Context, string) (DownloadSnapshot, error) {
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
	search := `{"base_url":"http://jackett.test","api_key":"key","enabled":true}`
	downloads := `{"enabled":true}`
	automation := `{"enabled":true,"preferred_quality":"1080p","min_seeders":5,"high_confidence_only":true,"prefer_smaller":false,"release_delay_minutes":20,"retry_window_hours":24,"max_candidates":5}`
	if _, err = db.ExecContext(ctx, `UPDATE application_settings SET data=? WHERE key='search';
UPDATE application_settings SET data=? WHERE key='torrent';
UPDATE application_settings SET data=? WHERE key='torrent_automation';`, search, downloads, automation); err != nil {
		t.Fatal(err)
	}
	airstamp := now.Add(-time.Hour).UTC().Format(time.RFC3339)
	if _, err = db.ExecContext(ctx, `INSERT INTO profiles(id,display_name,avatar,created_at,locale,auth_method) VALUES('profile-a','Alex','mint',1,'en','none');
INSERT INTO shows(id,name,premiered) VALUES('show-a','Example Show','2026-01-01');
INSERT INTO episodes(id,show_id,season,number,name,airstamp) VALUES('episode-a','show-a',1,2,'Second',?);
INSERT INTO profile_shows(profile_id,show_id,added_at) VALUES('profile-a','show-a',1);`, airstamp); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestAutomationDownloadsOnlyAfterVerifiedTorrentInspection(t *testing.T) {
	now := time.Date(2026, 9, 17, 19, 0, 0, 0, time.UTC)
	db := automationTestStore(t, now)
	client := &automationClient{}
	service := &AutomationService{
		DB: db,
		Control: automationRequester{},
		Clients: automationClientSource{client: client},
		Now: func() time.Time { return now },
	}
	processed, err := service.Run(context.Background())
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
	service := &AutomationService{
		DB: db,
		Control: automationRequester{magnetOnly: true},
		Clients: automationClientSource{client: client},
		Now: func() time.Time { return now },
	}
	processed, err := service.Run(context.Background())
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
