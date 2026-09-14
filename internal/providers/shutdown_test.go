package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestShutdownCancelsOutstandingCalls(t *testing.T) {
	c := control(t)
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done() }))
	defer server.Close()
	result := make(chan error, 1)
	go func() {
		_, e := c.Do(context.Background(), Request{Provider: "shutdown", URL: server.URL, TTL: time.Hour})
		result <- e
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("call did not start")
	}
	closed := make(chan struct{})
	go func() { c.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown did not cancel and drain calls")
	}
	if e := <-result; e == nil {
		t.Fatal("cancelled request succeeded")
	}
	if _, e := c.Do(context.Background(), Request{Provider: "shutdown", URL: server.URL}); e == nil {
		t.Fatal("new request accepted after shutdown")
	}
}
