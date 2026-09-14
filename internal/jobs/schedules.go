package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/CarlFlo/mediaManager/internal/activity"
	"github.com/CarlFlo/mediaManager/internal/scheduling"
	"github.com/CarlFlo/mediaManager/internal/settings"
)

type Schedule struct {
	Key      string `json:"key"`
	Schedule string `json:"schedule"`
	Enabled  bool   `json:"enabled"`
	Revision int64  `json:"revision"`
}

func (s *Service) SaveSchedule(ctx context.Context, in Schedule) error {
	if in.Key != "metadata" && in.Key != "maintenance" && in.Key != "backup" {
		return fmt.Errorf("unknown job")
	}
	if len(in.Schedule) > 100 {
		return fmt.Errorf("cron schedule is too long")
	}
	parsed, e := scheduling.Parse(in.Schedule)
	if e != nil {
		return e
	}
	next := parsed.Next(time.Now().UTC())
	if next.IsZero() {
		return fmt.Errorf("schedule has no next occurrence")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var oldSchedule string
	var oldEnabled bool
	var currentRevision int64
	if e = tx.QueryRowContext(ctx, "SELECT schedule,enabled,revision FROM jobs WHERE key=?", in.Key).Scan(&oldSchedule, &oldEnabled, &currentRevision); e != nil {
		return e
	}
	if currentRevision != in.Revision {
		return settings.ErrConflict
	}
	if oldSchedule == in.Schedule && oldEnabled == in.Enabled {
		return nil
	}
	res, e := tx.ExecContext(ctx, "UPDATE jobs SET schedule=?,enabled=?,next_run=CASE WHEN ?=0 OR paused=1 THEN 0 ELSE ? END,revision=revision+1 WHERE key=? AND revision=?", in.Schedule, in.Enabled, in.Enabled, next.Unix(), in.Key, in.Revision)
	if e != nil {
		return e
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
	if e = activity.Record(ctx, tx, activity.Event{Action: "schedule_updated", Profile: "user0", Message: fmt.Sprintf("%s %s schedule (%s UTC)", verb, in.Key, in.Schedule)}); e != nil {
		return e
	}
	return tx.Commit()
}

func (s *Service) Resume(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var spec string
	if e := s.DB.QueryRowContext(ctx, "SELECT schedule FROM jobs WHERE key=?", key).Scan(&spec); e != nil {
		return fmt.Errorf("unknown job")
	}
	parsed, e := scheduling.Parse(spec)
	if e != nil {
		return e
	}
	_, e = s.DB.ExecContext(ctx, "UPDATE jobs SET paused=0,failures=0,next_run=CASE WHEN enabled=1 THEN ? ELSE 0 END,revision=revision+1 WHERE key=?", parsed.Next(time.Now().UTC()).Unix(), key)
	return e
}
