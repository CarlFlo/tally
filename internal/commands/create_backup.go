package commands

import (
	"context"
	"fmt"

	"github.com/CarlFlo/tally/internal/backup"
)

func createBackup(ctx context.Context, b *backup.Service) error {
	name, err := b.Create(ctx, "manual")
	if err != nil {
		return err
	}
	printBackupCreated(name)
	return nil
}

func printBackupCreated(name string) {
	fmt.Printf("Backup created: %s\n", name)
}
