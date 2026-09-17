package localization

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
)

//go:embed en.json
var embeddedEnglish []byte

//go:embed uk.json
var embeddedUkrainian []byte

var bundledLocales = map[string][]byte{
	"en.json": embeddedEnglish,
	"uk.json": embeddedUkrainian,
}

var localePattern = regexp.MustCompile(`^[A-Za-z]{2,3}(?:-[A-Za-z0-9]{2,8})*$`)
var interpolationPattern = regexp.MustCompile(`\{\{\s*-?\s*([A-Za-z0-9_.]+)(?:\s*,[^{}]+)?\s*\}\}`)

const maxLocaleFileSize int64 = 4 << 20

var newFSWatcher = fsnotify.NewWatcher

type Meta struct {
	Locale         string `json:"locale"`
	Name           string `json:"name"`
	Direction      string `json:"direction"`
	CatalogVersion int    `json:"catalogVersion"`
}

type Catalog struct {
	Meta     Meta           `json:"meta"`
	Messages map[string]any `json:"messages"`
	Revision uint64         `json:"revision"`
}

type Status struct {
	Locale         string `json:"locale"`
	Name           string `json:"name"`
	Direction      string `json:"direction,omitempty"`
	CatalogVersion int    `json:"catalog_version,omitempty"`
	Valid          bool   `json:"valid"`
	Error          string `json:"error,omitempty"`
	ErrorCode      string `json:"error_code,omitempty"`
}

type entry struct {
	status   Status
	messages map[string]any
}

type Registry struct {
	dir       string
	mu        sync.RWMutex
	entries   map[string]entry
	revision  atomic.Uint64
	english   entry
	watcher   *fsnotify.Watcher
	done      chan struct{}
	closeOnce sync.Once
	onChange  func()
	wg        sync.WaitGroup
}

func New(dataDir string, onChange func()) (*Registry, error) {
	english, err := parse("en.json", embeddedEnglish)
	if err != nil {
		return nil, fmt.Errorf("embedded English localization is invalid: %w", err)
	}
	dir := filepath.Join(dataDir, "locales")
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create localization directory: %w", err)
	}
	r := &Registry{dir: dir, entries: map[string]entry{}, english: english, done: make(chan struct{}), onChange: onChange}
	for filename, data := range bundledLocales {
		item, parseErr := parse(filename, data)
		if parseErr == nil {
			parseErr = validatePlaceholders(item.messages, english.messages, "")
		}
		if parseErr != nil {
			return nil, fmt.Errorf("embedded %s localization is invalid: %w", filename, parseErr)
		}
		if err = r.syncBundledLocale(filename, data, item.status.CatalogVersion); err != nil {
			return nil, err
		}
	}
	if err = r.reload(false); err != nil {
		return nil, err
	}
	watcher, watchErr := newFSWatcher()
	if watchErr != nil {
		slog.Warn("Localization watcher unavailable; locale changes require a page reload", "error", watchErr)
		return r, nil
	}
	if watchErr = watcher.Add(dir); watchErr != nil {
		_ = watcher.Close()
		slog.Warn("Localization watcher unavailable; locale changes require a page reload", "error", watchErr)
		return r, nil
	}
	r.watcher = watcher
	r.wg.Add(1)
	go r.watch()
	return r, nil
}

func (r *Registry) Close() error {
	var err error
	r.closeOnce.Do(func() {
		close(r.done)
		if r.watcher != nil {
			err = r.watcher.Close()
		}
		r.wg.Wait()
	})
	return err
}

