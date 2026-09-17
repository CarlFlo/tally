package localization

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRegistryRestoresModifiedBundledLocalesOnStartup(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		bundled  []byte
	}{
		{name: "English", filename: "en.json", bundled: embeddedEnglish},
		{name: "Ukrainian", filename: "uk.json", bundled: embeddedUkrainian},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			locales := filepath.Join(dir, "locales")
			if err := os.MkdirAll(locales, 0o755); err != nil {
				t.Fatal(err)
			}

			modified := modifyBundledLocaleForTest(t, tt.bundled, func(raw map[string]any) {
				common := raw["common"].(map[string]any)
				common["save"] = "operator override"
			})
			path := filepath.Join(locales, tt.filename)
			if err := os.WriteFile(path, modified, 0o644); err != nil {
				t.Fatal(err)
			}

			registry, err := New(dir, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer registry.Close()

			onDisk, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(onDisk, tt.bundled) {
				t.Fatalf("%s bundled locale was not restored to the embedded copy", tt.filename)
			}
		})
	}
}

func TestRegistryUpdatesOlderBundledLocaleOnStartup(t *testing.T) {
	dir := t.TempDir()
	locales := filepath.Join(dir, "locales")
	if err := os.MkdirAll(locales, 0o755); err != nil {
		t.Fatal(err)
	}

	older := modifyBundledLocaleForTest(t, embeddedEnglish, func(raw map[string]any) {
		meta := raw["_meta"].(map[string]any)
		meta["catalogVersion"] = float64(1)
		common := raw["common"].(map[string]any)
		common["save"] = "Old save"
	})
	path := filepath.Join(locales, "en.json")
	if err := os.WriteFile(path, older, 0o644); err != nil {
		t.Fatal(err)
	}

	registry, err := New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()

	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(onDisk, embeddedEnglish) {
		t.Fatal("older bundled English locale was not upgraded")
	}
}

func TestRegistryDoesNotRewriteCurrentBundledLocale(t *testing.T) {
	dir := t.TempDir()
	locales := filepath.Join(dir, "locales")
	if err := os.MkdirAll(locales, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(locales, "uk.json")
	if err := os.WriteFile(path, embeddedUkrainian, 0o644); err != nil {
		t.Fatal(err)
	}
	stamp := time.Unix(123456789, 0)
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatal(err)
	}

	registry, err := New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(stamp) {
		t.Fatalf("current bundled locale was rewritten: modtime=%v want=%v", info.ModTime(), stamp)
	}
}

func TestRegistryLeavesCustomLocaleFilenameUntouched(t *testing.T) {
	dir := t.TempDir()
	locales := filepath.Join(dir, "locales")
	if err := os.MkdirAll(locales, 0o755); err != nil {
		t.Fatal(err)
	}
	custom := []byte(`{
		"_meta":{"locale":"sv","name":"Svenska","direction":"ltr","catalogVersion":1},
		"common":{"save":"Spara"}
	}`)
	path := filepath.Join(locales, "sv.json")
	if err := os.WriteFile(path, custom, 0o644); err != nil {
		t.Fatal(err)
	}

	registry, err := New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()

	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(onDisk, custom) {
		t.Fatal("custom locale with a unique filename was modified")
	}
	if !registry.Valid("sv") {
		t.Fatal("custom locale was not loaded")
	}
}

func modifyBundledLocaleForTest(t *testing.T, data []byte, mutate func(map[string]any)) []byte {
	t.Helper()
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	mutate(raw)
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(out, '\n')
}
