package providers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/CarlFlo/mediaManager/internal/config"
)

func (c *Coordinator) Do(ctx context.Context, r Request) (Response, error) {
	if !c.begin() {
		return Response{}, context.Canceled
	}
	defer c.active.Done()
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(c.ctx, cancel)
	defer stop()
	defer cancel()
	if err := config.ValidateURL(r.URL); err != nil {
		return Response{}, errors.New("invalid configured provider URL")
	}
	if r.Method == "" {
		r.Method = http.MethodGet
	}
	if r.MaxBytes == 0 {
		r.MaxBytes = 8 << 20
	}
	if r.Header == nil {
		r.Header = make(http.Header)
	}
	if e := c.ctx.Err(); e != nil {
		return Response{}, e
	}
	if r.Method != http.MethodGet || r.TTL <= 0 {
		return c.perform(ctx, r)
	}
	key := cacheKey(r)
	var leader atomic.Bool
	ch := c.flight.DoChan(key+strconv.FormatBool(r.Force), func() (any, error) {
		if !c.begin() {
			return nil, context.Canceled
		}
		defer c.active.Done()
		leader.Store(true)
		work, cancel := context.WithTimeout(c.ctx, 90*time.Second)
		defer cancel()
		return c.perform(work, r)
	})
	select {
	case <-ctx.Done():
		return Response{}, ctx.Err()
	case result := <-ch:
		if result.Shared && !leader.Load() {
			c.record(r, 0, "duplicate/coalesced", 0, 0)
		}
		if result.Err != nil {
			return Response{}, result.Err
		}
		return result.Val.(Response), nil
	}
}
