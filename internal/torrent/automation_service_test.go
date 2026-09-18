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
	withMagnet    bool
	magnetHash    string
	reportedSize  int64
	published     string
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
			(*r.torrentCalls)++
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
		(*r.searchCalls)++
	}
	var enclosure, itemLink string
	if r.magnetOnly || r.withMagnet {
		hash := r.magnetHash
		if hash == "" {
			hash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			if r.withMagnet {
				if metadata, parseErr := ParseTorrentMetadata([]byte(automationTorrentFixture("Example.Show.S01E02.1080p.WEB-DL.mkv"))); parseErr == nil {
					hash = metadata.InfoHashV1
				}
			}
		}
		enclosure = "magnet:?xt=urn:btih:" + hash
	} else {
		enclosure = `http://jackett.test/download`
	}
	if r.withMagnet {
		itemLink = "<link>http://jackett.test/download</link>"
	}
	size := r.reportedSize
	if size == 0 {
		size = 2048
	}
	pubDate := ""
	if r.published != "" {
		pubDate = "<pubDate>" + r.published + "</pubDate>"
	}
	feed := fmt.Sprintf(`<rss xmlns:torznab="http://torznab.com/schemas/2015/feed"><channel><item><title>Example.Show.S01E02.1080p.WEB-DL.H264-GROUP</title><guid>one</guid>%s%s<enclosure url="%s" length="%d"/><torznab:attr name="seeders" value="50"/></item></channel></rss>`, itemLink, pubDate, enclosure, size)
	return providers.Response{Body: []byte(feed), Status: 200}, nil
}

type automationClientSource struct{ client *automationClient }

func (s automationClientSource) Current(context.Context) (DownloadClient, error) {
	return s.client, nil
}

type automationClient struct {
	added          int
	magnetAdded    int
	magnet         string
	data           []byte
	addErr         error
	reconcile      bool
	listed         bool
	lastHash       string
	downloadsCalls int
	resolvedFiles  []TorrentFile
	resolvedErr    error
	resolvedCalls  int
	stopCalls      int
	removeCalls    int
	deleteFiles    bool
}

func (c *automationClient) Name() string                          { return "test" }
func (c *automationClient) TestConnection(context.Context) error { return nil }
func (c *automationClient) AddMagnet(_ context.Context, magnet string) error {
	c.magnetAdded++
	c.magnet = magnet
	c.lastHash = MagnetInfoHash(magnet)
	return c.addErr
}
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
	if (c.reconcile || c.listed) && c.lastHash != "" {
		return DownloadSnapshot{Torrents: []Download{{Hash: c.lastHash, Category: TallyCategory}}}, nil
	}
	return DownloadSnapshot{}, nil
}
func (c *automationClient) ResolvedFiles(context.Context, string) ([]TorrentFile, error) {
	c.resolvedCalls++
	if c.resolvedErr != nil {
		return nil, c.resolvedErr
	}
	return append([]TorrentFile(nil), c.resolvedFiles...), nil
}
func (c *automationClient) Stop(context.Context, string) error {
	c.stopCalls++
	return nil
}
func (c *automationClient) Start(context.Context, string) error { return nil }
func (c *automationClient) Remove(_ context.Context, _ string, deleteFiles bool) error {
	c.removeCalls++
	c.deleteFiles = deleteFiles
	return nil
}

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
	ctx := context.Background()
	store := settings.Store{DB: db}

	var search settings.Search
	searchRevision, err := store.Load(ctx, "search", &search)
	if err != nil {
		t.Fatal(err)
	}
	search = settings.Search{BaseURL: "http://jackett.test", APIKey: "key", Enabled: searchEnabled}
	if _, err = store.Save(ctx, "search", search, searchRevision); err != nil {
		t.Fatal(err)
	}

	var downloads settings.Torrent
	downloadRevision, err := store.Load(ctx, "torrent", &downloads)
	if err != nil {
		t.Fatal(err)
	}
	downloads.Enabled = downloadsEnabled
	if _, err = store.Save(ctx, "torrent", downloads, downloadRevision); err != nil {
		t.Fatal(err)
	}

	var automation settings.TorrentAutomation
	automationRevision, err := store.Load(ctx, "torrent_automation", &automation)
	if err != nil {
		t.Fatal(err)
	}
	automation = settings.DefaultTorrentAutomation()
	automation.Enabled = automationEnabled
	if _, err = store.Save(ctx, "torrent_automation", automation, automationRevision); err != nil {
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

func TestAutomationAllowsHighConfidenceMagnetFallback(t *testing.T) {
	now := time.Date(2026, 9, 17, 19, 0, 0, 0, time.UTC)
	db := automationTestStore(t, now)
	client := &automationClient{}
	processed, err := automationService(db, automationRequester{magnetOnly: true}, client, now).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 || client.added != 0 || client.magnetAdded != 1 {
		t.Fatalf("expected one magnet fallback, processed=%d torrents=%d magnets=%d", processed, client.added, client.magnetAdded)
	}
	runs, err := (AutomationStore{DB: db}).ListRuns(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].Status != RunDownloaded || runs[0].Confidence != ConfidenceHigh || runs[0].Verification != VerificationUnverified {
		t.Fatalf("unexpected magnet fallback outcome: %+v", runs)
	}
	if runs[0].SelectedInfoHash != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("magnet infohash was not preserved: %+v", runs[0])
	}
	foundMetadataOnly := false
	for _, step := range runs[0].DecisionLog {
		if step.Stage == "inspection" && step.Status == "metadata_only" {
			foundMetadataOnly = true
		}
	}
	if !foundMetadataOnly {
		t.Fatalf("metadata-only fallback was not explicit: %+v", runs[0].DecisionLog)
	}
}


