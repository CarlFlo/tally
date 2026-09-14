package providers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"time"
)

type invalidRequestError struct{}

func (invalidRequestError) Error() string { return "invalid configured provider request" }

// requestOnce performs one bounded HTTP exchange. Retry policy lives in perform.
func (c *Coordinator) requestOnce(ctx context.Context, r Request, cached cachedResponse) (Response, time.Duration, error) {
	call, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(call, r.Method, r.URL, bytes.NewReader(r.Body))
	if err != nil {
		return Response{}, 0, invalidRequestError{}
	}
	req.Header = r.Header.Clone()
	req.Header.Set("User-Agent", "Tally/1.0 (self-hosted TV tracker)")
	if cached.body != nil {
		if cached.etag != "" {
			req.Header.Set("If-None-Match", cached.etag)
		}
		if cached.modified != "" {
			req.Header.Set("If-Modified-Since", cached.modified)
		}
	}
	started := time.Now()
	res, err := c.client.Do(req)
	out := Response{Header: make(http.Header)}
	if err == nil {
		out.Status, out.Header = res.StatusCode, res.Header.Clone()
		out.Body, err = io.ReadAll(io.LimitReader(res.Body, r.MaxBytes+1))
		res.Body.Close()
		if int64(len(out.Body)) > r.MaxBytes {
			err = errors.New("provider response exceeded size limit")
		}
	}
	return out, time.Since(started), err
}
