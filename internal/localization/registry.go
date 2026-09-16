package localization

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
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

var localePattern = regexp.MustCompile(`^[A-Za-z]{2,3}(?:-[A-Za-z0-9]{2,8})*$`)

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
	if err = r.syncEnglish(); err != nil {
		return nil, err
	}
	if err = r.reload(false); err != nil {
		return nil, err
	}
	r.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create localization watcher: %w", err)
	}
	if err = r.watcher.Add(dir); err != nil {
		r.watcher.Close()
		return nil, fmt.Errorf("watch localization directory: %w", err)
	}
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

func (r *Registry) syncEnglish() error {
	path := filepath.Join(r.dir, "en.json")
	current, err := os.ReadFile(path)
	if err == nil {
		item, parseErr := parse("en.json", current)
		if parseErr == nil && item.status.CatalogVersion >= r.english.status.CatalogVersion {
			return nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read English localization: %w", err)
	}
	tmp, err := os.CreateTemp(r.dir, ".en-*.json")
	if err != nil {
		return fmt.Errorf("prepare English localization: %w", err)
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err = tmp.Chmod(0o644); err == nil {
		_, err = tmp.Write(embeddedEnglish)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write English localization: %w", err)
	}
	if err = os.Rename(name, path); err != nil {
		return fmt.Errorf("install English localization: %w", err)
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
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			slog.Warn("localization file unavailable", "file", file.Name(), "error", readErr)
			continue
		}
		item, parseErr := parse(file.Name(), data)
		if parseErr != nil {
			code := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
			item = entry{status: Status{Locale: code, Name: code, Valid: false, Error: publicError(parseErr)}}
			var meta struct{ Meta Meta `json:"_meta"` }
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
		// A valid same/newer config English file is preserved, but embedded
		// canonical English still supplies any missing keys in memory.
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
			if ok {
				slog.Warn("localization watcher error", "error", err)
			}
		case <-timerC:
			timerC = nil
			if err := r.reload(true); err != nil {
				slog.Warn("localization reload failed", "error", err)
			}
		}
	}
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

func publicError(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "invalid JSON"):
		return "Localization file contains invalid JSON."
	case strings.Contains(message, "_meta"), strings.Contains(message, "locale"), strings.Contains(message, "direction"), strings.Contains(message, "catalogVersion"):
		return "Localization metadata is invalid or incomplete."
	default:
		return "Localization file is invalid."
	}
}
