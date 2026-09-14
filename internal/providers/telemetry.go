package providers

import (
	"context"
	"log/slog"
	"time"
)

func (c *Coordinator) Avoid(provider, trigger, entity, reason string) {
	c.record(Request{Provider: provider, Trigger: trigger, Entity: entity}, 0, reason, 0, 0)
}

func (c *Coordinator) record(r Request, status int, reason string, retry int, d time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	now := time.Now()
	_, e := c.db.ExecContext(ctx, "INSERT INTO provider_requests(provider,trigger,entity,job_id,status_code,reason,retry_count,duration_ms,created_at) VALUES(?,?,?,?,?,?,?,?,?)", r.Provider, r.Trigger, r.Entity, r.JobID, status, reason, retry, d.Milliseconds(), now.Unix())
	if e != nil {
		slog.Error("record request", "error", e)
		return
	}
	call, success, failure, retried, rate, cache, conditional, avoided := 0, 0, 0, 0, 0, 0, 0, 0
	if reason == "request" || reason == "conditional cache hit" {
		call = 1
		if status >= 200 && status < 400 {
			success = 1
		} else {
			failure = 1
		}
		if retry > 0 {
			retried = 1
		}
		if status == 429 {
			rate = 1
		}
	} else {
		avoided = 1
	}
	if reason == "fresh cache" {
		cache = 1
	}
	if status == 304 {
		conditional = 1
		avoided = 1
	}
	_, e = c.db.ExecContext(ctx, `INSERT INTO provider_request_aggregates VALUES(?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(day,provider) DO UPDATE SET requests=requests+excluded.requests,successes=successes+excluded.successes,failures=failures+excluded.failures,retries=retries+excluded.retries,rate_limited=rate_limited+excluded.rate_limited,cache_hits=cache_hits+excluded.cache_hits,conditional_hits=conditional_hits+excluded.conditional_hits,avoided=avoided+excluded.avoided,latency_ms=latency_ms+excluded.latency_ms`, now.UTC().Format("2006-01-02"), r.Provider, call, success, failure, retried, rate, cache, conditional, avoided, d.Milliseconds())
	if e != nil {
		slog.Error("aggregate request", "error", e)
	}
	slog.Debug("provider request", "provider", r.Provider, "trigger", r.Trigger, "entity", r.Entity, "job_id", r.JobID, "status_code", status, "reason", reason, "retry_count", retry, "duration_ms", d.Milliseconds())
}
