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
