package backup

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/CarlFlo/mediaManager/internal/database"
)

// Restore is operator-only. The caller must hold the deployment lock with the server stopped.
func Restore(ctx context.Context, archive, dir string) error {
	stage, e := os.MkdirTemp(dir, ".restore-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(stage)
	if _, e = Extract(ctx, archive, stage); e != nil {
		return e
	}
	rollback := filepath.Join(dir, "pre-restore-"+time.Now().UTC().Format("20060102-150405")+"-"+database.ID()[:8])
	if e = os.Mkdir(rollback, 0700); e != nil {
		return e
	}
	old := []string{}
	for _, name := range []string{"app.db", "app.db-wal", "app.db-shm", "avatars"} {
		source := filepath.Join(dir, name)
		if _, e = os.Stat(source); os.IsNotExist(e) {
			continue
		}
		if e = os.Rename(source, filepath.Join(rollback, name)); e != nil {
			for _, n := range old {
				_ = os.Rename(filepath.Join(rollback, n), filepath.Join(dir, n))
			}
			return e
		}
		old = append(old, name)
	}
	installed := []string{}
	for _, name := range []string{"app.db", "avatars"} {
		source := filepath.Join(stage, name)
		if _, e = os.Stat(source); os.IsNotExist(e) {
			continue
		}
		if e = os.Rename(source, filepath.Join(dir, name)); e != nil {
			for _, n := range installed {
				_ = os.Rename(filepath.Join(dir, n), filepath.Join(stage, n))
			}
			for _, n := range old {
				_ = os.Rename(filepath.Join(rollback, n), filepath.Join(dir, n))
			}
			return e
		}
		installed = append(installed, name)
	}
	return nil
}
