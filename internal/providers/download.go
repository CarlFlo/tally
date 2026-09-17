package providers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"time"

	"github.com/CarlFlo/tally/internal/config"
)

type FileResponse struct {
	Path   string
	Status int
	Header http.Header
	Size   int64
}

// DownloadToTempFile performs a bounded provider GET while streaming the body
// directly to disk. It is intended for large binary responses that should not
// be retained in the Go heap. The caller owns the returned temporary file.
func (c *Coordinator) DownloadToTempFile(ctx context.Context, r Request, dir string) (FileResponse, error) {
	if !c.begin() {
		return FileResponse{}, context.Canceled
	}
	defer c.active.Done()
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(c.ctx, cancel)
	defer stop()
	defer cancel()
	if err := config.ValidateURL(r.URL); err != nil {
		return FileResponse{}, errors.New("invalid configured provider URL")
	}
	if r.Method == "" {
		r.Method = http.MethodGet
	}
	if r.Method != http.MethodGet {
		return FileResponse{}, errors.New("provider file download requires GET")
	}
	if r.MaxBytes == 0 {
		r.MaxBytes = 8 << 20
	}
	if r.Header == nil {
		r.Header = make(http.Header)
	}
	if err := c.ctx.Err(); err != nil {
		return FileResponse{}, err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return FileResponse{}, err
	}
	return c.performFileDownload(ctx, r, dir)
}

func (c *Coordinator) performFileDownload(ctx context.Context, r Request, dir string) (FileResponse, error) {
	s, release, err := c.acquire(ctx, r)
	if err != nil {
		return FileResponse{}, err
	}
	defer release()
	limit := c.retries
	if r.NoRetry {
		limit = 0
	}
	for attempt := 0; attempt <= limit; attempt++ {
		if err := c.waitTurn(ctx, r, s, attempt); err != nil {
			return FileResponse{}, err
		}
		response, elapsed, requestErr := c.downloadOnce(ctx, r, dir)
		var invalid invalidRequestError
		if errors.As(requestErr, &invalid) {
			return FileResponse{}, requestErr
		}
		status, header := response.Status, response.Header
		c.record(r, status, "request", attempt, elapsed)
		if requestErr == nil && status >= 200 && status < 300 {
			c.success(r.Provider, s)
			return response, nil
		}
		if response.Path != "" {
			_ = os.Remove(response.Path)
		}
		if ctx.Err() != nil {
			return FileResponse{}, ctx.Err()
		}
		retryable := requestErr != nil || status == 408 || status == 429 || status >= 500
		if retryable {
			c.failure(r.Provider, s)
		}
		if status == 429 || status == 503 {
			delay := RetryAfter(header.Get("Retry-After"), time.Now())
			if delay > 0 {
				s.mu.Lock()
				if until := time.Now().Add(delay); until.After(s.blocked) {
					s.blocked = until
					c.persist(r.Provider, s)
				}
				s.mu.Unlock()
				return FileResponse{}, fmt.Errorf("%s asked to pause requests; try again later", r.Provider)
			}
		}
		if !retryable || attempt == limit {
			if requestErr != nil {
				return FileResponse{}, fmt.Errorf("%s request failed (network, timeout, or response limit)", r.Provider)
			}
			return FileResponse{}, fmt.Errorf("%s returned HTTP %d", r.Provider, status)
		}
		delay := time.Duration(1<<attempt)*time.Second + time.Duration(rand.IntN(500))*time.Millisecond
		if err = sleep(ctx, delay); err != nil {
			return FileResponse{}, err
		}
	}
	return FileResponse{}, errors.New("provider retry limit reached")
}

func (c *Coordinator) downloadOnce(ctx context.Context, r Request, dir string) (FileResponse, time.Duration, error) {
	call, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(call, r.Method, r.URL, bytes.NewReader(r.Body))
	if err != nil {
		return FileResponse{}, 0, invalidRequestError{}
	}
	req.Header = r.Header.Clone()
	req.Header.Set("User-Agent", "Tally/1.0 (self-hosted TV tracker)")
	started := time.Now()
	res, err := c.client.Do(req)
	out := FileResponse{Header: make(http.Header)}
	if err != nil {
		return out, time.Since(started), err
	}
	defer res.Body.Close()
	out.Status, out.Header = res.StatusCode, res.Header.Clone()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return out, time.Since(started), nil
	}
	if res.ContentLength > r.MaxBytes {
		return out, time.Since(started), errors.New("provider response exceeded size limit")
	}
	file, err := os.CreateTemp(dir, ".provider-*")
	if err != nil {
		return out, time.Since(started), err
	}
	path := file.Name()
	remove := true
	defer func() {
		_ = file.Close()
		if remove {
			_ = os.Remove(path)
		}
	}()
	n, err := io.Copy(file, io.LimitReader(res.Body, r.MaxBytes+1))
	if err == nil && n > r.MaxBytes {
		err = errors.New("provider response exceeded size limit")
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return out, time.Since(started), err
	}
	remove = false
	out.Path, out.Size = path, n
	return out, time.Since(started), nil
}
