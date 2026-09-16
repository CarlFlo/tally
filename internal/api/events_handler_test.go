package api

import (
	"context"
	"strings"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/live"
)

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
	response := httptest.NewRecorder()
	done := make(chan error, 1)

	go func() {
		done <- s.events(response, request, auth.Session{})
	}()

	deadline := time.Now().Add(time.Second)
	for !strings.Contains(response.Body.String(), ": connected") && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(10 * time.Millisecond)
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

	body := response.Body.String()
	if strings.Contains(body, "\"jobs\"") || strings.Contains(body, "\"settings\"") {
		t.Fatalf("anonymous SSE leaked non-localization resources: %s", body)
	}
	if !strings.Contains(body, "\"locales\"") {
		t.Fatalf("anonymous SSE did not expose locale invalidation: %s", body)
	}
}
