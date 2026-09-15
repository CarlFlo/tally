package jobs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/config"
	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/providers"
	"github.com/CarlFlo/tally/internal/settings"
)

func TestWebhookUsesSavedSettingsAndStopsWhenDisabled(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, e := database.Open(ctx, dir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	control, e := providers.New(ctx, db, dir, 2, 0)
	if e != nil {
		t.Fatal(e)
	}
	defer control.Close()
	deliveries := make(chan string, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Key string `json:"key"`
		}
		if r.Method != http.MethodPost || json.NewDecoder(r.Body).Decode(&body) != nil {
			t.Error("invalid webhook payload")
		}
		deliveries <- body.Key
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	s := New(ctx, db, config.Config{}, nil, control, nil)
	if e = s.Start(); e != nil {
		t.Fatal(e)
	}
	defer s.Stop(ctx)
	store := settings.Store{DB: db}
	rev, e := store.Save(ctx, "notifications", settings.Webhook{Enabled: true, URL: server.URL}, 1)
	if e != nil {
		t.Fatal(e)
	}
	s.Alert("fixture:first", "error", "Local fixture only")
	select {
	case key := <-deliveries:
		if key != "fixture:first" {
			t.Fatal("wrong notification", key)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("saved webhook was not used")
	}
	if _, e = store.Save(ctx, "notifications", settings.Webhook{Enabled: false, URL: server.URL}, rev); e != nil {
		t.Fatal(e)
	}
	s.Alert("fixture:disabled", "error", "Must remain in app")
	select {
	case <-deliveries:
		t.Fatal("disabled webhook was called")
	case <-time.After(700 * time.Millisecond):
	}
	var count int
	if e = db.QueryRow("SELECT COUNT(*) FROM alerts WHERE key='fixture:disabled'").Scan(&count); e != nil || count != 1 {
		t.Fatal("disabling webhook removed in-app alerts", e)
	}
}
