package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/CarlFlo/tally/internal/backup"
	"github.com/CarlFlo/tally/internal/config"
)

func verifyBackup(ctx context.Context, c config.Config, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: tally verify-backup <backup.zip>")
	}
	archive, err := backupArchivePath(c, args[0])
	if err != nil {
		return err
	}
	stage, e := os.MkdirTemp(c.DataDir, ".verify-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(stage)
	_, e = backup.Extract(ctx, archive, stage)
	if e == nil {
		fmt.Println("Backup verified")
	}
	return e
}
