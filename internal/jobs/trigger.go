package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/CarlFlo/mediaManager/internal/activity"
	"github.com/CarlFlo/mediaManager/internal/database"
	"github.com/CarlFlo/mediaManager/internal/scheduling"
)

func (s *Service) Trigger(kind, trigger, show string) (string, error) {
	if kind != "metadata" && kind != "maintenance" && kind != "backup" {
		return "", fmt.Errorf("unknown job")
	}
	key := kind
	if show != "" {
		key = "metadata:tvmaze:show:" + show
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return "", fmt.Errorf("application is shutting down")
	}
	if trigger == "scheduled_refresh" {
		var enabled, paused bool
		if e := s.DB.QueryRow("SELECT enabled,paused FROM jobs WHERE key=?", kind).Scan(&enabled, &paused); e != nil || !enabled || paused {
			return "", fmt.Errorf("job schedule is disabled or paused")
		}
	}
	if _, ok := s.running[key]; ok {
		return "", fmt.Errorf("this job is already running")
	}
	select {
	case s.sem <- struct{}{}:
	default:
		return "", fmt.Errorf("all job slots are busy; try again shortly")
	}
	id := database.ID()
	ctx, cancel := context.WithTimeout(s.ctx, s.Config.JobRuntime)
	_, e := s.DB.ExecContext(ctx, "INSERT INTO job_runs(id,job_key,trigger,started_at,status) VALUES(?,?,?,?,'running')", id, key, trigger, time.Now().Unix())
	if e != nil {
		cancel()
		<-s.sem
		return "", e
	}
	if show == "" {
		var spec string
		if s.DB.QueryRow("SELECT schedule FROM jobs WHERE key=?", kind).Scan(&spec) == nil {
			if parsed, e := scheduling.Parse(spec); e == nil {
				_, _ = s.DB.Exec("UPDATE jobs SET next_run=CASE WHEN enabled=1 AND paused=0 THEN ? ELSE 0 END WHERE key=?", parsed.Next(time.Now().UTC()).Unix(), kind)
			}
		}
	}
	s.changed()
	s.running[key] = cancel
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() { cancel(); <-s.sem; s.mu.Lock(); delete(s.running, key); s.mu.Unlock() }()
		started := time.Now()
		result, e := s.run(ctx, id, kind, trigger, show)
		status := "success"
		errorText := ""
		if e != nil {
			status = "failed"
			errorText = e.Error()
			if errors.Is(ctx.Err(), context.Canceled) {
				status = "cancelled"
			}
			s.Alert("job:"+key, "error", key+": "+errorText)
		}
		var calls, hits int
		_ = s.DB.QueryRow("SELECT COALESCE(SUM(CASE WHEN reason IN ('request','conditional cache hit') THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN reason IN ('fresh cache','conditional cache hit') THEN 1 ELSE 0 END),0) FROM provider_requests WHERE job_id=?", id).Scan(&calls, &hits)
		_, err := s.DB.Exec("UPDATE job_runs SET ended_at=?,duration_ms=?,status=?,error=?,candidates=?,processed=?,skipped=?,changes=?,attempt=?,api_calls=?,cache_hits=? WHERE id=?", time.Now().Unix(), time.Since(started).Milliseconds(), status, errorText, result.Candidates, result.Processed, result.Skipped, result.Changes, result.Attempt, calls, hits, id)
		if err != nil {
			slog.Error("persist job result", "job_id", id, "error", err)
		}
		if show == "" {
			s.finishSchedule(kind, status)
		}
		if status == "success" && (kind == "backup" || trigger == "manual" || trigger == "manual_refresh") {
			label := kind
			if show != "" {
				_ = s.DB.QueryRow("SELECT name FROM shows WHERE id=?", show).Scan(&label)
			}
			if err := activity.Record(s.ctx, s.DB, activity.Event{Action: "job_succeeded", Profile: "user0", Message: "Completed " + label + " job"}); err != nil {
				slog.Error("record job activity", "error", err)
			}
		}
		slog.Info("job finished", "job_id", id, "job_key", key, "status", status, "processed", result.Processed)
		s.changed()
	}()
	return id, nil
}

func (s *Service) Cancel(id string) bool {
	var key string
	if s.DB.QueryRow("SELECT job_key FROM job_runs WHERE id=? AND status='running'", id).Scan(&key) != nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if cancel, ok := s.running[key]; ok {
		cancel()
		return true
	}
	return false
}
