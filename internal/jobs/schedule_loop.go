package jobs

import (
	"log/slog"
	"time"
)

const idleScheduleCheck = time.Hour

func (s *Service) wakeScheduler() {
	select {
	case s.scheduleWake <- struct{}{}:
	default:
	}
}

// RefreshSchedules asks the scheduler loop to re-read authoritative schedule
// state immediately. It is used after live database restoration.
func (s *Service) RefreshSchedules() {
	s.wakeScheduler()
}

func (s *Service) nextScheduleDelay(now time.Time) time.Duration {
	var next int64
	if err := s.DB.QueryRowContext(s.ctx, "SELECT COALESCE(MIN(next_run),0) FROM jobs WHERE enabled=1 AND paused=0 AND next_run>0").Scan(&next); err != nil {
		slog.Error("read next scheduled job", "error", err)
		return idleScheduleCheck
	}
	if next == 0 {
		return idleScheduleCheck
	}
	delay := time.Unix(next, 0).Sub(now)
	if delay < time.Second {
		return time.Second
	}
	return delay
}

func (s *Service) runDueSchedules(now time.Time) {
	rows, err := s.DB.Rows(s.ctx, "SELECT key FROM jobs WHERE enabled=1 AND paused=0 AND next_run>0 AND next_run<=?", now.Unix())
	if err != nil {
		slog.Error("read due scheduled jobs", "error", err)
		return
	}
	for _, row := range rows {
		key := row["key"].(string)
		if _, err = s.Trigger(key, "scheduled_refresh", ""); err != nil {
			slog.Warn("scheduled job was not started", "job_key", key, "error", err)
		}
	}
}

func (s *Service) scheduleLoop() {
	defer s.wg.Done()
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-timer.C:
			s.runDueSchedules(time.Now())
		case <-s.scheduleWake:
		}
		timer.Reset(s.nextScheduleDelay(time.Now()))
	}
}
