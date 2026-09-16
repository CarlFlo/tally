package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const swedishLocale = `{
	"_meta":{"locale":"sv","name":"Svenska","direction":"ltr","catalogVersion":1},
	"common":{"save":"Spara"}
}`

func TestLocaleEndpointsArePublic(t *testing.T) {
	_, h, _ := testServer(t, "disabled")

	index := request(t, h, "GET", "/api/locales", nil)
	expect(t, index, http.StatusOK)
	var listed struct {
		Revision uint64 `json:"revision"`
		Locales  []struct {
			Locale string `json:"locale"`
			Valid  bool   `json:"valid"`
		} `json:"locales"`
	}
	if err := json.Unmarshal(index.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if listed.Revision == 0 || len(listed.Locales) == 0 || listed.Locales[0].Locale != "en" || !listed.Locales[0].Valid {
		t.Fatalf("unexpected locale index: %+v", listed)
	}

	catalog := request(t, h, "GET", "/api/locales/en", nil)
	expect(t, catalog, http.StatusOK)
	var body struct {
		Meta struct {
			Locale         string `json:"locale"`
			CatalogVersion int    `json:"catalogVersion"`
		} `json:"meta"`
		Messages map[string]any `json:"messages"`
	}
	if err := json.Unmarshal(catalog.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Meta.Locale != "en" || body.Meta.CatalogVersion != 5 || body.Messages["common"] == nil {
		t.Fatalf("unexpected English catalog: %+v", body.Meta)
	}

	missing := request(t, h, "GET", "/api/locales/xx", nil)
	expect(t, missing, http.StatusConflict)
	if value(t, missing, "code") != "locale_unavailable" {
		t.Fatal("missing locale did not return a stable error code")
	}
}

func TestProfileLocalePersistsAcrossInvalidationAndRecovery(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	admin := &http.Cookie{Name: "tally_profile", Value: "profile-admin"}
	path := filepath.Join(s.Config.DataDir, "locales", "sv.json")
	if err := os.WriteFile(path, []byte(swedishLocale), 0o644); err != nil {
		t.Fatal(err)
	}
	waitForLocale(t, s, "sv", true)

	created := request(t, h, "POST", "/api/profiles", map[string]any{
		"name": "Swedish profile", "locale": "sv",
	}, admin)
	expect(t, created, http.StatusCreated)
	if value(t, created, "locale") != "sv" {
		t.Fatal("created profile did not retain requested locale")
	}

	invalid := request(t, h, "PATCH", "/api/profile", map[string]any{
		"name": "My profile", "locale": "xx",
	}, admin)
	expect(t, invalid, http.StatusBadRequest)
	if value(t, invalid, "code") != "profile_locale_invalid" {
		t.Fatal("invalid locale did not return a stable error code")
	}

	updated := request(t, h, "PATCH", "/api/profile", map[string]any{
		"name": "My profile", "locale": "sv",
	}, admin)
	expect(t, updated, http.StatusOK)

	if err := os.WriteFile(path, []byte(`{"_meta":{"locale":"sv","name":"Svenska"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	waitForLocale(t, s, "sv", false)

	preserved := request(t, h, "PATCH", "/api/profile", map[string]any{
		"name": "Renamed during fallback", "locale": "sv",
	}, admin)
	expect(t, preserved, http.StatusOK)
	var storedName, storedLocale string
	if err := s.DB.QueryRow("SELECT display_name,locale FROM profiles WHERE id='profile-admin'").Scan(&storedName, &storedLocale); err != nil {
		t.Fatal(err)
	}
	if storedName != "Renamed during fallback" || storedLocale != "sv" {
		t.Fatalf("profile update changed unavailable locale preference: name=%q locale=%q", storedName, storedLocale)
	}

	boot := request(t, h, "GET", "/api/bootstrap", nil, admin)
	expect(t, boot, http.StatusOK)
	var bootstrap struct {
		Profile map[string]any `json:"profile"`
	}
	if err := json.Unmarshal(boot.Body.Bytes(), &bootstrap); err != nil {
		t.Fatal(err)
	}
	if bootstrap.Profile["locale"] != "sv" {
		t.Fatalf("stored locale was overwritten during fallback: %#v", bootstrap.Profile["locale"])
	}
	if resolved := s.Locales.Resolve("sv"); resolved != "en" {
		t.Fatalf("invalid active locale resolved to %q, want en", resolved)
	}

	if err := os.WriteFile(path, []byte(swedishLocale), 0o644); err != nil {
		t.Fatal(err)
	}
	waitForLocale(t, s, "sv", true)
	if resolved := s.Locales.Resolve("sv"); resolved != "sv" {
		t.Fatalf("repaired locale resolved to %q, want sv", resolved)
	}
}

func TestLocaleFileChangePublishesLiveInvalidation(t *testing.T) {
	s, _, _ := testServer(t, "disabled")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updates := s.Events.Subscribe(ctx, "profile-admin")

	path := filepath.Join(s.Config.DataDir, "locales", "sv.json")
	if err := os.WriteFile(path, []byte(swedishLocale), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case event := <-updates:
		for _, change := range event.Changes {
			if change.Resource == "locales" {
				return
			}
		}
		t.Fatalf("unexpected live event: %+v", event)
	case <-time.After(3 * time.Second):
		t.Fatal("locale file change did not publish a live invalidation")
	}
}

func waitForLocale(t *testing.T, s *Server, locale string, valid bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if s.Locales.Valid(locale) == valid {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("locale %s valid=%v was not observed", locale, valid)
}
