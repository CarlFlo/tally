package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/CarlFlo/tally/internal/torrent"
)

func TestSettingsSecretRedaction(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	if _, e := s.Clients.Save(context.Background(), torrent.ClientUpdate{Adapter: "qbittorrent", Fields: map[string]string{"url": "http://client.invalid", "api_key": fixtureClientKey}}); e != nil {
		t.Fatal(e)
	}
	admin := &http.Cookie{Name: "tally_profile", Value: "profile-admin"}
	w := request(t, h, "GET", "/api/settings", nil, admin)
	expect(t, w, 200)
	if strings.Contains(w.Body.String(), fixtureClientKey) {
		t.Fatal("secret exposed")
	}
}
