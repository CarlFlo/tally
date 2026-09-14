package providers

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

type cachedResponse struct {
	body                        []byte
	etag, modified, contentType string
	expires                     int64
}

func (c *Coordinator) loadCache(ctx context.Context, r Request) cachedResponse {
	var cached cachedResponse
	if r.Method == http.MethodGet && r.TTL > 0 {
		err := c.cache.QueryRowContext(ctx, "SELECT body,etag,modified,content_type,expires FROM responses WHERE key=?", cacheKey(r)).Scan(&cached.body, &cached.etag, &cached.modified, &cached.contentType, &cached.expires)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				slog.Warn("read provider cache", "provider", r.Provider, "error", err)
			}
			return cachedResponse{}
		}
	}
	return cached
}

func (c *Coordinator) cacheSuccess(ctx context.Context, r Request, response Response, cached cachedResponse) Response {
	if r.Method == http.MethodGet && r.TTL > 0 {
		if response.Status != http.StatusNotModified {
			cached.etag = response.Header.Get("ETag")
			cached.modified = response.Header.Get("Last-Modified")
			cached.contentType = response.Header.Get("Content-Type")
		}
		_, err := c.cache.ExecContext(ctx, "INSERT INTO responses VALUES(?,?,?,?,?,?) ON CONFLICT(key) DO UPDATE SET body=excluded.body,etag=excluded.etag,modified=excluded.modified,content_type=excluded.content_type,expires=excluded.expires", cacheKey(r), response.Body, cached.etag, cached.modified, cached.contentType, time.Now().Add(r.TTL).Unix())
		if err != nil {
			slog.Warn("cache provider response", "provider", r.Provider, "error", err)
		}
	}
	if response.Status == http.StatusNotModified {
		response.Status = http.StatusOK
		response.Header.Set("Content-Type", cached.contentType)
	}
	return response
}
