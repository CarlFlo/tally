package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCacheCoalescingConditionalRequests(t *testing.T) {
	c := control(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		time.Sleep(30 * time.Millisecond)
		if r.Header.Get("If-None-Match") == "v1" {
			w.WriteHeader(304)
			return
		}
		w.Header().Set("ETag", "v1")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	req := Request{Provider: "test", URL: server.URL, TTL: time.Hour}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := c.Do(context.Background(), req)
			if e != nil || !strings.Contains(string(r.Body), "true") {
				t.Errorf("request failed: %v", e)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("coalescing made %d calls", calls.Load())
	}
	if _, e := c.Do(context.Background(), req); e != nil {
		t.Fatal(e)
	}
	if calls.Load() != 1 {
		t.Fatal("fresh cache missed")
	}
	req.Force = true
	r, e := c.Do(context.Background(), req)
	if e != nil || !strings.Contains(string(r.Body), "true") || calls.Load() != 2 {
		t.Fatalf("conditional cache failed: %v", e)
	}
}