func (r *Registry) List() []Status {
	r.mu.RLock()
	out := make([]Status, 0, len(r.entries))
	for _, item := range r.entries {
		out = append(out, item.status)
	}
	r.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].Locale == "en" {
			return true
		}
		if out[j].Locale == "en" {
			return false
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func (r *Registry) Catalog(locale string) (Catalog, bool) {
	r.mu.RLock()
	item, ok := r.entries[locale]
	revision := r.revision.Load()
	r.mu.RUnlock()
	if !ok || !item.status.Valid {
		return Catalog{}, false
	}
	return Catalog{Meta: Meta{Locale: item.status.Locale, Name: item.status.Name, Direction: item.status.Direction, CatalogVersion: item.status.CatalogVersion}, Messages: item.messages, Revision: revision}, true
}

func (r *Registry) Resolve(locale string) string {
	if locale == "" {
		return "en"
	}
	r.mu.RLock()
	item, ok := r.entries[locale]
	r.mu.RUnlock()
	if ok && item.status.Valid {
		return locale
	}
	return "en"
}

func (r *Registry) Valid(locale string) bool {
	_, ok := r.Catalog(locale)
	return ok
}

func (r *Registry) Revision() uint64 { return r.revision.Load() }

func (r *Registry) syncBundledLocale(filename string, data []byte, bundledVersion int) error {
	path := filepath.Join(r.dir, filename)
	info, statErr := os.Lstat(path)
	if statErr == nil && !info.Mode().IsRegular() {
		// Never follow symlinks or replace special files at a server-owned
		// bundled catalog path. The embedded English catalog remains the final
		// runtime fallback if its persistent path cannot be reconciled.
		slog.Warn("Bundled localization path is not a regular file; leaving it untouched", "file", path, "mode", info.Mode())
		return nil
	}
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		slog.Warn("Bundled localization path could not be inspected; attempting catalog restore", "file", path, "error", statErr)
	}
	if statErr == nil {
		current, readErr := readLocaleFile(path)
		if readErr == nil {
			item, parseErr := parse(filename, current)
			if parseErr == nil {
				parseErr = validatePlaceholders(item.messages, r.english.messages, "")
			}
			if parseErr == nil {
				switch {
				case item.status.CatalogVersion > bundledVersion:
					// Preserve a catalog written by a newer Tally release so a
					// temporary application downgrade does not destroy it.
					return nil
				case item.status.CatalogVersion == bundledVersion && bytes.Equal(current, data):
					// Avoid touching the file when the managed copy is already exact.
					return nil
				}
			}
		} else {
			slog.Warn("Bundled localization file could not be read; restoring embedded catalog", "file", path, "error", readErr)
		}
	}

	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	tmp, err := os.CreateTemp(r.dir, "."+base+"-*.json")
	if err != nil {
		return fmt.Errorf("prepare bundled localization %s: %w", filename, err)
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err = tmp.Chmod(0o644); err == nil {
		_, err = tmp.Write(data)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write bundled localization %s: %w", filename, err)
	}
	if err = os.Rename(name, path); err != nil {
		return fmt.Errorf("install bundled localization %s: %w", filename, err)
	}
	return nil
}

func (r *Registry) reload(notify bool) error {
	files, err := os.ReadDir(r.dir)
	if err != nil {
		return fmt.Errorf("scan localization directory: %w", err)
	}
	next := make(map[string]entry)
	for _, file := range files {
		if file.IsDir() || !strings.EqualFold(filepath.Ext(file.Name()), ".json") {
			continue
		}
		path := filepath.Join(r.dir, file.Name())
		info, infoErr := file.Info()
		if infoErr != nil || !info.Mode().IsRegular() {
			code := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
			err := infoErr
			if err == nil {
				err = fmt.Errorf("not a regular file")
			}
			slog.Warn("localization entry is not a regular file", "file", file.Name(), "error", err)
			next[code] = entry{status: Status{
				Locale:    code,
				Name:      code,
				Valid:     false,
				Error:     "Localization entry must be a regular file.",
				ErrorCode: "language.notRegularFile",
			}}
			continue
		}
		data, readErr := readLocaleFile(path)
		if readErr != nil {
			code := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
			slog.Warn("localization file unavailable", "file", file.Name(), "error", readErr)
			next[code] = entry{status: Status{
				Locale:    code,
				Name:      code,
				Valid:     false,
				Error:     "Localization file could not be read.",
				ErrorCode: "language.unreadableFile",
			}}
			continue
		}
		item, parseErr := parse(file.Name(), data)
		if parseErr == nil {
			parseErr = validatePlaceholders(item.messages, r.english.messages, "")
		}
		if parseErr != nil {
			code := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
			item = entry{status: Status{Locale: code, Name: code, Valid: false, Error: publicError(parseErr), ErrorCode: publicErrorCode(parseErr)}}
			var meta struct {
				Meta Meta `json:"_meta"`
			}
			if json.Unmarshal(data, &meta) == nil {
				// The filename remains authoritative for invalid files. A bad
				// _meta.locale must never shadow a separate valid locale.
				if meta.Meta.Name != "" {
					item.status.Name = meta.Meta.Name
				}
				item.status.Direction = meta.Meta.Direction
				item.status.CatalogVersion = meta.Meta.CatalogVersion
			}
			slog.Warn("invalid localization file", "file", file.Name(), "error", parseErr)
		}
		next[item.status.Locale] = item
	}
	if item, ok := next["en"]; !ok || !item.status.Valid {
		// Embedded English is always available even if the config copy was damaged at runtime.
		next["en"] = r.english
	} else {
		// A valid config English file from a newer Tally release may be
		// preserved on disk. Embedded canonical English still supplies any
		// missing keys in memory when that happens.
		item.messages = mergeMessages(r.english.messages, item.messages)
		next["en"] = item
	}
	r.mu.Lock()
	r.entries = next
	r.revision.Add(1)
	r.mu.Unlock()
	if notify && r.onChange != nil {
		r.onChange()
	}
	return nil
}

func (r *Registry) watch() {
	defer r.wg.Done()
	var timer *time.Timer
	var timerC <-chan time.Time
	schedule := func() {
		if timer == nil {
			timer = time.NewTimer(150 * time.Millisecond)
		} else {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(150 * time.Millisecond)
		}
		timerC = timer.C
	}
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	for {
		select {
		case <-r.done:
			return
		case event, ok := <-r.watcher.Events:
			if !ok {
				return
			}
			if strings.EqualFold(filepath.Ext(event.Name), ".json") {
				schedule()
			}
		case err, ok := <-r.watcher.Errors:
			if !ok {
				return
			}
			slog.Warn("localization watcher error; reconciling locale registry", "error", err)
			if reloadErr := r.reload(true); reloadErr != nil {
				slog.Warn("localization reconciliation failed", "error", reloadErr)
			}
		case <-timerC:
			timerC = nil
			if err := r.reload(true); err != nil {
				slog.Warn("localization reload failed", "error", err)
			}
		}
	}
}

func readLocaleFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxLocaleFileSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxLocaleFileSize {
		return nil, fmt.Errorf("localization file exceeds %d-byte limit", maxLocaleFileSize)
	}
	return data, nil
}

