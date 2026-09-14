package commands

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/CarlFlo/mediaManager/internal/backup"
)

func createBackup(ctx context.Context, b *backup.Service) error {
	name, e := b.Create(ctx, "manual")
	if e == nil {
		fmt.Println(filepath.Join(b.Path, name))
	}
	return e
}
