package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestCircuitOpensAndControlledProbeRecovers(t *testing.T) {
	c := control(t)
	var failed atomic.Bool
	failed.Store(true)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if failed.Load() {
			w.WriteHeader(500)
		} else {
			time.Sleep(40 * time.Millisecond)
			w.Write([]byte("ok"))
		}
	}))
	defer server.Close()
	req := Request{Provider: "circuit", URL: server.URL}
	for i := 0; i < 5; i++ {
		_, _ = c.Do(context.Background(), req)
	}
	s := c.State("circuit")
	s.mu.Lock()
	mode := s.mode
	s.mu.Unlock()
	if mode != "open" {
		t.Fatal("circuit did not open")
	}
	_, _ = c.Do(context.Background(), req)
	if calls.Load() != 5 {
		t.Fatal("open circuit made request")
	}
	failed.Store(false)
	s.mu.Lock()
	s.blocked = time.Now().Add(-time.Second)
	s.mu.Unlock()
	if _, e := c.Do(context.Background(), req); e != nil {
		t.Fatal(e)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mode != "healthy" {
		t.Fatal("probe did not recover")
	}
}
