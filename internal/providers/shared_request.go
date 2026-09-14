package providers

import (
	"context"
	"errors"
	"time"
)

type sharedRequest struct {
	done     chan struct{}
	cancel   context.CancelFunc
	waiters  int
	response Response
	err      error
}

// Keep shared work alive while callers need it, cancel when the last leaves.
// A bound on distinct flights also bounds goroutines waiting for provider slots.
func (c *Coordinator) shared(ctx context.Context, key string, r Request) (Response, error) {
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}
	c.flightMu.Lock()
	if c.flights == nil {
		c.flights = make(map[string]*sharedRequest)
	}
	f, exists := c.flights[key]
	if !exists {
		if c.flightActive >= 64 {
			c.flightMu.Unlock()
			return Response{}, errors.New("provider request queue is full; try again later")
		}
		if !c.begin() {
			c.flightMu.Unlock()
			return Response{}, context.Canceled
		}
		work, cancel := context.WithTimeout(c.ctx, 90*time.Second)
		f = &sharedRequest{done: make(chan struct{}), cancel: cancel}
		c.flights[key] = f
		c.flightActive++
		go func() {
			defer c.active.Done()
			defer cancel()
			f.response, f.err = c.perform(work, r)
			c.flightMu.Lock()
			c.flightActive--
			if c.flights[key] == f {
				delete(c.flights, key)
			}
			close(f.done)
			c.flightMu.Unlock()
		}()
	}
	f.waiters++
	c.flightMu.Unlock()
	defer func() {
		c.flightMu.Lock()
		defer c.flightMu.Unlock()
		f.waiters--
		if f.waiters == 0 {
			f.cancel()
			if c.flights[key] == f {
				delete(c.flights, key)
			}
		}
	}()
	select {
	case <-ctx.Done():
		return Response{}, ctx.Err()
	case <-f.done:
		if err := ctx.Err(); err != nil {
			return Response{}, err
		}
		if exists {
			c.record(r, 0, "duplicate/coalesced", 0, 0)
		}
		return f.response, f.err
	}
}