func TestAutomationMagnetVerificationStaysPendingUntilFilesResolve(t *testing.T) {
	now := time.Date(2026, 9, 17, 19, 0, 0, 0, time.UTC)
	db := automationTestStore(t, now)
	client := &automationClient{listed: true}
	service := automationService(db, automationRequester{magnetOnly: true}, client, now)
	if processed, err := service.Run(context.Background()); err != nil || processed != 1 {
		t.Fatalf("initial magnet run failed: processed=%d err=%v", processed, err)
	}
	if processed, err := service.Run(context.Background()); err != nil || processed != 0 {
		t.Fatalf("pending metadata should not count as completed verification: processed=%d err=%v", processed, err)
	}
	runs, err := (AutomationStore{DB: db}).ListRuns(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].PostVerification == nil || runs[0].PostVerification.Status != "pending" || runs[0].PostVerification.Attempts != 1 {
		t.Fatalf("magnet verification was not kept pending: %+v", runs)
	}
	if client.removeCalls != 0 {
		t.Fatal("pending magnet metadata was removed")
	}
}

func TestAutomationMagnetVerificationAcceptsResolvedSafePayload(t *testing.T) {
	now := time.Date(2026, 9, 17, 19, 0, 0, 0, time.UTC)
	db := automationTestStore(t, now)
	if _, err := db.Exec("UPDATE episodes SET runtime=40 WHERE id='episode-a'"); err != nil {
		t.Fatal(err)
	}
	client := &automationClient{
		listed: true,
		resolvedFiles: []TorrentFile{{Path: "Example.Show.S01E02.1080p.WEB-DL.mkv", Size: 2 * 1024 * 1024 * 1024}},
	}
	service := automationService(db, automationRequester{magnetOnly: true, reportedSize: 2 * 1024 * 1024 * 1024}, client, now)
	if processed, err := service.Run(context.Background()); err != nil || processed != 1 {
		t.Fatalf("initial magnet run failed: processed=%d err=%v", processed, err)
	}
	if processed, err := service.Run(context.Background()); err != nil || processed != 1 {
		t.Fatalf("resolved verification did not complete: processed=%d err=%v", processed, err)
	}
	runs, err := (AutomationStore{DB: db}).ListRuns(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	verification := runs[0].PostVerification
	if verification == nil || verification.Status != "verified" || verification.Assessment == nil || verification.Assessment.Verification != VerificationVerified {
		t.Fatalf("resolved safe payload was not verified: %+v", verification)
	}
	if verification.SizeProfile == nil || !verification.SizeProfile.Known || !verification.SizeProfile.InActiveRange {
		t.Fatalf("resolved payload did not retain size evidence: %+v", verification)
	}
	if runs[0].Verification != VerificationUnverified {
		t.Fatalf("original immutable magnet decision was rewritten: %+v", runs[0])
	}
	if client.removeCalls != 0 {
		t.Fatal("verified magnet was removed")
	}
}

func TestAutomationMagnetVerificationRejectsAndDeletesUnsafePayload(t *testing.T) {
	now := time.Date(2026, 9, 17, 19, 0, 0, 0, time.UTC)
	db := automationTestStore(t, now)
	client := &automationClient{
		listed: true,
		resolvedFiles: []TorrentFile{
			{Path: "Example.Show.S01E02.1080p.WEB-DL.mkv", Size: 2 * 1024 * 1024 * 1024},
			{Path: "setup.exe", Size: 1024},
		},
	}
	service := automationService(db, automationRequester{magnetOnly: true}, client, now)
	if processed, err := service.Run(context.Background()); err != nil || processed != 1 {
		t.Fatalf("initial magnet run failed: processed=%d err=%v", processed, err)
	}
	if processed, err := service.Run(context.Background()); err != nil || processed != 1 {
		t.Fatalf("unsafe post verification did not complete: processed=%d err=%v", processed, err)
	}
	runs, err := (AutomationStore{DB: db}).ListRuns(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	verification := runs[0].PostVerification
	if verification == nil || verification.Status != "rejected" || verification.Assessment == nil || !verification.Assessment.Rejected() {
		t.Fatalf("unsafe payload was not rejected: %+v", verification)
	}
	if client.stopCalls != 1 || client.removeCalls != 1 || !client.deleteFiles {
		t.Fatalf("unsafe magnet was not stopped and deleted: stop=%d remove=%d deleteFiles=%v", client.stopCalls, client.removeCalls, client.deleteFiles)
	}
	blocked, err := (AutomationStore{DB: db}).IsBadInfoHash(context.Background(), runs[0].SelectedInfoHash)
	if err != nil || !blocked {
		t.Fatalf("unsafe magnet hash was not globally blocked: blocked=%v err=%v", blocked, err)
	}
}

func TestAutomationMagnetVerificationRejectsActualSizeOutsideSubmissionRange(t *testing.T) {
	now := time.Date(2026, 9, 17, 19, 0, 0, 0, time.UTC)
	db := automationTestStore(t, now)
	if _, err := db.Exec("UPDATE episodes SET runtime=40 WHERE id='episode-a'"); err != nil {
		t.Fatal(err)
	}
	client := &automationClient{
		listed: true,
		resolvedFiles: []TorrentFile{{Path: "Example.Show.S01E02.mkv", Size: 20 * 1024 * 1024}},
	}
	service := automationService(db, automationRequester{magnetOnly: true, reportedSize: 2 * 1024 * 1024 * 1024}, client, now)
	if processed, err := service.Run(context.Background()); err != nil || processed != 1 {
		t.Fatalf("initial magnet run failed: processed=%d err=%v", processed, err)
	}
	if processed, err := service.Run(context.Background()); err != nil || processed != 1 {
		t.Fatalf("size post verification did not complete: processed=%d err=%v", processed, err)
	}
	runs, err := (AutomationStore{DB: db}).ListRuns(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	verification := runs[0].PostVerification
	if verification == nil || verification.Status != "rejected" || verification.SizeProfile == nil || verification.SizeProfile.InActiveRange {
		t.Fatalf("actual size outlier was not rejected: %+v", verification)
	}
	if client.removeCalls != 1 || !client.deleteFiles {
		t.Fatal("actual size outlier was not removed with files")
	}
}

func TestAutomationPrefersInspectableTorrentWhenMagnetAlsoExists(t *testing.T) {
	now := time.Date(2026, 9, 17, 19, 0, 0, 0, time.UTC)
	db := automationTestStore(t, now)
	client := &automationClient{}
	processed, err := automationService(db, automationRequester{withMagnet: true}, client, now).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 || client.added != 1 || client.magnetAdded != 0 {
		t.Fatalf("inspectable torrent was not preferred: processed=%d torrents=%d magnets=%d", processed, client.added, client.magnetAdded)
	}
}

func TestAutomationRejectsMismatchedTorrentAndMagnetIdentity(t *testing.T) {
	now := time.Date(2026, 9, 17, 19, 0, 0, 0, time.UTC)
	db := automationTestStore(t, now)
	client := &automationClient{}
	requester := automationRequester{withMagnet: true, magnetHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	processed, err := automationService(db, requester, client, now).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 || client.added != 0 || client.magnetAdded != 0 {
		t.Fatalf("mismatched transports reached client: processed=%d torrents=%d magnets=%d", processed, client.added, client.magnetAdded)
	}
	runs, err := (AutomationStore{DB: db}).ListRuns(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].Status != RunNoVerifiedCandidate {
		t.Fatalf("unexpected transport mismatch outcome: %+v", runs)
	}
	found := false
	for _, step := range runs[0].DecisionLog {
		if step.Stage == "inspection" && step.Status == "rejected" {
			found = true
		}
	}
	if !found {
		t.Fatalf("transport mismatch was not recorded: %+v", runs[0].DecisionLog)
	}
}

func TestAutomationFallsBackToMagnetWhenTorrentCannotBeFetched(t *testing.T) {
	now := time.Date(2026, 9, 17, 19, 0, 0, 0, time.UTC)
	db := automationTestStore(t, now)
	client := &automationClient{}
	requester := automationRequester{withMagnet: true, beforeTorrent: func() error { return errors.New("torrent unavailable") }}
	processed, err := automationService(db, requester, client, now).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 || client.added != 0 || client.magnetAdded != 1 {
		t.Fatalf("torrent fetch failure did not use magnet fallback: processed=%d torrents=%d magnets=%d", processed, client.added, client.magnetAdded)
	}
}

func TestAutomationFiltersImplausibleKnownSizeRate(t *testing.T) {
	now := time.Date(2026, 9, 17, 19, 0, 0, 0, time.UTC)
	db := automationTestStore(t, now)
	if _, err := db.Exec("UPDATE episodes SET runtime=40 WHERE id='episode-a'"); err != nil {
		t.Fatal(err)
	}
	client := &automationClient{}
	processed, err := automationService(db, automationRequester{magnetOnly: true, reportedSize: 20 * 1024 * 1024}, client, now).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 || client.added != 0 || client.magnetAdded != 0 {
		t.Fatalf("size outlier reached client: processed=%d torrents=%d magnets=%d", processed, client.added, client.magnetAdded)
	}
	runs, err := (AutomationStore{DB: db}).ListRuns(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].Status != RunNoVerifiedCandidate {
		t.Fatalf("unexpected size-filter outcome: %+v", runs)
	}
	found := false
	for _, step := range runs[0].DecisionLog {
		if step.Stage == "filter" {
			if count, ok := step.Data["size_rate_filtered"].(float64); ok && count == 1 {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("size-rate rejection was not recorded: %+v", runs[0].DecisionLog)
	}
}

func TestAutomationWaitsForMinimumReleaseAgeWhenJackettReportsUploadTime(t *testing.T) {
	now := time.Date(2026, 9, 17, 19, 0, 0, 0, time.UTC)
	db := automationTestStore(t, now)
	client := &automationClient{}
	requester := automationRequester{
		magnetOnly: true,
		published:  now.Add(-5 * time.Minute).Format(time.RFC1123Z),
	}
	processed, err := automationService(db, requester, client, now).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 || client.added != 0 || client.magnetAdded != 0 {
		t.Fatalf("too-new release reached client: processed=%d torrents=%d magnets=%d", processed, client.added, client.magnetAdded)
	}
	runs, err := (AutomationStore{DB: db}).ListRuns(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].Status != RunNoVerifiedCandidate {
		t.Fatalf("unexpected release-delay outcome: %+v", runs)
	}
	found := false
	for _, step := range runs[0].DecisionLog {
		if step.Stage == "filter" {
			if count, ok := step.Data["release_delay_filtered"].(float64); ok && count == 1 {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("release-age delay was not recorded: %+v", runs[0].DecisionLog)
	}
}

func TestAutomationAllowsReleasePastMinimumUploadAge(t *testing.T) {
	now := time.Date(2026, 9, 17, 19, 0, 0, 0, time.UTC)
	db := automationTestStore(t, now)
	client := &automationClient{}
	requester := automationRequester{
		magnetOnly: true,
		published:  now.Add(-25 * time.Minute).Format(time.RFC1123Z),
	}
	processed, err := automationService(db, requester, client, now).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 || client.magnetAdded != 1 {
		t.Fatalf("old-enough release was not submitted: processed=%d magnets=%d", processed, client.magnetAdded)
	}
}

func TestReleaseAgeMissingOrMalformedTimestampIsNeutral(t *testing.T) {
	now := time.Date(2026, 9, 17, 19, 0, 0, 0, time.UTC)
	for _, published := range []string{"", "not-a-date"} {
		result := EvaluateReleaseAge(published, now, 20*time.Minute)
		if result.Known || !result.Ready {
			t.Fatalf("timestamp %q should be neutral: %+v", published, result)
		}
	}
}

func TestAutomationNoOpsWhenRequiredCapabilityIsDisabled(t *testing.T) {
	now := time.Date(2026, 9, 17, 19, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name                          string
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
		ctx := context.Background()
		store := settings.Store{DB: db}
		var downloads settings.Torrent
		revision, err := store.Load(ctx, "torrent", &downloads)
		if err != nil {
			return err
		}
		downloads.Enabled = false
		_, err = store.Save(ctx, "torrent", downloads, revision)
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