func mergeMessages(fallback, override map[string]any) map[string]any {
	out := make(map[string]any, len(fallback)+len(override))
	for key, value := range fallback {
		if nested, ok := value.(map[string]any); ok {
			out[key] = mergeMessages(nested, nil)
		} else {
			out[key] = value
		}
	}
	for key, value := range override {
		nestedOverride, overrideOK := value.(map[string]any)
		nestedFallback, fallbackOK := out[key].(map[string]any)
		if overrideOK && fallbackOK {
			out[key] = mergeMessages(nestedFallback, nestedOverride)
			continue
		}
		if overrideOK {
			out[key] = mergeMessages(nil, nestedOverride)
			continue
		}
		out[key] = value
	}
	return out
}

func parse(filename string, data []byte) (entry, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return entry{}, fmt.Errorf("invalid JSON: %w", err)
	}
	metaRaw, ok := raw["_meta"]
	if !ok {
		return entry{}, errors.New("missing _meta")
	}
	metaBytes, _ := json.Marshal(metaRaw)
	var meta Meta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return entry{}, errors.New("invalid _meta")
	}
	if !localePattern.MatchString(meta.Locale) {
		return entry{}, errors.New("invalid locale code")
	}
	expected := strings.TrimSuffix(filename, filepath.Ext(filename))
	if meta.Locale != expected {
		return entry{}, fmt.Errorf("locale %q does not match filename %q", meta.Locale, expected)
	}
	if strings.TrimSpace(meta.Name) == "" {
		return entry{}, errors.New("missing locale display name")
	}
	if meta.Direction != "ltr" && meta.Direction != "rtl" {
		return entry{}, errors.New("direction must be ltr or rtl")
	}
	if meta.CatalogVersion < 1 {
		return entry{}, errors.New("catalogVersion must be at least 1")
	}
	delete(raw, "_meta")
	if len(raw) == 0 {
		return entry{}, errors.New("translation catalog is empty")
	}
	if err := validateNode(raw, ""); err != nil {
		return entry{}, err
	}
	return entry{status: Status{Locale: meta.Locale, Name: meta.Name, Direction: meta.Direction, CatalogVersion: meta.CatalogVersion, Valid: true}, messages: raw}, nil
}

