package backup

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const archiveWatchDebounce = 200 * time.Millisecond

// ArchiveWatcher observes completed backup archives in a single backup
// directory. It is event-driven; it never polls the filesystem.
type ArchiveWatcher struct {
	watcher   *fsnotify.Watcher
	onChange  func()
	done      chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
}

func WatchArchives(path string, onChange func()) (*ArchiveWatcher, error) {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return nil, err
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err = watcher.Add(path); err != nil {
		_ = watcher.Close()
		return nil, err
	}
	result := &ArchiveWatcher{
		watcher:  watcher,
		onChange: onChange,
		done:     make(chan struct{}),
	}
	result.wg.Add(1)
	go result.watch()
	return result, nil
}

func (w *ArchiveWatcher) Close() error {
	if w == nil {
		return nil
	}
	var err error
	w.closeOnce.Do(func() {
		close(w.done)
		err = w.watcher.Close()
		w.wg.Wait()
	})
	return err
}

func archiveWatchEvent(event fsnotify.Event) bool {
	if !validArchiveName(filepath.Base(event.Name)) {
		return false
	}
	return event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename|fsnotify.Remove) != 0
}

func (w *ArchiveWatcher) watch() {
	defer w.wg.Done()
	var timer *time.Timer
	var timerC <-chan time.Time
	schedule := func() {
		if timer == nil {
			timer = time.NewTimer(archiveWatchDebounce)
		} else {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(archiveWatchDebounce)
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
		case <-w.done:
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if archiveWatchEvent(event) {
				schedule()
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			slog.Warn("backup archive watcher error; reconciling archive list", "error", err)
			schedule()
		case <-timerC:
			timerC = nil
			if w.onChange != nil {
				w.onChange()
			}
		}
	}
}
