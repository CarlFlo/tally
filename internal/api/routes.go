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
		access  routeAccess
	}{
		{"GET /api/settings/{section}", s.editableSettings, adminRoute},
		{"GET /api/events", s.events, publicRoute},
		{"GET /api/locales", s.locales, publicRoute},
		{"GET /api/locales/{locale}", s.localeCatalog, publicRoute},
		{"PUT /api/settings/{section}", s.updateSettings, adminRoute},
		{"POST /api/settings/search/test", s.testJackett, adminRoute},
		{"POST /api/settings/scheduling/preview", s.schedulePreview, adminRoute},
		{"POST /api/jobs/{kind}/resume", s.resumeJob, adminRoute},
		{"GET /api/show-actions", s.showActions, authenticatedRoute},
		{"POST /api/show-actions", s.queueShowAction, authenticatedRoute},
		{"GET /api/shows/suggestions", s.suggestions, authenticatedRoute},
		{"PATCH /api/shows/{id}/favorite", s.favoriteShow, authenticatedRoute},
		{"POST /api/settings/notifications/test", s.testNotification, adminRoute},
		{"POST /api/settings/notifications/preview", s.notificationTimePreview, adminRoute},
		{"GET /api/capabilities", s.capabilities, authenticatedRoute},
		{"GET /api/backups", s.backups, adminRoute},
		{"POST /api/backups", s.createManualBackup, adminRoute},
		{"GET /api/backups/{id}/download", s.downloadBackup, adminRoute},
		{"POST /api/backups/{id}/restore", s.restoreBackup, adminRoute},
		{"DELETE /api/backups/{id}", s.deleteBackup, adminRoute},
		{"GET /api/inbox", s.inbox, authenticatedRoute},
		{"POST /api/inbox/{action}", s.markInbox, authenticatedRoute},
		{"DELETE /api/inbox/{id}", s.dismissInbox, authenticatedRoute},
		{"GET /api/logs", s.logs, authenticatedRoute},
		{"PATCH /api/browser/preferences", s.browserPreferences, publicRoute},
		{"DELETE /api/shows/{id}/watch-history", s.clearWatchHistory, authenticatedRoute},
		{"GET /api/bootstrap", s.bootstrap, publicRoute},
		{"POST /api/auth/register", s.registerProfile, publicRoute},
		{"POST /api/auth/login", s.login, publicRoute},
		{"POST /api/auth/logout", s.logout, publicRoute},
		{"POST /api/auth/password", s.password, authenticatedRoute},
		{"GET /api/auth/sessions", s.sessions, authenticatedRoute},
		{"DELETE /api/auth/sessions/{id}", s.revokeSession, authenticatedRoute},
		{"GET /api/profiles", s.profiles, authenticatedRoute},
		{"POST /api/profiles", s.createProfile, adminRoute},
		{"POST /api/profiles/select", s.selectProfile, publicRoute},
		{"PATCH /api/profile", s.updateProfile, authenticatedRoute},
		{"PATCH /api/profiles/{id}/admin", s.updateProfileAdmin, adminRoute},
		{"PATCH /api/profiles/{id}/authentication", s.updateProfileAuthentication, adminRoute},
		{"DELETE /api/profiles/{id}", s.deleteProfile, authenticatedRoute},
		{"POST /api/profile/avatar", s.uploadAvatar, authenticatedRoute},
		{"GET /api/avatars/{name}", s.avatar, publicRoute},
		{"PATCH /api/preferences", s.preferences, authenticatedRoute},
		{"GET /api/shows", s.shows, authenticatedRoute},
		{"GET /api/shows/search", s.searchShows, authenticatedRoute},
		{"POST /api/shows", s.addShow, authenticatedRoute},
		{"GET /api/shows/{id}", s.show, authenticatedRoute},
		{"DELETE /api/shows/{id}", s.removeShow, authenticatedRoute},
		{"POST /api/shows/{id}/refresh", s.refreshShow, authenticatedRoute},
		{"POST /api/shows/{id}/bulk", s.bulkState, authenticatedRoute},
		{"GET /api/calendar", s.calendar, authenticatedRoute},
		{"PATCH /api/episodes/{id}", s.episodeState, authenticatedRoute},
		{"GET /api/images", s.image, authenticatedRoute},
		{"GET /api/jobs", s.jobs, adminRoute},
		{"POST /api/jobs/{kind}", s.runJob, adminRoute},
		{"DELETE /api/jobs/runs/{id}", s.cancelJob, adminRoute},
		{"GET /api/statistics", s.statistics, adminRoute},
		{"GET /api/settings", s.settings, adminRoute},
		{"GET /api/alerts", s.alerts, adminRoute},
		{"DELETE /api/alerts/{id}", s.dismissAlert, adminRoute},
		{"POST /api/torrents/search", s.torrentSearch, authenticatedRoute},
		{"GET /api/torrents/search/{id}/evaluation", s.torrentSearchEvaluation, authenticatedRoute},
		{"POST /api/torrents/send", s.torrentSend, authenticatedRoute},
		{"GET /api/torrents/downloads", s.torrentDownloads, authenticatedRoute},
		{"POST /api/torrents/downloads/{hash}/{action}", s.torrentDownloadAction, authenticatedRoute},
		{"DELETE /api/torrents/downloads/{hash}", s.removeTorrentDownload, authenticatedRoute},
		{"DELETE /api/torrents/history/{kind}", s.clearTorrentHistory, authenticatedRoute},
		{"GET /api/torrents/automation/runs", s.torrentAutomationRuns, authenticatedRoute},
		{"GET /api/flows", s.flowList, adminRoute},
		{"POST /api/flows", s.flowCreate, adminRoute},
		{"GET /api/flows/runs", s.flowRuns, adminRoute},
		{"GET /api/flows/runs/{id}", s.flowRun, adminRoute},
		{"GET /api/flows/{id}", s.flowGet, adminRoute},
		{"PUT /api/flows/{id}", s.flowUpdate, adminRoute},
		{"DELETE /api/flows/{id}", s.flowDelete, adminRoute},
		{"POST /api/flows/{id}/reset", s.flowReset, adminRoute},
		{"POST /api/flows/{id}/replay", s.flowReplay, adminRoute},
		{"GET /api/torrents/automation/runs/{id}", s.torrentAutomationRun, authenticatedRoute},
		{"POST /api/torrents/automation/runs/{id}/bad", s.markTorrentAutomationRunBad, authenticatedRoute},
		{"GET /api/torrents/automation/shows", s.torrentAutomationShows, authenticatedRoute},
		{"GET /api/torrents/automation/shows/{id}", s.torrentShowPolicy, authenticatedRoute},
		{"PUT /api/torrents/automation/shows/{id}", s.updateTorrentShowPolicy, adminRoute},
		{"PUT /api/torrents/automation/shows/{id}/chain", s.updateTorrentShowChain, adminRoute},
		{"GET /api/torrents/automation/shows/{id}/media-profile", s.torrentShowMediaProfile, authenticatedRoute},
		{"PUT /api/torrents/automation/shows/{id}/media-profile", s.updateTorrentShowMediaProfile, adminRoute},
		{"GET /api/downloader", s.getDownloader, adminRoute},
		{"PUT /api/downloader", s.saveDownloader, adminRoute},
		{"POST /api/downloader/test", s.testDownloader, adminRoute},
		{"GET /api/torrents/history", s.torrentHistory, authenticatedRoute},
	}
	for _, r := range routes {
		mux.HandleFunc(r.pattern, s.wrap(r.fn, r.access))
	}
	mux.HandleFunc("/", s.assets)
	return s.security(mux)
}
