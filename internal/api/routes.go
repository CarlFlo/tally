package api

import (
	"net/http"

	"github.com/CarlFlo/tally/internal/live"
)

func (s *Server) Handler() http.Handler {
	if s.Events == nil {
		s.Events = live.New()
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { jsonResponse(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /readyz", s.readiness)
	routes := []struct {
		pattern string
		fn      handler
		public  bool
	}{
		{"GET /api/settings/{section}", s.editableSettings, false},
		{"GET /api/events", s.events, true},
		{"GET /api/locales", s.locales, true},
		{"GET /api/locales/{locale}", s.localeCatalog, true},
		{"PUT /api/settings/{section}", s.updateSettings, false},
		{"POST /api/settings/search/test", s.testJackett, false},
		{"POST /api/settings/scheduling/preview", s.schedulePreview, false},
		{"POST /api/jobs/{kind}/resume", s.resumeJob, false},
		{"GET /api/show-actions", s.showActions, false},
		{"POST /api/show-actions", s.queueShowAction, false},
		{"GET /api/shows/suggestions", s.suggestions, false},
		{"PATCH /api/shows/{id}/favorite", s.favoriteShow, false},
		{"POST /api/settings/notifications/test", s.testNotification, false},
		{"POST /api/settings/notifications/preview", s.notificationTimePreview, false},
		{"GET /api/capabilities", s.capabilities, false},
		{"GET /api/backups", s.backups, false},
		{"POST /api/backups", s.createManualBackup, false},
		{"GET /api/backups/{id}/download", s.downloadBackup, false},
		{"POST /api/backups/{id}/restore", s.restoreBackup, false},
		{"DELETE /api/backups/{id}", s.deleteBackup, false},
		{"GET /api/inbox", s.inbox, false},
		{"POST /api/inbox/{action}", s.markInbox, false},
		{"DELETE /api/inbox/{id}", s.dismissInbox, false},
		{"GET /api/logs", s.logs, false},
		{"PATCH /api/browser/preferences", s.browserPreferences, true},
		{"DELETE /api/shows/{id}/watch-history", s.clearWatchHistory, false},
		{"GET /api/bootstrap", s.bootstrap, true},
		{"POST /api/auth/register", s.registerProfile, true},
		{"POST /api/auth/login", s.login, true},
		{"POST /api/auth/recover", s.recover, true},
		{"POST /api/auth/logout", s.logout, true},
		{"POST /api/auth/password", s.password, false},
		{"GET /api/auth/sessions", s.sessions, false},
		{"DELETE /api/auth/sessions/{id}", s.revokeSession, false},
		{"GET /api/profiles", s.profiles, false},
		{"POST /api/profiles", s.createProfile, false},
		{"POST /api/profiles/select", s.selectProfile, true},
		{"PATCH /api/profile", s.updateProfile, false},
		{"PATCH /api/profiles/{id}/admin", s.updateProfileAdmin, false},
		{"PATCH /api/profiles/{id}/authentication", s.updateProfileAuthentication, false},
		{"DELETE /api/profiles/{id}", s.deleteProfile, false},
		{"POST /api/profile/avatar", s.uploadAvatar, false},
		{"GET /api/avatars/{name}", s.avatar, true},
		{"PATCH /api/preferences", s.preferences, false},
		{"GET /api/shows", s.shows, false},
		{"GET /api/shows/search", s.searchShows, false},
		{"POST /api/shows", s.addShow, false},
		{"GET /api/shows/{id}", s.show, false},
		{"DELETE /api/shows/{id}", s.removeShow, false},
		{"POST /api/shows/{id}/refresh", s.refreshShow, false},
		{"POST /api/shows/{id}/bulk", s.bulkState, false},
		{"GET /api/calendar", s.calendar, false},
		{"PATCH /api/episodes/{id}", s.episodeState, false},
		{"GET /api/images", s.image, false},
		{"GET /api/jobs", s.jobs, false},
		{"POST /api/jobs/{kind}", s.runJob, false},
		{"DELETE /api/jobs/runs/{id}", s.cancelJob, false},
		{"GET /api/statistics", s.statistics, false},
		{"GET /api/settings", s.settings, false},
		{"GET /api/alerts", s.alerts, false},
		{"DELETE /api/alerts/{id}", s.dismissAlert, false},
		{"POST /api/torrents/search", s.torrentSearch, false},
		{"POST /api/torrents/send", s.torrentSend, false},
		{"GET /api/torrents/downloads", s.torrentDownloads, false},
		{"POST /api/torrents/downloads/{hash}/{action}", s.torrentDownloadAction, false},
		{"DELETE /api/torrents/downloads/{hash}", s.removeTorrentDownload, false},
		{"DELETE /api/torrents/history/{kind}", s.clearTorrentHistory, false},
		{"GET /api/torrents/automation/runs", s.torrentAutomationRuns, false},
		{"GET /api/torrents/automation/runs/{id}", s.torrentAutomationRun, false},
		{"POST /api/torrents/automation/runs/{id}/bad", s.markTorrentAutomationRunBad, false},
		{"GET /api/torrents/automation/shows", s.torrentAutomationShows, false},
		{"GET /api/torrents/automation/shows/{id}", s.torrentShowPolicy, false},
		{"PUT /api/torrents/automation/shows/{id}", s.updateTorrentShowPolicy, false},
		{"GET /api/torrents/automation/shows/{id}/media-profile", s.torrentShowMediaProfile, false},
		{"PUT /api/torrents/automation/shows/{id}/media-profile", s.updateTorrentShowMediaProfile, false},
		{"GET /api/downloader", s.getDownloader, false},
		{"PUT /api/downloader", s.saveDownloader, false},
		{"POST /api/downloader/test", s.testDownloader, false},
		{"GET /api/torrents/history", s.torrentHistory, false},
	}
	for _, r := range routes {
		mux.HandleFunc(r.pattern, s.wrap(r.fn, r.public))
	}
	mux.HandleFunc("/", s.assets)
	return s.security(mux)
}
