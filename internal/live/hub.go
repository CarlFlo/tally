package live

import (
	"context"
	"sync"
)

type Change struct {
	Resource string `json:"resource"`
	ID       string `json:"id,omitempty"`
}

type Event struct {
	Version int      `json:"version"`
	Changes []Change `json:"changes"`
}

type subscription struct {
	profile string
	updates chan Event
}

// Hub broadcasts small state-change hints. Events never contain application data.
type Hub struct {
	mu          sync.Mutex
	subscribers map[*subscription]struct{}
}

func New() *Hub {
	return &Hub{subscribers: make(map[*subscription]struct{})}
}

func (h *Hub) Subscribe(ctx context.Context, profile string) <-chan Event {
	sub := &subscription{profile: profile, updates: make(chan Event, 1)}
	h.mu.Lock()
	h.subscribers[sub] = struct{}{}
	h.mu.Unlock()
	go func() {
		<-ctx.Done()
		h.mu.Lock()
		delete(h.subscribers, sub)
		h.mu.Unlock()
	}()
	return sub.updates
}

func event(resources ...string) Event {
	seen := make(map[string]struct{}, len(resources))
	changes := make([]Change, 0, len(resources))
	for _, resource := range resources {
		if resource == "" {
			continue
		}
		if _, ok := seen[resource]; ok {
			continue
		}
		seen[resource] = struct{}{}
		changes = append(changes, Change{Resource: resource})
	}
	return Event{Version: 1, Changes: changes}
}

func merge(a, b Event) Event {
	seen := make(map[string]struct{}, len(a.Changes)+len(b.Changes))
	changes := make([]Change, 0, len(a.Changes)+len(b.Changes))
	for _, source := range [][]Change{a.Changes, b.Changes} {
		for _, change := range source {
			key := change.Resource + "\x00" + change.ID
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			changes = append(changes, change)
		}
	}
	return Event{Version: 1, Changes: changes}
}

// Publish sends a global event when profile is empty, otherwise only to
// subscribers for that profile. If a subscriber already has an unread event,
// new resource hints are merged into it rather than dropped.
func (h *Hub) Publish(profile string, resources ...string) {
	if h == nil || len(resources) == 0 {
		return
	}
	next := event(resources...)
	if len(next.Changes) == 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for subscriber := range h.subscribers {
		if profile != "" && subscriber.profile != profile {
			continue
		}
		select {
		case subscriber.updates <- next:
		default:
			var pending Event
			select {
			case pending = <-subscriber.updates:
			default:
			}
			combined := merge(pending, next)
			select {
			case subscriber.updates <- combined:
			default:
			}
		}
	}
}
