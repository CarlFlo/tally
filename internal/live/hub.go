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

// Publish sends a global event when profile is empty, otherwise only to
// subscribers for that profile. Duplicate resource hints are collapsed.
func (h *Hub) Publish(profile string, resources ...string) {
	if h == nil || len(resources) == 0 {
		return
	}
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
	if len(changes) == 0 {
		return
	}
	event := Event{Version: 1, Changes: changes}
	h.mu.Lock()
	defer h.mu.Unlock()
	for subscriber := range h.subscribers {
		if profile != "" && subscriber.profile != profile {
			continue
		}
		select {
		case subscriber.updates <- event:
		default:
		}
	}
}
