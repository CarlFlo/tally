package providers

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"time"
)

func (c *Coordinator) perform(ctx context.Context, r Request) (Response, error) {
	cached := c.loadCache(ctx, r)
	if !r.Force && cached.expires > time.Now().Unix() {
		c.record(r, 0, "fresh cache", 0, 0)
		return Response{cached.body, 200, http.Header{"Content-Type": []string{cached.contentType}}}, nil
	}
	s, release, err := c.acquire(ctx, r)
	if err != nil {
		return Response{}, err
	}
	defer release()
	limit := c.retries
	if r.NoRetry {
		limit = 0
	}
	for attempt := 0; attempt <= limit; attempt++ {
		if err := c.waitTurn(ctx, r, s, attempt); err != nil {
			return Response{}, err
		}
		response, elapsed, e := c.requestOnce(ctx, r, cached)
		var invalid invalidRequestError
		if errors.As(e, &invalid) {
			return Response{}, e
		}
		status, header := response.Status, response.Header
		reason := "request"
		if status == 304 && cached.body != nil && e == nil {
			reason = "conditional cache hit"
			response.Body = cached.body
		}
		c.record(r, status, reason, attempt, elapsed)
		if e == nil && ((status >= 200 && status < 300) || (status == 304 && cached.body != nil)) {
			c.success(r.Provider, s)
			return c.cacheSuccess(ctx, r, response, cached), nil
		}
		if ctx.Err() != nil {
			return Response{}, ctx.Err()
		}
		retryable := e != nil || status == 408 || status == 429 || status >= 500
		if retryable {
			c.failure(r.Provider, s)
		}
		if status == 429 || status == 503 {
			delay := RetryAfter(header.Get("Retry-After"), time.Now())
			if delay > 0 {
				s.mu.Lock()
				if until := time.Now().Add(delay); until.After(s.blocked) {
					s.blocked = until
				}
				c.persist(r.Provider, s)
				s.mu.Unlock()
				return Response{}, fmt.Errorf("%s asked to pause requests; try again later", r.Provider)
			}
		}
		if !retryable || attempt == limit {
			if e != nil {
				return Response{}, fmt.Errorf("%s request failed (network, timeout, or response limit)", r.Provider)
			}
			return Response{}, fmt.Errorf("%s returned HTTP %d", r.Provider, status)
		}
		delay := time.Duration(1<<attempt)*time.Second + time.Duration(rand.IntN(500))*time.Millisecond
		if e = sleep(ctx, delay); e != nil {
			return Response{}, e
		}
	}
	return Response{}, errors.New("provider retry limit reached")
}
