package providers

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

func RetryAfter(raw string, now time.Time) time.Duration {
	if n, e := strconv.ParseInt(raw, 10, 64); e == nil && n >= 0 {
		if n > 86400*7 {
			n = 86400 * 7
		}
		return time.Duration(n) * time.Second
	}
	if t, e := http.ParseTime(raw); e == nil && t.After(now) {
		return t.Sub(now)
	}
	return 0
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
