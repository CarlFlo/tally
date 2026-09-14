package torrent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestOldClientCredentialsRequireAPIKey(t *testing.T) {
	ctx := context.Background()
	store := clientStore(t)
	_, e := store.DB.Exec(`UPDATE download_client_settings SET adapter='qbittorrent',fields=?,revision=1 WHERE id=1`, `{"url":"http://legacy.example","username":"old-user","password":"old-secret"}`)
	if e != nil {
		t.Fatal(e)
	}
	saved, e := store.Load(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if saved.Configured() {
		t.Fatal("old password configuration is ready to send")
	}
	if _, e := store.Current(ctx); e == nil || !strings.Contains(e.Error(), "API key") {
		t.Fatal("missing API key was not explained", e)
	}
	raw, _ := json.Marshal(saved.View())
	if strings.Contains(string(raw), "old-secret") || strings.Contains(string(raw), "old-user") || saved.View().SecretsConfigured["api_key"] {
		t.Fatal("old credentials exposed or treated as API key")
	}
	if saved.View().Fields["url"] != "http://legacy.example" {
		t.Fatal("existing address lost")
	}
	_, e = store.Save(ctx, ClientUpdate{Adapter: saved.Adapter, Revision: saved.Revision, Fields: map[string]string{"url": saved.Fields["url"], "api_key": testAPIKey}})
	if e != nil {
		t.Fatal(e)
	}
	saved, e = store.Load(ctx)
	if e != nil || !saved.Configured() || len(saved.Fields) != 2 {
		t.Fatal("API key save did not replace legacy fields", e)
	}
}
