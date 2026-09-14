package providers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSharedRequestCancelsAfterLastWaiter(t *testing.T) {
	c := control(t)
	started, stopped := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(stopped)
	}))
	defer server.Close()
	req := Request{Provider: "test", URL: server.URL, TTL: time.Hour}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := c.Do(ctx, req); done <- err }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request did not start")
	}
	second, cancelSecond := context.WithCancel(context.Background())
	defer cancelSecond()
	other := make(chan error, 1)
	go func() { _, err := c.Do(second, req); other <- err }()
	deadline := time.Now().Add(time.Second)
	for {
		c.flightMu.Lock()
		joined := false
		for _, f := range c.flights {
			joined = f.waiters == 2
		}
		c.flightMu.Unlock()
		if joined {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("second waiter did not join")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("first caller: %v", err)
	}
	select {
	case <-stopped:
		t.Fatal("cancelled work still needed by second caller")
	default:
	}
	cancelSecond()
	if err := <-other; !errors.Is(err, context.Canceled) {
		t.Fatalf("second caller: %v", err)
	}
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("orphaned downstream request")
	}
}

func TestSharedRequestBoundsOutstandingWork(t *testing.T) {
	c := control(t)
	c.flightActive = 64
	_, err := c.Do(context.Background(), Request{Provider: "test", URL: "http://127.0.0.1:1", TTL: time.Hour})
	if err == nil || err.Error() != "provider request queue is full; try again later" {
		t.Fatalf("expected admission rejection, got %v", err)
	}
	if len(c.flights) != 0 {
		t.Fatal("rejected request started work")
	}
}
