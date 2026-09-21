package api

import (
	"errors"
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/jobs"
	"github.com/CarlFlo/tally/internal/settings"
)

func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if e := s.operator(session); e != nil {
		return e
	}
	section := r.PathValue("section")
	key := section
	var value any
	var revision int64
	switch section {
	case "backups":
		var in struct {
			Data     settings.Backups `json:"data"`
			Revision int64            `json:"revision"`
		}
		if e := decode(r, &in); e != nil {
			return e
		}
		if e := settings.ValidateBackups(in.Data); e != nil {
			return bad(e.Error())
		}
		value, revision = in.Data, in.Revision
	case "notifications":
		var in struct {
			Data         settings.Webhook `json:"data"`
			Revision     int64            `json:"revision"`
			ClearSecrets map[string]bool  `json:"clear_secrets,omitempty"`
		}
		if e := decode(r, &in); e != nil {
			return e
		}
		for key := range in.ClearSecrets {
			if key != "url" && key != "discord_url" {
				return bad("unknown notification secret")
			}
		}
		var current settings.Webhook
		if _, e := s.settingsStore().Load(r.Context(), "notifications", &current); e != nil {
			return e
		}
		in.Data = settings.MergeWebhookSecrets(in.Data, current, in.ClearSecrets)
		if e := settings.ValidateWebhook(in.Data, s.Config.Timezone); e != nil {
			return bad(e.Error())
		}
		value = in.Data.Defaults(s.Config.Timezone)
		revision = in.Revision
	case "search":
		var in struct {
			Data     settings.Search `json:"data"`
			Revision int64           `json:"revision"`
		}
		if e := decode(r, &in); e != nil {
			return e
		}
		var current settings.Search
		if _, e := s.settingsStore().Load(r.Context(), "search", &current); e != nil {
			return e
		}
		in.Data = settings.MergeSearchSecrets(in.Data, current)
		if e := settings.ValidateSearch(in.Data); e != nil {
			return bad(e.Error())
		}
		value = in.Data.Effective()
		revision = in.Revision
	case "torrent":
		var in struct {
			Data     settings.Torrent `json:"data"`
			Revision int64            `json:"revision"`
		}
		if e := decode(r, &in); e != nil {
			return e
		}
		value, revision = in.Data, in.Revision
	case "torrent-automation":
		var in struct {
			Data     settings.TorrentAutomation `json:"data"`
			Revision int64                      `json:"revision"`
		}
		if e := decode(r, &in); e != nil {
			return e
		}
		in.Data = in.Data.Effective()
		if e := settings.ValidateTorrentAutomation(in.Data); e != nil {
			return bad(e.Error())
		}
		value, revision = in.Data, in.Revision
		key = "torrent_automation"
	case "scheduling":
		var in jobs.Schedule
		if e := decode(r, &in); e != nil {
			return e
		}
		if e := s.Jobs.SaveSchedule(r.Context(), in, session.Profile); e != nil {
			if errors.Is(e, settings.ErrConflict) {
				return apiError{409, e.Error()}
			}
			return jobRequestError(e)
		}
		var revision int64
		if e := s.DB.QueryRowContext(r.Context(), "SELECT revision FROM jobs WHERE key=?", in.Key).Scan(&revision); e != nil {
			return e
		}
		jsonResponse(w, 200, map[string]any{"ok": true, "revision": revision})
		return nil
	default:
		return apiError{404, "unknown settings section"}
	}
	rev, e := s.settingsStore().Save(r.Context(), key, value, revision, session.Profile)
	if e != nil {
		if errors.Is(e, settings.ErrConflict) {
			return apiError{409, e.Error()}
		}
		return e
	}
	jsonResponse(w, 200, map[string]any{"revision": rev})
	return nil
}
