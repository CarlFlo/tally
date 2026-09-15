package providers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/CarlFlo/tally/internal/config"
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
	return c.shared(ctx, key+strconv.FormatBool(r.Force), r)
}
