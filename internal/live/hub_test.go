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
	updates := hub.Subscribe(ctx)
	hub.Publish()
	hub.Publish()
	select {
	case <-updates:
	case <-time.After(time.Second):
		t.Fatal("change was not delivered")
	}
	select {
	case <-updates:
		t.Fatal("unread changes were not coalesced")
	default:
	}
}
