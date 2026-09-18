package jobs

import (
	"context"
	"testing"

	"github.com/CarlFlo/tally/internal/config"
	"github.com/CarlFlo/tally/internal/database"
)

type fakeTorrentAutomator struct {
	runs      int
	processed int
	err       error
}

func (f *fakeTorrentAutomator) Run(context.Context) (int, error) {
	f.runs++
	return f.processed, f.err
}

func TestTorrentAutomationIsIndependentScheduledJob(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	automator := &fakeTorrentAutomator{processed: 2}
	service := New(ctx, db, config.Config{JobConcurrency: 1}, nil, nil, nil)
	service.TorrentAutomation = automator
	if err = service.initializeSchedules(); err != nil {
		t.Fatal(err)
	}

	var schedule string
	var enabled int
	if err = db.QueryRow("SELECT schedule,enabled FROM jobs WHERE key='torrent_automation'").Scan(&schedule, &enabled); err != nil {
		t.Fatal(err)
	}
	if schedule != "*/15 * * * *" || enabled != 0 {
		t.Fatalf("unexpected torrent automation schedule: %q enabled=%d", schedule, enabled)
	}

	if _, err = service.TriggerAndWait(ctx, "torrent_automation", "scheduled_refresh", ""); err == nil {
		t.Fatal("disabled experimental torrent automation job accepted a scheduled run")
	}
	var revision int64
	if err = db.QueryRow("SELECT revision FROM jobs WHERE key='torrent_automation'").Scan(&revision); err != nil {
		t.Fatal(err)
	}
	if err = service.SaveSchedule(ctx, Schedule{
		Key: "torrent_automation", Schedule: schedule, Enabled: true, Revision: revision,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = service.TriggerAndWait(ctx, "torrent_automation", "scheduled_refresh", ""); err != nil {
		t.Fatal(err)
	}
	if automator.runs != 1 {
		t.Fatalf("enabled scheduler did not execute torrent automation: runs=%d", automator.runs)
	}
}
