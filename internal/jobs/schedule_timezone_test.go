package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/config"
	"github.com/CarlFlo/tally/internal/database"
)

func TestConfiguredTimezoneControlsStoredNextRun(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := database.Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := New(ctx, db, config.Config{Timezone: "Europe/Stockholm"}, nil, nil, nil)
	defer service.Stop(ctx)
	if err = service.initializeSchedules(); err != nil {
		t.Fatal(err)
	}
	var next int64
	if err = db.QueryRow("SELECT next_run FROM jobs WHERE key='backup'").Scan(&next); err != nil {
		t.Fatal(err)
	}
	location, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		t.Fatal(err)
	}
	run := time.Unix(next, 0).In(location)
	if run.Hour() != 3 || run.Minute() != 0 {
		t.Fatalf("backup scheduled at %v, want 03:00 Europe/Stockholm", run)
	}
}
