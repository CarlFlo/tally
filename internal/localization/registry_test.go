package localization

import (
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
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
	if !registry.Valid("uk") {
		t.Fatal("bundled Ukrainian locale was not available")
	}
	if _, err = os.Stat(filepath.Join(dir, "locales", "uk.json")); err != nil {
		t.Fatal("Ukrainian locale was not written to the config directory:", err)
	}
}

func TestRegistryPreservesExistingBundledLocale(t *testing.T) {
	dir := t.TempDir()
	locales := filepath.Join(dir, "locales")
	if err := os.MkdirAll(locales, 0o755); err != nil {
		t.Fatal(err)
	}
	custom := []byte(`{
		"_meta":{"locale":"uk","name":"Українська","direction":"ltr","catalogVersion":99},
		"common":{"save":"Моє збереження"}
	}`)
	path := filepath.Join(locales, "uk.json")
	if err := os.WriteFile(path, custom, 0o644); err != nil {
		t.Fatal(err)
	}

	registry, err := New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()

	catalog, ok := registry.Catalog("uk")
	if !ok {
		t.Fatal("existing Ukrainian locale was not loaded")
	}
	common, ok := catalog.Messages["common"].(map[string]any)
	if !ok || common["save"] != "Моє збереження" {
		t.Fatalf("existing Ukrainian locale was overwritten: %#v", catalog.Messages["common"])
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(custom) {
		t.Fatal("existing Ukrainian locale file was modified")
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
	if catalog.Meta.CatalogVersion != 28 {
		t.Fatalf("catalog version=%d, want 28", catalog.Meta.CatalogVersion)
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

func TestRegistryRejectsSymlinkLocaleEntries(t *testing.T) {
	dir := t.TempDir()
	locales := filepath.Join(dir, "locales")
	if err := os.MkdirAll(locales, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "outside.json")
	if err := os.WriteFile(target, []byte(`{
		"_meta":{"locale":"sv","name":"Svenska","direction":"ltr","catalogVersion":1},
		"common":{"save":"Spara"}
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(locales, "sv.json")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	registry, err := New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()

	if registry.Valid("sv") {
		t.Fatal("symlink locale was accepted")
	}
	var statusFound bool
	for _, status := range registry.List() {
		if status.Locale == "sv" {
			statusFound = true
			if status.Valid || status.ErrorCode != "language.notRegularFile" {
				t.Fatalf("unexpected symlink status: %+v", status)
			}
		}
	}
	if !statusFound {
		t.Fatal("symlink locale was not retained as a disabled entry")
	}
}

func TestRegistryRejectsMismatchedTranslationPlaceholders(t *testing.T) {
	dir := t.TempDir()
	registry, err := New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()

	path := filepath.Join(dir, "locales", "sv.json")
	mismatched := []byte(`{
		"_meta":{"locale":"sv","name":"Svenska","direction":"ltr","catalogVersion":1},
		"profile":{"passwordHint":"Använd minst {{minimum}} tecken."}
	}`)
	if err = os.WriteFile(path, mismatched, 0o644); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool {
		for _, status := range registry.List() {
			if status.Locale == "sv" {
				return !status.Valid && status.ErrorCode == "language.placeholderMismatch"
			}
		}
		return false
	})
}

func TestRegistryRejectsMalformedTranslationPlaceholder(t *testing.T) {
	dir := t.TempDir()
	registry, err := New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()

	path := filepath.Join(dir, "locales", "sv.json")
	malformed := []byte(`{
		"_meta":{"locale":"sv","name":"Svenska","direction":"ltr","catalogVersion":1},
		"common":{"poster":"{{name poster"}
	}`)
	if err = os.WriteFile(path, malformed, 0o644); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool {
		for _, status := range registry.List() {
			if status.Locale == "sv" {
				return !status.Valid && status.ErrorCode == "language.placeholderMismatch"
			}
		}
		return false
	})
}

func TestRegistryDoesNotFollowEnglishSymlinkOnStartup(t *testing.T) {
	dir := t.TempDir()
	locales := filepath.Join(dir, "locales")
	if err := os.MkdirAll(locales, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "outside-en.json")
	if err := os.WriteFile(target, []byte(`{
		"_meta":{"locale":"en","name":"English","direction":"ltr","catalogVersion":99},
		"common":{"save":"Linked save"}
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(locales, "en.json")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	registry, err := New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()

	catalog, ok := registry.Catalog("en")
	if !ok {
		t.Fatal("embedded English fallback was not available")
	}
	if catalog.Meta.CatalogVersion != registry.english.status.CatalogVersion {
		t.Fatalf("catalog version=%d, want bundled version %d", catalog.Meta.CatalogVersion, registry.english.status.CatalogVersion)
	}
	common := catalog.Messages["common"].(map[string]any)
	if common["save"] != "Save" {
		t.Fatalf("English symlink target was followed: %#v", common["save"])
	}
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("server unexpectedly rewrote the non-regular English path")
	}
}

func TestRegistryContinuesWhenWatcherCannotStart(t *testing.T) {
	original := newFSWatcher
	newFSWatcher = func() (*fsnotify.Watcher, error) {
		return nil, errors.New("watcher unavailable")
	}
	t.Cleanup(func() { newFSWatcher = original })

	registry, err := New(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("New returned an error when only the watcher failed: %v", err)
	}
	if !registry.Valid("en") {
		t.Fatal("English locale was unavailable without filesystem watching")
	}
	if registry.watcher != nil {
		t.Fatal("registry unexpectedly retained a watcher after startup failure")
	}
	if err = registry.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRegistryRejectsOversizedLocaleFile(t *testing.T) {
	dir := t.TempDir()
	locales := filepath.Join(dir, "locales")
	if err := os.MkdirAll(locales, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(locales, "zz.json")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = file.Truncate(maxLocaleFileSize + 1); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}

	registry, err := New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()

	for _, status := range registry.List() {
		if status.Locale == "zz" {
			if status.Valid {
				t.Fatal("oversized locale file was accepted")
			}
			if status.ErrorCode != "language.unreadableFile" {
				t.Fatalf("unexpected oversized locale error code: %q", status.ErrorCode)
			}
			return
		}
	}
	t.Fatal("oversized locale file was not retained as a disabled entry")
}
