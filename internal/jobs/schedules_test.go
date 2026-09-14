package jobs

import (
	"context"
	"errors"
	"testing"

	"github.com/CarlFlo/mediaManager/internal/config"
	"github.com/CarlFlo/mediaManager/internal/database"
	"github.com/CarlFlo/mediaManager/internal/settings"
)

func TestSchedulesPersistPauseAfterFailuresAndResume(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, e := database.Open(ctx, dir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	s := New(ctx, db, config.Config{}, nil, nil, nil)
	defer s.Stop(ctx)
	if e = s.initializeSchedules(); e != nil {
		t.Fatal(e)
	}
	if e = s.SaveSchedule(ctx, Schedule{Key: "metadata", Schedule: "15 8 * * 1-5", Enabled: false, Revision: 1}); e != nil {
		t.Fatal(e)
	}
	if e = s.SaveSchedule(ctx, Schedule{Key: "metadata", Schedule: "15 8 * * 1-5", Enabled: false, Revision: 2}); e != nil {
		t.Fatal("unchanged schedule save failed", e)
	}
	if e = s.SaveSchedule(ctx, Schedule{Key: "metadata", Schedule: "bad", Enabled: true, Revision: 2}); e == nil {
		t.Fatal("bad schedule accepted")
	}
	if e = s.SaveSchedule(ctx, Schedule{Key: "metadata", Schedule: "0 1 * * *", Enabled: true, Revision: 1}); !errors.Is(e, settings.ErrConflict) {
		t.Fatal("stale schedule overwrote newer edit")
	}
	var logged int
	if err := db.QueryRow("SELECT COUNT(*) FROM activity_log WHERE action='schedule_updated' AND message='Disabled metadata schedule (15 8 * * 1-5 UTC)'").Scan(&logged); err != nil || logged != 1 {
		t.Fatal("schedule success/conflict activity incorrect", err)
	}
	if e = s.initializeSchedules(); e != nil {
		t.Fatal(e)
	}
	var enabled, paused bool
	var next int64
	var spec string
	db.QueryRow("SELECT schedule,enabled,next_run FROM jobs WHERE key='metadata'").Scan(&spec, &enabled, &next)
	if enabled || next != 0 || spec != "15 8 * * 1-5" {
		t.Fatal("restart replaced disabled schedule")
	}
	for i := 0; i < 3; i++ {
		s.finishSchedule("backup", "failed")
	}
	db.QueryRow("SELECT paused,next_run FROM jobs WHERE key='backup'").Scan(&paused, &next)
	if !paused || next != 0 {
		t.Fatal("repeated failures did not pause schedule")
	}
	if _, e = s.Trigger("backup", "scheduled_refresh", ""); e == nil {
		t.Fatal("paused job started automatically")
	}
	if e = s.Resume(ctx, "backup"); e != nil {
		t.Fatal(e)
	}
	db.QueryRow("SELECT paused,next_run FROM jobs WHERE key='backup'").Scan(&paused, &next)
	if paused || next <= 0 {
		t.Fatal("resume did not schedule next run")
	}
}
