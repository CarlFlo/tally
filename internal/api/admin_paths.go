package api

import "strings"

func adminPath(path string) bool {
	for _, prefix := range []string{"/api/settings", "/api/jobs", "/api/statistics", "/api/alerts", "/api/downloader", "/api/backups"} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}
