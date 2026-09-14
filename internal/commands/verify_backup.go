package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/CarlFlo/mediaManager/internal/backup"
	"github.com/CarlFlo/mediaManager/internal/config"
)

func verifyBackup(ctx context.Context, c config.Config, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: tally verify-backup <archive>")
	}
	stage, e := os.MkdirTemp(c.DataDir, ".verify-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(stage)
	_, e = backup.Extract(ctx, args[0], stage)
	if e == nil {
		fmt.Println("Backup verified")
	}
	return e
}
