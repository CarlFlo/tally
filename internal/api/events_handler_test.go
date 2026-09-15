package api

import (
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
