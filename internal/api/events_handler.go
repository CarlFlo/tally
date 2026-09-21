package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) events(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if session.Profile == "" && s.Auth != nil {
		resolved, err := s.Auth.Resolve(r)
		if err == nil {
			session = resolved
		} else if !errors.Is(err, auth.ErrSignInRequired) && !errors.Is(err, auth.ErrSessionExpired) {
			return err
		}
	}
	publicOnly := session.Profile == "" || session.Restricted
	subscriptionProfile := session.Profile
	if publicOnly {
		subscriptionProfile = ""
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		return apiError{http.StatusNotImplemented, "live updates are unavailable"}
	}
	// SSE connections are intentionally long-lived. Clear the server-wide write
	// deadline for this request while retaining heartbeats and request-context
	// cancellation.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
	updates := s.Events.Subscribe(r.Context(), subscriptionProfile)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()
	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-s.Events.Done():
			return nil
		case <-r.Context().Done():
			return nil
		case event := <-updates:
			var data []byte
			var err error
			if publicOnly {
				hasLocales := false
				for _, change := range event.Changes {
					if change.Resource == "locales" {
						hasLocales = true
						break
					}
				}
				if !hasLocales {
					continue
				}
				data, err = json.Marshal(map[string]any{
					"version": event.Version,
					"changes": []map[string]string{{"resource": "locales"}},
				})
			} else {
				data, err = json.Marshal(event)
			}
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
