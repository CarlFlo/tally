package backup

import (
	"path/filepath"
	"strings"
)

func safeName(name string) bool {
	if name == "app.db" || name == "manifest.json" {
		return true
	}
	if !strings.HasPrefix(name, "avatars/") || strings.Contains(name, "\\") {
		return false
	}
	base := strings.TrimPrefix(name, "avatars/")
	return base != "" && filepath.Base(base) == base && strings.HasSuffix(base, ".png") && !strings.Contains(base, "..")
}
