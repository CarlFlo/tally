package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/live"
)


type sseRecorder struct {
	header  http.Header
	mu      sync.Mutex
	body    bytes.Buffer
	flushed chan struct{}
	once    sync.Once
}

func newSSERecorder() *sseRecorder {
	return &sseRecorder{header: make(http.Header), flushed: make(chan struct{})}
}

func (r *sseRecorder) Header() http.Header { return r.header }
func (r *sseRecorder) WriteHeader(int)     {}
func (r *sseRecorder) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.body.Write(p)
}
func (r *sseRecorder) Flush() {
	r.once.Do(func() { close(r.flushed) })
}
func (r *sseRecorder) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.body.String()
}

func TestEventsExitWhenLiveHubCloses(t *testing.T) {
	hub := live.New()
	s := &Server{Events: hub}
	request := httptest.NewRequest("GET", "/api/events", nil)
	response := httptest.NewRecorder()
	done := make(chan error, 1)

	go func() {
		done <- s.events(response, request, auth.Session{Profile: "profile-admin"})
	}()

	hub.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("SSE handler did not exit when live hub closed")
	}
}

func TestAnonymousEventsOnlyExposeLocaleInvalidations(t *testing.T) {
	hub := live.New()
	s := &Server{Events: hub}
	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest("GET", "/api/events", nil).WithContext(ctx)
	response := newSSERecorder()
	done := make(chan error, 1)

	go func() {
		done <- s.events(response, request, auth.Session{})
	}()

	select {
	case <-response.flushed:
	case <-time.After(time.Second):
		t.Fatal("anonymous SSE handler did not connect")
	}

	hub.Publish("", "jobs")
	hub.Publish("", "locales", "settings")
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("anonymous SSE handler did not exit")
	}

	body := response.String()
	if strings.Contains(body, "\"jobs\"") || strings.Contains(body, "\"settings\"") {
		t.Fatalf("anonymous SSE leaked non-localization resources: %s", body)
	}
	if !strings.Contains(body, "\"locales\"") {
		t.Fatalf("anonymous SSE did not expose locale invalidation: %s", body)
	}
}
