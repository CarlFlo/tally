package providers

import (
	"log/slog"
	"sync"
	"time"
)

type state struct {
	sem                   chan struct{}
	mu                    sync.Mutex
	next, blocked, window time.Time
	failures, opens       int
	mode                  string
	probe                 bool
}

func (c *Coordinator) State(provider string) *state {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.states[provider]
	if s == nil {
		s = &state{sem: make(chan struct{}, c.concurrency), mode: "healthy"}
		var blocked, window int64
		_ = c.db.QueryRow("SELECT state,failures,window_start,blocked_until,open_count FROM provider_state WHERE provider=?", provider).Scan(&s.mode, &s.failures, &window, &blocked, &s.opens)
		s.blocked = time.Unix(blocked, 0)
		s.window = time.Unix(window, 0)
		c.states[provider] = s
	}
	return s
}

func (c *Coordinator) persist(p string, s *state) {
	_, e := c.db.Exec("INSERT INTO provider_state VALUES(?,?,?,?,?,?) ON CONFLICT(provider) DO UPDATE SET state=excluded.state,failures=excluded.failures,window_start=excluded.window_start,blocked_until=excluded.blocked_until,open_count=excluded.open_count", p, s.mode, s.failures, s.window.Unix(), s.blocked.Unix(), s.opens)
	if e != nil {
		slog.Error("persist provider state", "provider", p, "error", e)
	}
}
