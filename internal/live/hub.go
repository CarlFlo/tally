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

// Hub broadcasts small state-change hints. Events never contain application data.
type Hub struct {
	mu          sync.Mutex
	subscribers map[chan Event]struct{}
}

func New() *Hub {
	return &Hub{subscribers: make(map[chan Event]struct{})}
}

func (h *Hub) Subscribe(ctx context.Context) <-chan Event {
	updates := make(chan Event, 1)
	h.mu.Lock()
	h.subscribers[updates] = struct{}{}
	h.mu.Unlock()
	go func() {
		<-ctx.Done()
		h.mu.Lock()
		delete(h.subscribers, updates)
		h.mu.Unlock()
	}()
	return updates
}

func (h *Hub) Publish(resources ...string) {
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
		select {
		case subscriber <- event:
		default:
		}
	}
}
