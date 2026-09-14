package commands

import (
	"context"
	"fmt"

	"github.com/CarlFlo/mediaManager/internal/backup"
	"github.com/CarlFlo/mediaManager/internal/config"
)

func restoreBackup(ctx context.Context, c config.Config, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: tally restore <archive>")
	}
	if err := backup.Restore(ctx, args[0], c.DataDir); err != nil {
		return err
	}
	fmt.Println("Backup restored; previous data preserved in pre-restore directory")
	return nil
}
