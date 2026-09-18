package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/CarlFlo/tally/internal/activity"
	"github.com/CarlFlo/tally/internal/database"
)

func (s *Service) Trigger(kind, trigger, show string) (string, error) {
	id, _, err := s.trigger(kind, trigger, show)
	return id, err
}

func (s *Service) TriggerAndWait(ctx context.Context, kind, trigger, show string) (string, error) {
	id, done, err := s.trigger(kind, trigger, show)
	if err != nil {
		return "", err
	}
	select {
	case runErr := <-done:
		return id, runErr
	case <-ctx.Done():
		return id, ctx.Err()
	}
}

func (s *Service) trigger(kind, trigger, show string) (string, <-chan error, error) {
	if kind != "metadata" && kind != "maintenance" && kind != "backup" {
		return "", nil, fmt.Errorf("unknown job")
	}
	key := kind
	if show != "" {
		key = "metadata:tvmaze:show:" + show
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return "", nil, fmt.Errorf("application is shutting down")
	}
	if trigger == "scheduled_refresh" {
		var enabled, paused bool
		if err := s.DB.QueryRow("SELECT enabled,paused FROM jobs WHERE key=?", kind).Scan(&enabled, &paused); err != nil || !enabled || paused {
			return "", nil, fmt.Errorf("job schedule is disabled or paused")
		}
	}
	if _, ok := s.running[key]; ok {
		return "", nil, fmt.Errorf("this job is already running")
	}
	select {
	case s.sem <- struct{}{}:
	default:
		return "", nil, fmt.Errorf("all job slots are busy; try again shortly")
	}
	id := database.ID()
	ctx, cancel := context.WithTimeout(s.ctx, s.Config.JobRuntime)
	_, err := s.DB.ExecContext(ctx, "INSERT INTO job_runs(id,job_key,trigger,started_at,status) VALUES(?,?,?,?,'running')", id, key, trigger, time.Now().Unix())
	if err != nil {
		cancel()
		<-s.sem
		return "", nil, err
	}
	if show == "" {
		var spec string
		if s.DB.QueryRow("SELECT schedule FROM jobs WHERE key=?", kind).Scan(&spec) == nil {
			if next, nextErr := s.nextScheduledRun(spec, time.Now()); nextErr == nil {
				_, _ = s.DB.Exec("UPDATE jobs SET next_run=CASE WHEN enabled=1 AND paused=0 THEN ? ELSE 0 END WHERE key=?", next.Unix(), kind)
				s.wakeScheduler()
			}
		}
	}
	s.changed("jobs", "statistics")
	done := make(chan error, 1)
	s.running[key] = cancel
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() { cancel(); <-s.sem; s.mu.Lock(); delete(s.running, key); s.mu.Unlock() }()
		started := time.Now()
		result, runErr := s.run(ctx, id, kind, trigger, show)
		status := "success"
		errorText := ""
		if runErr != nil {
			status = "failed"
			errorText = runErr.Error()
			if errors.Is(ctx.Err(), context.Canceled) {
				status = "cancelled"
			}
			s.Alert("job:"+key, "error", key+": "+errorText)
		}
		var calls, hits int
		_ = s.DB.QueryRow("SELECT COALESCE(SUM(CASE WHEN reason IN ('request','conditional cache hit') THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN reason IN ('fresh cache','conditional cache hit') THEN 1 ELSE 0 END),0) FROM provider_requests WHERE job_id=?", id).Scan(&calls, &hits)
		_, persistErr := s.DB.Exec("UPDATE job_runs SET ended_at=?,duration_ms=?,status=?,error=?,candidates=?,processed=?,skipped=?,changes=?,attempt=?,api_calls=?,cache_hits=? WHERE id=?", time.Now().Unix(), time.Since(started).Milliseconds(), status, errorText, result.Candidates, result.Processed, result.Skipped, result.Changes, result.Attempt, calls, hits, id)
		if persistErr != nil {
			slog.Error("persist job result", "job_id", id, "error", persistErr)
		}
		if show == "" {
			s.finishSchedule(kind, status)
		}
		if status == "success" && (kind == "backup" || trigger == "manual" || trigger == "manual_refresh") {
			label := kind
			if show != "" {
				_ = s.DB.QueryRow("SELECT name FROM shows WHERE id=?", show).Scan(&label)
			}
			if err := activity.Record(s.ctx, s.DB, activity.Event{Action: "job_succeeded", Message: "Completed " + label + " job"}); err != nil {
				slog.Error("record job activity", "error", err)
			}
		}
		slog.Info("job finished", "job_id", id, "job_key", key, "status", status, "processed", result.Processed)
		resources := []string{"jobs", "statistics", "logs"}
		if status == "success" && kind == "backup" {
			resources = append(resources, "backups")
		}
		s.changed(resources...)
		if status == "success" && (kind == "backup" || trigger == "manual" || trigger == "manual_refresh") {
			s.changedProfile("", "inbox")
		}
		done <- runErr
		close(done)
	}()
	return id, done, nil
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
