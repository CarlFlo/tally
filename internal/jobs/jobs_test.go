package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/CarlFlo/mediaManager/internal/config"
	"github.com/CarlFlo/mediaManager/internal/database"
	"github.com/CarlFlo/mediaManager/internal/metadata"
	"github.com/CarlFlo/mediaManager/internal/providers"
)

type blockingTV struct{ started chan struct{} }

func (p *blockingTV) SearchShows(context.Context, string) ([]metadata.SearchResult, error) {
	return nil, nil
}
func (p *blockingTV) GetShow(ctx context.Context, _ string) (*metadata.Show, error) {
	select {
	case p.started <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return nil, ctx.Err()
}
func (p *blockingTV) GetEpisodes(context.Context, string) ([]metadata.Episode, error) {
	return nil, nil
}
func TestJobDedupCancellationAndDeadline(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, e := database.Open(ctx, dir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	p, e := providers.New(ctx, db, dir, 2, 0)
	if e != nil {
		t.Fatal(e)
	}
	defer p.Close()
	_, e = db.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('user0','Fixture','violet',1); INSERT INTO shows(id,name) VALUES('show','Example'); INSERT INTO external_ids VALUES('tvmaze','show','7','show'); INSERT INTO profile_shows(profile_id,show_id,added_at) VALUES('user0','show',1)")
	if e != nil {
		t.Fatal(e)
	}
	fake := &blockingTV{started: make(chan struct{}, 1)}
	s := New(ctx, db, config.Config{JobConcurrency: 1, JobRuntime: time.Second, BatchSize: 5}, &metadata.Service{DB: db, Provider: fake}, p, nil)
	defer s.Stop(context.Background())
	id, e := s.Trigger("metadata", "manual_refresh", "show")
	if e != nil {
		t.Fatal(e)
	}
	select {
	case <-fake.started:
	case <-time.After(2 * time.Second):
		t.Fatal("job did not start")
	}
	if _, e = s.Trigger("metadata", "manual_refresh", "show"); e == nil {
		t.Fatal("duplicate logical job accepted")
	}
	if !s.Cancel(id) {
		t.Fatal("cancellation not accepted")
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var status string
		_ = db.QueryRow("SELECT status FROM job_runs WHERE id=?", id).Scan(&status)
		if status == "cancelled" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("cancelled job not persisted")
}
func TestAlertDeduplication(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, e := database.Open(ctx, dir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	s := New(ctx, db, config.Config{JobConcurrency: 1}, nil, nil, nil)
	s.Alert("provider:test", "error", "Unavailable")
	s.Alert("provider:test", "error", "Unavailable")
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM alerts").Scan(&count)
	if count != 1 {
		t.Fatal("alerts not deduplicated")
	}
	s.Alert("provider:test", "info", "Recovered")
	var message string
	_ = db.QueryRow("SELECT message FROM alerts").Scan(&message)
	if message != "Recovered" {
		t.Fatal("recovery alert not updated")
	}
}

func TestJobDeadlineCountsAsFailure(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, e := database.Open(ctx, dir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	p, e := providers.New(ctx, db, dir, 2, 0)
	if e != nil {
		t.Fatal(e)
	}
	defer p.Close()
	if _, e = db.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('user0','Fixture','violet',1); INSERT INTO shows(id,name) VALUES('show','Example'); INSERT INTO external_ids VALUES('tvmaze','show','7','show'); INSERT INTO profile_shows(profile_id,show_id,added_at) VALUES('user0','show',1)"); e != nil {
		t.Fatal(e)
	}
	s := New(ctx, db, config.Config{JobRuntime: 50 * time.Millisecond, BatchSize: 5}, &metadata.Service{DB: db, Provider: &blockingTV{started: make(chan struct{}, 1)}}, p, nil)
	defer s.Stop(ctx)
	if e = s.initializeSchedules(); e != nil {
		t.Fatal(e)
	}
	id, e := s.Trigger("metadata", "manual_refresh", "")
	if e != nil {
		t.Fatal(e)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var status string
		var failures int
		_ = db.QueryRow("SELECT status FROM job_runs WHERE id=?", id).Scan(&status)
		_ = db.QueryRow("SELECT failures FROM jobs WHERE key='metadata'").Scan(&failures)
		if status == "failed" && failures == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("job timeout did not count toward repeated failure pause")
}
