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
	updates := hub.Subscribe(ctx, "profileA")
	hub.Publish("", "jobs", "jobs", "statistics")
	hub.Publish("", "calendar")
	select {
	case event := <-updates:
		if event.Version != 1 || len(event.Changes) != 3 {
			t.Fatalf("unexpected event: %#v", event)
		}
		if event.Changes[0].Resource != "jobs" || event.Changes[1].Resource != "statistics" || event.Changes[2].Resource != "calendar" {
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

func TestHubScopesProfileEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hub := New()
	profileA := hub.Subscribe(ctx, "profileA")
	profileB := hub.Subscribe(ctx, "profileB")

	hub.Publish("profileB", "calendar")

	select {
	case <-profileA:
		t.Fatal("profile event leaked to another profile")
	default:
	}
	select {
	case event := <-profileB:
		if len(event.Changes) != 1 || event.Changes[0].Resource != "calendar" {
			t.Fatalf("unexpected event: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("profile event was not delivered")
	}
}


func TestHubCloseSignalsShutdownAndStopsPublishing(t *testing.T) {
	hub := New()
	hub.Close()
	hub.Close()

	select {
	case <-hub.Done():
	case <-time.After(time.Second):
		t.Fatal("hub shutdown signal was not closed")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updates := hub.Subscribe(ctx, "profileA")
	hub.Publish("", "jobs")
	select {
	case <-updates:
		t.Fatal("closed hub published an update")
	default:
	}
}
