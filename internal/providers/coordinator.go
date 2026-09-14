// Package providers owns outbound HTTP, shared caching, throttling and telemetry.
package providers

import (
	"context"
	"database/sql"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/CarlFlo/mediaManager/internal/database"
)

type Coordinator struct {
	db                   *database.Store
	cache                *sql.DB
	client               *http.Client
	concurrency, retries int
	mu                   sync.Mutex
	states               map[string]*state
	flight               singleflight.Group
	ctx                  context.Context
	cancel               context.CancelFunc
	Alert                func(string, string, string)
	lifecycle            sync.Mutex
	active               sync.WaitGroup
	closing              bool
}

func New(ctx context.Context, db *database.Store, dir string, concurrency, retries int) (*Coordinator, error) {
	cache, e := sql.Open("sqlite", filepath.ToSlash(filepath.Join(dir, "cache.db")))
	if e != nil {
		return nil, e
	}
	cache.SetMaxOpenConns(1)
	if _, e = cache.Exec("PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; CREATE TABLE IF NOT EXISTS responses (key TEXT PRIMARY KEY,body BLOB NOT NULL,etag TEXT NOT NULL,modified TEXT NOT NULL,content_type TEXT NOT NULL,expires INTEGER NOT NULL)"); e != nil {
		cache.Close()
		return nil, e
	}
	ctx, cancel := context.WithCancel(ctx)
	return &Coordinator{db: db, cache: cache, concurrency: concurrency, retries: retries, states: map[string]*state{}, ctx: ctx, cancel: cancel, client: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func (c *Coordinator) Close() error {
	c.lifecycle.Lock()
	c.closing = true
	c.cancel()
	c.lifecycle.Unlock()
	c.active.Wait()
	return c.cache.Close()
}

func (c *Coordinator) begin() bool {
	c.lifecycle.Lock()
	defer c.lifecycle.Unlock()
	if c.closing {
		return false
	}
	c.active.Add(1)
	return true
}
