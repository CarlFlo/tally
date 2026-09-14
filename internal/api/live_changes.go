package api

import (
	"net/http"
	"strings"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

// liveChanges is the single mapping from successful HTTP mutations to cached
// frontend resource domains. Keep transport concerns out of individual handlers.
func liveChanges(r *http.Request, session auth.Session) (string, []string) {
	path := r.URL.Path
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return "", nil
	}

	// Validation/test endpoints do not mutate authoritative application state.
	if path == "/api/settings/scheduling/preview" ||
		path == "/api/settings/search/test" ||
		path == "/api/settings/notifications/test" ||
		path == "/api/downloader/test" {
		return "", nil
	}

	// Shared deployment state is visible to every profile.
	if strings.HasPrefix(path, "/api/settings/") {
		switch strings.TrimPrefix(path, "/api/settings/") {
		case "search":
			return "", []string{"editable-settings", "settings", "capabilities"}
		case "notifications":
			return "", []string{"editable-settings", "settings"}
		case "backups":
			return "", []string{"editable-settings", "settings", "backups"}
		case "scheduling":
			return "", []string{"schedules", "jobs", "logs"}
		}
	}
	if strings.HasPrefix(path, "/api/jobs/") {
		return "", []string{"jobs", "schedules", "statistics"}
	}
	if path == "/api/downloader" {
		return "", []string{"downloader", "settings", "capabilities"}
	}
	if strings.HasPrefix(path, "/api/alerts/") {
		return "", []string{"logs", "inbox"}
	}

	// Profile/account changes affect only browsers using the same profile.
	profile := session.Profile
	switch {
	case path == "/api/show-actions":
		return profile, []string{"show-actions"}
	case strings.HasSuffix(path, "/favorite") && strings.HasPrefix(path, "/api/shows/"):
		return profile, []string{"shows", "show", "calendar"}
	case strings.HasSuffix(path, "/watch-history") && strings.HasPrefix(path, "/api/shows/"):
		return profile, []string{"shows", "show", "calendar", "logs"}
	case strings.HasSuffix(path, "/bulk") && strings.HasPrefix(path, "/api/shows/"):
		return profile, []string{"shows", "show", "calendar"}
	case strings.HasSuffix(path, "/refresh") && strings.HasPrefix(path, "/api/shows/"):
		return "", []string{"jobs"}
	case path == "/api/shows" || (strings.HasPrefix(path, "/api/shows/") && r.Method == http.MethodDelete):
		return profile, []string{"shows", "show", "calendar", "show-actions", "logs"}
	case strings.HasPrefix(path, "/api/episodes/"):
		return profile, []string{"shows", "show", "calendar"}
	case strings.HasPrefix(path, "/api/inbox"):
		return profile, []string{"inbox"}
	case path == "/api/preferences" || path == "/api/profile" || path == "/api/profile/avatar":
		return profile, []string{"bootstrap"}
	case strings.HasPrefix(path, "/api/auth/sessions/") || path == "/api/auth/password":
		return profile, []string{"bootstrap", "sessions"}
	case path == "/api/profiles" || strings.HasPrefix(path, "/api/profiles/"):
		return "", []string{"bootstrap"}
	case path == "/api/torrents/search" || path == "/api/torrents/send":
		return profile, []string{"torrent-history"}
	}

	return "", nil
}
