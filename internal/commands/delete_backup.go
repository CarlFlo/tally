package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/CarlFlo/tally/internal/backup"
)

func deleteBackup(ctx context.Context, b *backup.Service, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: tally delete-backup <filename>")
	}
	name := args[0]
	if filepath.Base(name) != name || !strings.HasSuffix(strings.ToLower(name), ".zip") {
		return fmt.Errorf("supply a backup filename, not a path")
	}
	archives, err := b.Archives(ctx)
	if err != nil {
		return err
	}
	for _, archive := range archives {
		if archive.Filename != name {
			continue
		}
		if err = b.DeleteArchive(ctx, archive.ID); err != nil {
			return err
		}
		fmt.Println("Backup deleted")
		return nil
	}
	return backup.ErrNotFound
}
