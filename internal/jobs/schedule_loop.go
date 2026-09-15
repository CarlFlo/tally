package jobs

import "time"

const idleScheduleCheck = time.Hour

func (s *Service) wakeScheduler() {
	select {
	case s.scheduleWake <- struct{}{}:
	default:
	}
}

func (s *Service) nextScheduleDelay(now time.Time) time.Duration {
	var next int64
	if err := s.DB.QueryRowContext(s.ctx, "SELECT COALESCE(MIN(next_run),0) FROM jobs WHERE enabled=1 AND paused=0 AND next_run>0").Scan(&next); err != nil || next == 0 {
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
		return
	}
	for _, row := range rows {
		_, _ = s.Trigger(row["key"].(string), "scheduled_refresh", "")
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
