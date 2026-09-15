package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/CarlFlo/tally/internal/activity"
	"github.com/CarlFlo/tally/internal/settings"
)

type Schedule struct {
	Key      string `json:"key"`
	Schedule string `json:"schedule"`
	Enabled  bool   `json:"enabled"`
	Revision int64  `json:"revision"`
}

func (s *Service) SaveSchedule(ctx context.Context, in Schedule, actors ...string) error {
	actor := ""
	if len(actors) > 0 {
		actor = actors[0]
	}
	if in.Key != "metadata" && in.Key != "maintenance" && in.Key != "backup" {
		return fmt.Errorf("unknown job")
	}
	if len(in.Schedule) > 100 {
		return fmt.Errorf("cron schedule is too long")
	}
	next, err := s.nextScheduledRun(in.Schedule, time.Now())
	if err != nil {
		return err
	}
	if next.IsZero() {
		return fmt.Errorf("schedule has no next occurrence")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var oldSchedule string
	var oldEnabled bool
	var currentRevision int64
	if err = tx.QueryRowContext(ctx, "SELECT schedule,enabled,revision FROM jobs WHERE key=?", in.Key).Scan(&oldSchedule, &oldEnabled, &currentRevision); err != nil {
		return err
	}
	if currentRevision != in.Revision {
		return settings.ErrConflict
	}
	if oldSchedule == in.Schedule && oldEnabled == in.Enabled {
		return nil
	}
	res, err := tx.ExecContext(ctx, "UPDATE jobs SET schedule=?,enabled=?,next_run=CASE WHEN ?=0 OR paused=1 THEN 0 ELSE ? END,revision=revision+1 WHERE key=? AND revision=?", in.Schedule, in.Enabled, in.Enabled, next.Unix(), in.Key, in.Revision)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return settings.ErrConflict
	}
	verb := "Updated"
	if oldEnabled != in.Enabled {
		if in.Enabled {
			verb = "Enabled"
		} else {
			verb = "Disabled"
		}
	}
	message := fmt.Sprintf("%s %s schedule (%s %s)", verb, in.Key, in.Schedule, s.scheduleTimezone())
	if err = activity.Record(ctx, tx, activity.Event{Action: "schedule_updated", Profile: actor, Message: message}); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	s.wakeScheduler()
	return nil
}

func (s *Service) Resume(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var spec string
	if err := s.DB.QueryRowContext(ctx, "SELECT schedule FROM jobs WHERE key=?", key).Scan(&spec); err != nil {
		return fmt.Errorf("unknown job")
	}
	next, err := s.nextScheduledRun(spec, time.Now())
	if err != nil {
		return err
	}
	if _, err = s.DB.ExecContext(ctx, "UPDATE jobs SET paused=0,failures=0,next_run=CASE WHEN enabled=1 THEN ? ELSE 0 END,revision=revision+1 WHERE key=?", next.Unix(), key); err != nil {
		return err
	}
	s.wakeScheduler()
	return nil
}
