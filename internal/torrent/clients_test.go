package torrent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/CarlFlo/mediaManager/internal/database"
)

func clientStore(t *testing.T) *ClientStore {
	t.Helper()
	db, e := database.Open(context.Background(), t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	return &ClientStore{DB: db}
}

func TestClientSettingsSecretsRevisionsAndDisable(t *testing.T) {
	ctx := context.Background()
	store := clientStore(t)
	in := ClientUpdate{Adapter: "qbittorrent", Fields: map[string]string{"url": "http://client.example/base/", "api_key": testAPIKey}}
	saved, e := store.Save(ctx, in)
	if e != nil {
		t.Fatal(e)
	}
	if saved.Fields["url"] != "http://client.example/base" || saved.Fields["api_key"] != in.Fields["api_key"] {
		t.Fatal("normalization altered the API key or missed the URL")
	}
	raw, _ := json.Marshal(saved.View())
	if strings.Contains(string(raw), testAPIKey) || !saved.View().SecretsConfigured["api_key"] {
		t.Fatal("secret returned or missing configured flag")
	}
	raw, _ = json.Marshal(saved)
	if strings.Contains(string(raw), testAPIKey) {
		t.Fatal("internal settings can accidentally serialize credentials")
	}
	in.Revision = saved.Revision
	delete(in.Fields, "api_key")
	kept, e := store.Save(ctx, in)
	if e != nil || kept.Fields["api_key"] != saved.Fields["api_key"] {
		t.Fatal("saving without a new API key lost the saved secret", e)
	}
	if _, e = store.Save(ctx, in); !errors.Is(e, ErrClientConflict) {
		t.Fatal("stale settings overwrote another edit", e)
	}
	in.Revision = kept.Revision
	in.Fields["url"] = "https://other.example"
	prepared, e := store.Prepare(ctx, in)
	if e != nil || prepared.Fields["api_key"] != saved.Fields["api_key"] {
		t.Fatal("saved credentials were not retained after changing endpoint", e)
	}
	in.Fields["api_key"] = ""
	if _, e := store.Prepare(ctx, in); e == nil {
		t.Fatal("clearing a required API key was accepted")
	}
	persisted, _ := (&ClientStore{DB: store.DB}).Load(ctx)
	if persisted.Fields["url"] != saved.Fields["url"] {
		t.Fatal("preparing a connection test changed saved settings")
	}
	in.Fields["url"] = saved.Fields["url"]
	in.Fields["api_key"] = "qbt_aaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	rotated, e := store.Save(ctx, in)
	if e != nil || rotated.Fields["api_key"] != in.Fields["api_key"] || !rotated.Configured() {
		t.Fatal("API key replacement failed", e)
	}
	disabled, e := store.Save(ctx, ClientUpdate{Revision: rotated.Revision, Fields: map[string]string{}})
	if e != nil || disabled.Adapter != "" || len(disabled.Fields) != 0 {
		t.Fatal("disable retained client details", e)
	}
	if _, e := store.Current(ctx); !errors.Is(e, ErrNoClient) {
		t.Fatal("disabled client is still available", e)
	}
}
