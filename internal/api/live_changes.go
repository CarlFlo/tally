package api

import (
	"net/http"
	"strings"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

type liveUpdate struct {
	Profile   string
	Resources []string
}

func update(profile string, resources ...string) liveUpdate {
	return liveUpdate{Profile: profile, Resources: resources}
}

func profileAndAdmin(profile string, profileResources []string, adminResources ...string) []liveUpdate {
	updates := []liveUpdate{update(profile, profileResources...)}
	if profile != "user0" && len(adminResources) > 0 {
		updates = append(updates, update("user0", adminResources...))
	}
	return updates
}

// liveChanges is the single mapping from successful HTTP mutations to cached
// frontend resource domains. Keep transport concerns out of individual handlers.
func liveChanges(r *http.Request, session auth.Session) []liveUpdate {
	path := r.URL.Path
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return nil
	}

	// Validation/test endpoints do not mutate authoritative application state.
	if path == "/api/settings/scheduling/preview" ||
		path == "/api/settings/search/test" ||
		path == "/api/settings/notifications/test" ||
		path == "/api/downloader/test" {
		return nil
	}

	// Shared deployment state is visible to every profile.
	if strings.HasPrefix(path, "/api/settings/") {
		switch strings.TrimPrefix(path, "/api/settings/") {
		case "search":
			return []liveUpdate{update("", "editable-settings", "settings", "capabilities")}
		case "notifications":
			return []liveUpdate{update("", "editable-settings", "settings")}
		case "backups":
			return []liveUpdate{update("", "editable-settings", "settings", "backups")}
		case "scheduling":
			return []liveUpdate{update("", "schedules", "jobs", "logs")}
		}
	}
	if strings.HasPrefix(path, "/api/jobs/") {
		return []liveUpdate{update("", "jobs", "schedules", "statistics")}
	}
	if path == "/api/downloader" {
		return []liveUpdate{update("", "downloader", "settings", "capabilities")}
	}

	profile := session.Profile
	switch {
	case path == "/api/show-actions":
		return []liveUpdate{update(profile, "show-actions")}
	case strings.HasSuffix(path, "/favorite") && strings.HasPrefix(path, "/api/shows/"):
		return []liveUpdate{update(profile, "shows", "show", "calendar")}
	case strings.HasSuffix(path, "/watch-history") && strings.HasPrefix(path, "/api/shows/"):
		return profileAndAdmin(profile, []string{"shows", "show", "calendar", "logs"}, "logs")
	case strings.HasSuffix(path, "/bulk") && strings.HasPrefix(path, "/api/shows/"):
		return []liveUpdate{update(profile, "shows", "show", "calendar")}
	case strings.HasSuffix(path, "/refresh") && strings.HasPrefix(path, "/api/shows/"):
		return []liveUpdate{update("", "jobs")}
	case path == "/api/shows" || (strings.HasPrefix(path, "/api/shows/") && r.Method == http.MethodDelete):
		return profileAndAdmin(profile, []string{"shows", "show", "calendar", "show-actions", "logs"}, "logs")
	case strings.HasPrefix(path, "/api/episodes/"):
		return []liveUpdate{update(profile, "shows", "show", "calendar")}
	case strings.HasPrefix(path, "/api/inbox"):
		return []liveUpdate{update(profile, "inbox")}
	case path == "/api/preferences" || path == "/api/profile" || path == "/api/profile/avatar":
		return []liveUpdate{update(profile, "bootstrap")}
	case strings.HasPrefix(path, "/api/auth/sessions/") || path == "/api/auth/password":
		return []liveUpdate{update(profile, "bootstrap", "sessions")}
	case path == "/api/profiles" || strings.HasPrefix(path, "/api/profiles/"):
		return []liveUpdate{update("", "bootstrap")}
	case path == "/api/torrents/search" || path == "/api/torrents/send":
		return profileAndAdmin(profile, []string{"torrent-history"}, "statistics")
	}

	return nil
}
