package providers

import (
	"time"
)

func (c *Coordinator) success(p string, s *state) {
	s.mu.Lock()
	recovered := s.mode == "half-open" || s.mode == "open"
	// An earlier concurrent request must not erase another request's backoff.
	if !s.probe && time.Now().Before(s.blocked) {
		s.mu.Unlock()
		return
	}
	s.mode = "healthy"
	s.failures = 0
	s.opens = 0
	s.blocked = time.Time{}
	c.persist(p, s)
	s.mu.Unlock()
	if recovered && c.Alert != nil {
		c.Alert("provider:"+p, "info", p+" connection recovered")
	}
}

func (c *Coordinator) failure(p string, s *state) {
	s.mu.Lock()
	if time.Since(s.window) > 10*time.Minute {
		s.window = time.Now()
		s.failures = 0
	}
	s.failures++
	opened := s.failures >= 5 || s.mode == "half-open"
	s.mode = "degraded"
	if opened {
		s.mode = "open"
		s.opens++
		n := s.opens - 1
		if n > 4 {
			n = 4
		}
		delay := 30 * time.Minute * time.Duration(1<<n)
		if delay > 6*time.Hour {
			delay = 6 * time.Hour
		}
		s.blocked = time.Now().Add(delay)
	}
	c.persist(p, s)
	s.mu.Unlock()
	if opened && c.Alert != nil {
		c.Alert("provider:"+p, "error", p+" requests paused after repeated failures")
	}
}
