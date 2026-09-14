package torrent

import (
	"context"
	"testing"
)

func TestClientSettingsRejectUnsupportedAndUnsafeInput(t *testing.T) {
	store := clientStore(t)
	for _, in := range []ClientUpdate{
		{Adapter: "unknown"},
		{Adapter: "qbittorrent"},
		{Adapter: "qbittorrent", Fields: map[string]string{"url": "http://host"}},
		{Adapter: "qbittorrent", Fields: map[string]string{"url": "http://host", "username": "admin", "password": "old"}},
		{Adapter: "qbittorrent", Fields: map[string]string{"url": "file:///tmp/client"}},
		{Adapter: "qbittorrent", Fields: map[string]string{"url": "http://user:secret@host"}},
		{Adapter: "qbittorrent", Fields: map[string]string{"url": "http://host?secret=key"}},
		{Adapter: "qbittorrent", Fields: map[string]string{"url": "http://host", "unexpected": "value"}},
		{Adapter: "", Fields: map[string]string{"password": "unused"}},
	} {
		if _, e := store.Save(context.Background(), in); e == nil {
			t.Fatal("invalid settings accepted")
		}
	}
}

func TestClientSettingsRejectMalformedAPIKeys(t *testing.T) {
	store := clientStore(t)
	for _, key := range []string{"", "qbt_", "password", "Bearer " + testAPIKey, " " + testAPIKey, testAPIKey + "\r\nX-Injected: 1", testAPIKey + "\t", "qbt_\u00fcmlaut"} {
		if _, e := store.Save(context.Background(), ClientUpdate{Adapter: "qbittorrent", Fields: map[string]string{"url": "http://client.example", "api_key": key}}); e == nil {
			t.Fatal("invalid API key accepted")
		}
	}
}
