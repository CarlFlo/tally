package jobs

import "time"

func (s *Service) finishSchedule(key, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if status == "failed" {
		_, _ = s.DB.Exec("UPDATE jobs SET failures=failures+1,paused=CASE WHEN failures+1>=3 THEN 1 ELSE paused END,next_run=CASE WHEN failures+1>=3 THEN 0 ELSE next_run END WHERE key=?", key)
	} else if status == "success" {
		_, _ = s.DB.Exec("UPDATE jobs SET failures=0,paused=0 WHERE key=?", key)
		_, _ = s.DB.Exec("UPDATE alerts SET active=0 WHERE key=?", "job:"+key)
	}
	var spec string
	var enabled, paused bool
	if s.DB.QueryRow("SELECT schedule,enabled,paused FROM jobs WHERE key=?", key).Scan(&spec, &enabled, &paused) == nil {
		var next int64
		if enabled && !paused {
			if nextRun, err := s.nextScheduledRun(spec, time.Now()); err == nil {
				next = nextRun.Unix()
			}
		}
		_, _ = s.DB.Exec("UPDATE jobs SET next_run=? WHERE key=?", next, key)
	}
	s.wakeScheduler()
}
