package live

import (
	"context"
	"testing"
	"time"
)

func TestHubCoalescesAndDeliversChanges(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hub := New()
	updates := hub.Subscribe(ctx, "user0")
	hub.Publish("", "jobs", "jobs", "statistics")
	hub.Publish("", "calendar")
	select {
	case event := <-updates:
		if event.Version != 1 || len(event.Changes) != 2 {
			t.Fatalf("unexpected event: %#v", event)
		}
		if event.Changes[0].Resource != "jobs" || event.Changes[1].Resource != "statistics" {
			t.Fatalf("unexpected changes: %#v", event.Changes)
		}
	case <-time.After(time.Second):
		t.Fatal("change was not delivered")
	}
	select {
	case <-updates:
		t.Fatal("unread changes were not coalesced")
	default:
	}
}
