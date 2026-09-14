package providers

import (
	"context"
	"fmt"
	"time"
)

// acquire reserves a provider slot and, when needed, its single recovery probe.
func (c *Coordinator) acquire(ctx context.Context, r Request) (*state, func(), error) {
	s := c.State(r.Provider)
	select {
	case s.sem <- struct{}{}:
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
	s.mu.Lock()
	if time.Now().Before(s.blocked) {
		s.mu.Unlock()
		<-s.sem
		c.record(r, 0, "provider backoff", 0, 0)
		return nil, nil, fmt.Errorf("%s is cooling down; try again later", r.Provider)
	}
	if s.probe {
		s.mu.Unlock()
		<-s.sem
		return nil, nil, fmt.Errorf("%s recovery probe in progress", r.Provider)
	}
	if s.mode == "open" || s.mode == "half-open" {
		s.probe = true
		s.mode = "half-open"
	}
	isProbe := s.probe
	s.mu.Unlock()
	release := func() {
		if isProbe {
			s.mu.Lock()
			s.probe = false
			if s.mode == "half-open" {
				s.mode = "open"
				s.blocked = time.Now().Add(time.Minute)
				c.persist(r.Provider, s)
			}
			s.mu.Unlock()
		}
		<-s.sem
	}
	return s, release, nil
}
