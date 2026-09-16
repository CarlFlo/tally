package localization

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestRegistrySeedsEnglishAndFallsBack(t *testing.T) {
	dir := t.TempDir()
	registry, err := New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()

	if !registry.Valid("en") {
		t.Fatal("English locale was not available")
	}
	if registry.Resolve("missing") != "en" {
		t.Fatal("missing locale did not resolve to English")
	}
	if _, err = os.Stat(filepath.Join(dir, "locales", "en.json")); err != nil {
		t.Fatal("English locale was not written to the config directory:", err)
	}
}

func TestRegistryShowsInvalidLocaleAndHotReloadsRepair(t *testing.T) {
	dir := t.TempDir()
	var changes atomic.Int32
	registry, err := New(dir, func() { changes.Add(1) })
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()

	path := filepath.Join(dir, "locales", "sv.json")
	if err = os.WriteFile(path, []byte(`{"_meta":{"locale":"sv","name":"Svenska"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool {
		for _, status := range registry.List() {
			if status.Locale == "sv" {
				return !status.Valid && status.Error != ""
			}
		}
		return false
	})

	valid := []byte(`{
		"_meta":{"locale":"sv","name":"Svenska","direction":"ltr","catalogVersion":1},
		"common":{"save":"Spara"}
	}`)
	if err = os.WriteFile(path, valid, 0o644); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return registry.Valid("sv") })
	catalog, ok := registry.Catalog("sv")
	if !ok || catalog.Messages["common"] == nil {
		t.Fatal("repaired Swedish catalog was not loaded")
	}
	if changes.Load() == 0 {
		t.Fatal("hot reload did not notify listeners")
	}

	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return registry.Resolve("sv") == "en" })
}

func TestRegistryRepairsBrokenEnglishOnStartup(t *testing.T) {
	dir := t.TempDir()
	locales := filepath.Join(dir, "locales")
	if err := os.MkdirAll(locales, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locales, "en.json"), []byte("broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	registry, err := New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()
	if !registry.Valid("en") {
		t.Fatal("English locale was not repaired")
	}
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}

func TestRegistryUpdatesOlderEnglishCatalog(t *testing.T) {
	dir := t.TempDir()
	locales := filepath.Join(dir, "locales")
	if err := os.MkdirAll(locales, 0o755); err != nil {
		t.Fatal(err)
	}
	old := []byte(`{
		"_meta":{"locale":"en","name":"English","direction":"ltr","catalogVersion":1},
		"common":{"save":"Old save"}
	}`)
	if err := os.WriteFile(filepath.Join(locales, "en.json"), old, 0o644); err != nil {
		t.Fatal(err)
	}

	registry, err := New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()

	catalog, ok := registry.Catalog("en")
	if !ok {
		t.Fatal("English locale was not available")
	}
	if catalog.Meta.CatalogVersion != 3 {
		t.Fatalf("catalog version=%d, want 3", catalog.Meta.CatalogVersion)
	}
	common, ok := catalog.Messages["common"].(map[string]any)
	if !ok || common["save"] != "Save" {
		t.Fatalf("old English catalog was not replaced: %#v", catalog.Messages["common"])
	}
}

func TestInvalidLocaleMetadataCannotShadowValidLocale(t *testing.T) {
	dir := t.TempDir()
	locales := filepath.Join(dir, "locales")
	if err := os.MkdirAll(locales, 0o755); err != nil {
		t.Fatal(err)
	}
	valid := []byte(`{
		"_meta":{"locale":"sv","name":"Svenska","direction":"ltr","catalogVersion":1},
		"common":{"save":"Spara"}
	}`)
	shadow := []byte(`{
		"_meta":{"locale":"sv","name":"Broken Swedish","direction":"ltr","catalogVersion":1},
		"common":{"save":"Fel"}
	}`)
	if err := os.WriteFile(filepath.Join(locales, "sv.json"), valid, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locales, "zz.json"), shadow, 0o644); err != nil {
		t.Fatal(err)
	}

	registry, err := New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()

	if !registry.Valid("sv") {
		t.Fatal("valid Swedish locale was shadowed by invalid metadata")
	}
	var foundBroken bool
	for _, status := range registry.List() {
		if status.Locale == "zz" {
			foundBroken = !status.Valid
		}
	}
	if !foundBroken {
		t.Fatal("invalid zz.json was not retained as a disabled locale")
	}
}

func TestNewerPartialEnglishKeepsEmbeddedFallbackKeys(t *testing.T) {
	dir := t.TempDir()
	locales := filepath.Join(dir, "locales")
	if err := os.MkdirAll(locales, 0o755); err != nil {
		t.Fatal(err)
	}
	custom := []byte(`{
		"_meta":{"locale":"en","name":"English","direction":"ltr","catalogVersion":99},
		"common":{"save":"Custom save"}
	}`)
	path := filepath.Join(locales, "en.json")
	if err := os.WriteFile(path, custom, 0o644); err != nil {
		t.Fatal(err)
	}

	registry, err := New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()

	catalog, ok := registry.Catalog("en")
	if !ok {
		t.Fatal("English locale was not available")
	}
	if catalog.Meta.CatalogVersion != 99 {
		t.Fatalf("catalog version=%d, want preserved version 99", catalog.Meta.CatalogVersion)
	}
	common := catalog.Messages["common"].(map[string]any)
	if common["save"] != "Custom save" {
		t.Fatalf("config English override was lost: %#v", common["save"])
	}
	if catalog.Messages["calendar"] == nil {
		t.Fatal("embedded English fallback keys were not merged")
	}
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(onDisk) != string(custom) {
		t.Fatal("newer English config file was unexpectedly rewritten")
	}
}
