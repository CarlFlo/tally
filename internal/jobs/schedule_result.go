package jobs

import (
	"log/slog"
	"time"
)

func (s *Service) finishSchedule(key, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if status == "failed" {
		if _, err := s.DB.Exec("UPDATE jobs SET failures=failures+1,paused=CASE WHEN failures+1>=3 THEN 1 ELSE paused END,next_run=CASE WHEN failures+1>=3 THEN 0 ELSE next_run END WHERE key=?", key); err != nil {
			slog.Error("persist failed schedule result", "job_key", key, "error", err)
		}
	} else if status == "success" {
		if _, err := s.DB.Exec("UPDATE jobs SET failures=0,paused=0 WHERE key=?", key); err != nil {
			slog.Error("persist successful schedule result", "job_key", key, "error", err)
		}
		if _, err := s.DB.Exec("UPDATE alerts SET active=0 WHERE key=?", "job:"+key); err != nil {
			slog.Warn("clear recovered job alert", "job_key", key, "error", err)
		}
	}
	var spec string
	var enabled, paused bool
	if err := s.DB.QueryRow("SELECT schedule,enabled,paused FROM jobs WHERE key=?", key).Scan(&spec, &enabled, &paused); err != nil {
		slog.Error("read schedule after job result", "job_key", key, "error", err)
		s.wakeScheduler()
		return
	}
	var next int64
	if enabled && !paused {
		nextRun, err := s.nextScheduledRun(spec, time.Now())
		if err != nil {
			slog.Error("calculate next schedule after job result", "job_key", key, "error", err)
		} else {
			next = nextRun.Unix()
		}
	}
	if _, err := s.DB.Exec("UPDATE jobs SET next_run=? WHERE key=?", next, key); err != nil {
		slog.Error("persist next schedule after job result", "job_key", key, "error", err)
	}
	s.wakeScheduler()
}
