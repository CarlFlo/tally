package commands

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/CarlFlo/tally/internal/config"
)

func backupArchivePath(c config.Config, archive string) (string, error) {
	archive = strings.TrimSpace(archive)
	if archive == "" {
		return "", fmt.Errorf("backup archive cannot be empty")
	}
	if filepath.IsAbs(archive) {
		return filepath.Clean(archive), nil
	}

	clean := filepath.Clean(archive)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("relative backup archive must be inside the backups directory")
	}
	return filepath.Abs(filepath.Join(c.DataDir, "backups", clean))
}
