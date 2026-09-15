package torrent

import (
	"context"
	"testing"

	"github.com/CarlFlo/tally/internal/database"
)

func TestClientSettingsPersistAcrossDatabaseReopen(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, e := database.Open(ctx, dir)
	if e != nil {
		t.Fatal(e)
	}
	store := &ClientStore{DB: db}
	_, e = store.Save(ctx, ClientUpdate{Adapter: "qbittorrent", Fields: map[string]string{"url": "http://client.example", "api_key": testAPIKey}})
	db.Close()
	if e != nil {
		t.Fatal(e)
	}
	db, e = database.Open(ctx, dir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	saved, e := (&ClientStore{DB: db}).Load(ctx)
	if e != nil || saved.Adapter != "qbittorrent" || saved.Fields["api_key"] != testAPIKey {
		t.Fatal("settings did not persist across restart", e)
	}
}
