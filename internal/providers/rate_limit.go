package providers

import (
	"context"
	"fmt"
	"time"
)

func (c *Coordinator) waitTurn(ctx context.Context, r Request, s *state, attempt int) error {
	s.mu.Lock()
	if time.Now().Before(s.blocked) {
		s.mu.Unlock()
		return c.backoffError(r, attempt)
	}
	wait := max(time.Until(s.next), 0)
	s.next = time.Now().Add(wait + 500*time.Millisecond)
	s.mu.Unlock()
	if err := sleep(ctx, wait); err != nil {
		return err
	}
	s.mu.Lock()
	blocked := time.Now().Before(s.blocked)
	s.mu.Unlock()
	if blocked {
		return c.backoffError(r, attempt)
	}
	return c.ctx.Err()
}

func (c *Coordinator) backoffError(r Request, attempt int) error {
	c.record(r, 0, "provider backoff", attempt, 0)
	return fmt.Errorf("%s is cooling down; try again later", r.Provider)
}
