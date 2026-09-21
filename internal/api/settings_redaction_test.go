package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/CarlFlo/tally/internal/settings"
	"github.com/CarlFlo/tally/internal/torrent"
)

func TestSettingsSecretRedactionAndPreservation(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	ctx := context.Background()
	if _, err := s.Clients.Save(ctx, torrent.ClientUpdate{
		Adapter: "qbittorrent",
		Fields: map[string]string{"url": "http://client.invalid", "api_key": fixtureClientKey},
	}); err != nil {
		t.Fatal(err)
	}

	searchValue := "fixture-indexer-credential"
	response := request(t, h, "PUT", "/api/settings/search", map[string]any{
		"revision": 1,
		"data": settings.Search{BaseURL: "http://indexer.invalid", APIKey: searchValue, Enabled: true},
	})
	expect(t, response, http.StatusOK)

	notificationValue := "http://notify.invalid/hook"
	response = request(t, h, "PUT", "/api/settings/notifications", map[string]any{
		"revision": 1,
		"data": settings.Webhook{Enabled: true, URL: notificationValue},
	})
	expect(t, response, http.StatusOK)

	for _, path := range []string{"/api/downloader", "/api/settings/search", "/api/settings/notifications", "/api/settings"} {
		response = request(t, h, "GET", path, nil)
		expect(t, response, http.StatusOK)
		body := response.Body.String()
		for _, value := range []string{fixtureClientKey, searchValue, notificationValue} {
			if strings.Contains(body, value) {
				t.Fatalf("%s exposed stored connection data", path)
			}
		}
	}

	searchResponse := request(t, h, "GET", "/api/settings/search", nil)
	var searchEnvelope map[string]json.RawMessage
	if err := json.Unmarshal(searchResponse.Body.Bytes(), &searchEnvelope); err != nil { t.Fatal(err) }
	var redactedSearch settings.Search
	var searchRevision int64
	var searchConfigured map[string]bool
	if err := json.Unmarshal(searchEnvelope["data"], &redactedSearch); err != nil { t.Fatal(err) }
	if err := json.Unmarshal(searchEnvelope["revision"], &searchRevision); err != nil { t.Fatal(err) }
	if err := json.Unmarshal(searchEnvelope["secrets_configured"], &searchConfigured); err != nil { t.Fatal(err) }
	if redactedSearch.APIKey != "" || !searchConfigured["api_key"] {
		t.Fatal("Jackett credential was not redacted with configured metadata")
	}
	redactedSearch.Enabled = false
	response = request(t, h, "PUT", "/api/settings/search", map[string]any{"revision": searchRevision, "data": redactedSearch})
	expect(t, response, http.StatusOK)
	var storedSearch settings.Search
	if _, err := s.settingsStore().Load(ctx, "search", &storedSearch); err != nil { t.Fatal(err) }
	if storedSearch.APIKey != searchValue {
		t.Fatal("blank redacted Jackett credential replaced the stored value")
	}

	notificationResponse := request(t, h, "GET", "/api/settings/notifications", nil)
	var notificationEnvelope map[string]json.RawMessage
	if err := json.Unmarshal(notificationResponse.Body.Bytes(), &notificationEnvelope); err != nil { t.Fatal(err) }
	var redactedNotification settings.Webhook
	var notificationRevision int64
	var notificationConfigured map[string]bool
	if err := json.Unmarshal(notificationEnvelope["data"], &redactedNotification); err != nil { t.Fatal(err) }
	if err := json.Unmarshal(notificationEnvelope["revision"], &notificationRevision); err != nil { t.Fatal(err) }
	if err := json.Unmarshal(notificationEnvelope["secrets_configured"], &notificationConfigured); err != nil { t.Fatal(err) }
	if redactedNotification.URL != "" || !notificationConfigured["url"] {
		t.Fatal("notification endpoint was not redacted with configured metadata")
	}
	redactedNotification.Prefix = "Updated"
	response = request(t, h, "PUT", "/api/settings/notifications", map[string]any{"revision": notificationRevision, "data": redactedNotification})
	expect(t, response, http.StatusOK)
	var storedNotification settings.Webhook
	if _, err := s.settingsStore().Load(ctx, "notifications", &storedNotification); err != nil { t.Fatal(err) }
	if storedNotification.URL != notificationValue {
		t.Fatal("blank redacted notification endpoint replaced the stored value")
	}
}
