package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func validArchiveName(name string) bool {
	return name != "" &&
		filepath.Base(name) == name &&
		strings.HasSuffix(strings.ToLower(name), ".zip") &&
		!strings.ContainsAny(name, "/\\:")
}

func (s *Service) archivePath(name string) (string, error) {
	if !validArchiveName(name) {
		return "", fmt.Errorf("unsafe backup filename")
	}
	path := filepath.Join(s.Path, name)
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("backup archive is not a regular file")
	}
	return path, nil
}
