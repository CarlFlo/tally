package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/CarlFlo/tally/internal/backup"
	"github.com/CarlFlo/tally/internal/config"
)

func backupService(c config.Config) *backup.Service {
	return &backup.Service{
		DataDir:  c.DataDir,
		Path:     filepath.Join(c.DataDir, "backups"),
		Timezone: c.Timezone,
	}
}

func printBackupList(ctx context.Context, c config.Config) error {
	archives, err := backupService(c).Archives(ctx)
	if err != nil {
		return err
	}
	if len(archives) == 0 {
		fmt.Println("No backups found.")
		return nil
	}

	fmt.Println("Available backups:")
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "FILENAME\tTYPE")
	for _, archive := range archives {
		fmt.Fprintf(w, "%s\t%s\n", archive.Filename, archive.Kind)
	}
	return w.Flush()
}
