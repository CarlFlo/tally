package api

import (
	"context"
	"strings"
	"testing"

	"github.com/CarlFlo/tally/internal/torrent"
)

func TestSettingsSecretRedaction(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	if _, e := s.Clients.Save(context.Background(), torrent.ClientUpdate{Adapter: "qbittorrent", Fields: map[string]string{"url": "http://client.invalid", "api_key": fixtureClientKey}}); e != nil {
		t.Fatal(e)
	}
	s.Config.OIDCSecret = "DO-NOT-EXPOSE"
	w := request(t, h, "GET", "/api/settings", nil)
	expect(t, w, 200)
	if strings.Contains(w.Body.String(), "DO-NOT-EXPOSE") || strings.Contains(w.Body.String(), fixtureClientKey) {
		t.Fatal("secret exposed")
	}
}
