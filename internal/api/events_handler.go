package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) events(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return apiError{http.StatusNotImplemented, "live updates are unavailable"}
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()
	updates := s.Events.Subscribe(r.Context())
	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return nil
		case event := <-updates:
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-heartbeat.C:
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}
