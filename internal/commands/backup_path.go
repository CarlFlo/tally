package commands

import (
	"fmt"
	"os"
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
		path := filepath.Clean(archive)
		if info, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("backup archive not found: %s", path)
			}
			return "", err
		} else if !info.Mode().IsRegular() {
			return "", fmt.Errorf("backup archive is not a regular file: %s", path)
		}
		return path, nil
	}

	clean := filepath.Clean(archive)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("relative backup archive must be inside the backups directory")
	}

	candidates := []string{clean}
	if !strings.HasSuffix(strings.ToLower(clean), ".zip") {
		candidates = append(candidates, clean+".zip")
	}
	for _, candidate := range candidates {
		path := filepath.Join(c.DataDir, "backups", candidate)
		info, err := os.Stat(path)
		if err == nil {
			if !info.Mode().IsRegular() {
				return "", fmt.Errorf("backup archive is not a regular file: %s", path)
			}
			return filepath.Abs(path)
		}
		if !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("backup %q not found in %s; run tally restore or tally verify-backup without a filename to list available backups", archive, filepath.Join(c.DataDir, "backups"))
}