func validateNode(node map[string]any, prefix string) error {
	for key, value := range node {
		if strings.TrimSpace(key) == "" {
			return errors.New("translation key cannot be empty")
		}
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		switch typed := value.(type) {
		case string:
			if typed == "" {
				return fmt.Errorf("translation %s is empty", path)
			}
		case map[string]any:
			if len(typed) == 0 {
				return fmt.Errorf("translation group %s is empty", path)
			}
			if err := validateNode(typed, path); err != nil {
				return err
			}
		default:
			return fmt.Errorf("translation %s must be a string or object", path)
		}
	}
	return nil
}

func validatePlaceholders(messages, english map[string]any, prefix string) error {
	for key, value := range messages {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		englishValue, hasEnglish := english[key]
		switch translated := value.(type) {
		case string:
			translatedSet, err := interpolationVariables(translated)
			if err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			englishString, englishIsString := englishValue.(string)
			if !hasEnglish || !englishIsString {
				continue
			}
			englishSet, err := interpolationVariables(englishString)
			if err != nil {
				return fmt.Errorf("embedded English %s: %w", path, err)
			}
			if !sameVariables(translatedSet, englishSet) {
				return fmt.Errorf("interpolation placeholders for %s do not match English", path)
			}
		case map[string]any:
			englishMap, _ := englishValue.(map[string]any)
			if englishMap == nil {
				englishMap = map[string]any{}
			}
			if err := validatePlaceholders(translated, englishMap, path); err != nil {
				return err
			}
		}
	}
	return nil
}

func interpolationVariables(value string) (map[string]struct{}, error) {
	variables := make(map[string]struct{})
	cleaned := interpolationPattern.ReplaceAllStringFunc(value, func(match string) string {
		parts := interpolationPattern.FindStringSubmatch(match)
		if len(parts) > 1 {
			variables[parts[1]] = struct{}{}
		}
		return ""
	})
	if strings.Contains(cleaned, "{{") || strings.Contains(cleaned, "}}") {
		return nil, errors.New("malformed interpolation placeholder")
	}
	return variables, nil
}

func sameVariables(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for key := range a {
		if _, ok := b[key]; !ok {
			return false
		}
	}
	return true
}

func publicErrorCode(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "interpolation placeholder"):
		return "language.placeholderMismatch"
	case strings.Contains(message, "invalid JSON"):
		return "language.invalidJson"
	case strings.Contains(message, "_meta"), strings.Contains(message, "locale"), strings.Contains(message, "direction"), strings.Contains(message, "catalogVersion"):
		return "language.missingMetadata"
	default:
		return "language.invalidFile"
	}
}

func publicError(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "interpolation placeholder"):
		return "Translation placeholders do not match the English catalog."
	case strings.Contains(message, "invalid JSON"):
		return "Localization file contains invalid JSON."
	case strings.Contains(message, "_meta"), strings.Contains(message, "locale"), strings.Contains(message, "direction"), strings.Contains(message, "catalogVersion"):
		return "Localization metadata is invalid or incomplete."
	default:
		return "Localization file is invalid."
	}
}
