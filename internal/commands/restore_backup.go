package commands

import (
	"context"
	"fmt"

	"github.com/CarlFlo/tally/internal/backup"
	"github.com/CarlFlo/tally/internal/config"
)

func restoreBackup(ctx context.Context, c config.Config, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: tally restore <backup.zip>")
	}
	archive, err := backupArchivePath(c, args[0])
	if err != nil {
		return err
	}
	if err = backup.Restore(ctx, archive, c.DataDir); err != nil {
		return err
	}
	fmt.Println("Backup restored successfully.")
	return nil
}
