package live

import (
	"context"
	"sync"
)

// Hub broadcasts state-change hints. Events contain no application data.
type Hub struct {
	mu          sync.Mutex
	subscribers map[chan struct{}]struct{}
}

func New() *Hub {
	return &Hub{subscribers: make(map[chan struct{}]struct{})}
}

func (h *Hub) Subscribe(ctx context.Context) <-chan struct{} {
	updates := make(chan struct{}, 1)
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

func (h *Hub) Publish() {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for subscriber := range h.subscribers {
		select {
		case subscriber <- struct{}{}:
		default:
		}
	}
}
