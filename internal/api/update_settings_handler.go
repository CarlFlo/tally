package api

import (
	"errors"
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
	"github.com/CarlFlo/mediaManager/internal/jobs"
	"github.com/CarlFlo/mediaManager/internal/settings"
)

func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if e := s.operator(session); e != nil {
		return e
	}
	section := r.PathValue("section")
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
			Data     settings.Webhook `json:"data"`
			Revision int64            `json:"revision"`
		}
		if e := decode(r, &in); e != nil {
			return e
		}
		if e := settings.ValidateWebhook(in.Data); e != nil {
			return bad(e.Error())
		}
		value = in.Data.Defaults()
		revision = in.Revision
	case "search":
		var in struct {
			Data     settings.Search `json:"data"`
			Revision int64           `json:"revision"`
		}
		if e := decode(r, &in); e != nil {
			return e
		}
		if e := settings.ValidateSearch(in.Data); e != nil {
			return bad(e.Error())
		}
		value = in.Data.Effective()
		revision = in.Revision
	case "scheduling":
		var in jobs.Schedule
		if e := decode(r, &in); e != nil {
			return e
		}
		if e := s.Jobs.SaveSchedule(r.Context(), in); e != nil {
			if errors.Is(e, settings.ErrConflict) {
				return apiError{409, e.Error()}
			}
			return bad(e.Error())
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
	rev, e := s.settingsStore().Save(r.Context(), section, value, revision)
	if e != nil {
		if errors.Is(e, settings.ErrConflict) {
			return apiError{409, e.Error()}
		}
		return e
	}
	jsonResponse(w, 200, map[string]any{"revision": rev})
	return nil
}
