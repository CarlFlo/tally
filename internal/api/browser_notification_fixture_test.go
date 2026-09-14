package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func notificationBrowserFixture(t *testing.T, next http.Handler) http.Handler {
	t.Helper()
	var mu sync.Mutex
	messages := []map[string]any{}
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if r.Method != "POST" || json.NewDecoder(r.Body).Decode(&body) != nil {
			http.Error(w, "invalid fixture payload", 400)
			return
		}
		mu.Lock()
		messages = append(messages, body)
		mu.Unlock()
		w.WriteHeader(204)
	}))
	t.Cleanup(endpoint.Close)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /__fixture/notifications", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		jsonResponse(w, 200, map[string]any{"url": endpoint.URL, "messages": messages})
	})
	mux.Handle("/", next)
	return mux
}
