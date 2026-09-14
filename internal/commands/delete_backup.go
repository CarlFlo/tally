package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CarlFlo/mediaManager/internal/backup"
	"github.com/CarlFlo/mediaManager/internal/database"
)

func deleteBackup(ctx context.Context, db *database.Store, b *backup.Service, args []string) error {
	var e error
	if len(args) != 1 {
		return fmt.Errorf("usage: tally delete-backup <filename>")
	}
	name := args[0]
	if filepath.Base(name) != name || !strings.HasSuffix(name, ".zip") {
		return fmt.Errorf("supply a backup filename, not a path")
	}
	if e = os.Remove(filepath.Join(b.Path, name)); e != nil {
		return e
	}
	_, e = db.ExecContext(ctx, "DELETE FROM backup_records WHERE filename=?", name)
	if e == nil {
		fmt.Println("Backup deleted")
	}
	return e
}
