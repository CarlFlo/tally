package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/config"
	"github.com/CarlFlo/tally/internal/database"
)

func TestNextScheduleDelayUsesNearestEnabledRun(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := New(ctx, db, config.Config{Timezone: "UTC"}, nil, nil, nil)
	defer service.Stop(ctx)
	if err = service.initializeSchedules(); err != nil {
		t.Fatal(err)
	}

	now := time.Now().Truncate(time.Second)
	if _, err = db.Exec("UPDATE jobs SET enabled=0,next_run=0"); err != nil {
		t.Fatal(err)
	}
	if got := service.nextScheduleDelay(now); got != idleScheduleCheck {
		t.Fatalf("idle delay = %v, want %v", got, idleScheduleCheck)
	}

	if _, err = db.Exec("UPDATE jobs SET enabled=1,paused=0,next_run=? WHERE key='backup'", now.Add(42*time.Second).Unix()); err != nil {
		t.Fatal(err)
	}
	got := service.nextScheduleDelay(now)
	if got < 41*time.Second || got > 43*time.Second {
		t.Fatalf("next delay = %v, want about 42s", got)
	}

	if _, err = db.Exec("UPDATE jobs SET next_run=? WHERE key='backup'", now.Add(-time.Minute).Unix()); err != nil {
		t.Fatal(err)
	}
	if got = service.nextScheduleDelay(now); got != time.Second {
		t.Fatalf("overdue delay = %v, want 1s", got)
	}
}
