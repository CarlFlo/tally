package restore

import (
	"context"

	"github.com/CarlFlo/tally/internal/backup"
	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/flows"
	"github.com/CarlFlo/tally/internal/settings"
)

type Publisher interface {
	Publish(string, ...string)
}

type Scheduler interface {
	RefreshSchedules()
}

type Coordinator struct {
	DB     *database.Store
	Backup *backup.Service
	Events Publisher
	Jobs   Scheduler
}

func (c Coordinator) Archive(ctx context.Context, id string) (backup.Manifest, error) {
	manifest, err := c.Backup.RestoreArchive(ctx, id)
	if err != nil {
		return manifest, err
	}
	return manifest, c.after(ctx)
}

func (c Coordinator) File(ctx context.Context, path string) (backup.Manifest, error) {
	manifest, err := c.Backup.RestoreLive(ctx, path)
	if err != nil {
		return manifest, err
	}
	return manifest, c.after(ctx)
}

func (c Coordinator) after(ctx context.Context) error {
	if err := (settings.Store{DB: c.DB}).Ensure(ctx); err != nil {
		return err
	}
	if err := (flows.Store{DB: c.DB}).EnsureDefaults(ctx); err != nil {
		return err
	}
	if c.Jobs != nil {
		c.Jobs.RefreshSchedules()
	}
	if c.Events != nil {
		c.Events.Publish("", "bootstrap", "calendar", "shows", "show-actions", "jobs", "schedules", "statistics", "logs", "settings", "editable-settings", "downloader", "backups", "inbox", "torrent-history", "capabilities", "sessions")
	}
	return nil
}
