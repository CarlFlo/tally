package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryAfterSurvivesManualRefresh(t *testing.T) {
	c := control(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(429)
	}))
	defer server.Close()
	req := Request{Provider: "limited", URL: server.URL, TTL: time.Hour}
	if _, e := c.Do(context.Background(), req); e == nil {
		t.Fatal("429 succeeded")
	}
	req.Force = true
	if _, e := c.Do(context.Background(), req); e == nil {
		t.Fatal("manual refresh bypassed backoff")
	}
	if calls.Load() != 1 {
		t.Fatal("called during backoff")
	}
	if RetryAfter("120", time.Now()) != 2*time.Minute {
		t.Fatal("seconds parse failed")
	}
	now := time.Now().UTC().Truncate(time.Second)
	if RetryAfter(now.Add(time.Minute).Format(http.TimeFormat), now) != time.Minute {
		t.Fatal("HTTP date parse failed")
	}
}
