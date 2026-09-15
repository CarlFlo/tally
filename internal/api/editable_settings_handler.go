package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
	"github.com/CarlFlo/mediaManager/internal/settings"
)

func (s *Server) settingsStore() settings.Store { return settings.Store{DB: s.DB} }

func (s *Server) editableSettings(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if e := s.operator(session); e != nil {
		return e
	}
	// Secrets are deliberately visible to the deployment owner; never cache them.
	w.Header().Set("Cache-Control", "no-store")
	section := r.PathValue("section")
	if section == "scheduling" {
		rows, e := s.DB.Rows(r.Context(), "SELECT * FROM jobs ORDER BY key")
		if e != nil {
			return e
		}
		addSchedulePreviews(rows, s.Config.Timezone)
		jsonResponse(w, 200, rows)
		return nil
	}
	var out any
	switch section {
	case "backups":
		out = &settings.Backups{}
	case "notifications":
		out = &settings.Webhook{}
	case "search":
		out = &settings.Search{}
	default:
		return apiError{404, "unknown settings section"}
	}
	rev, e := s.settingsStore().Load(r.Context(), section, out)
	if e != nil {
		return e
	}
	if search, ok := out.(*settings.Search); ok {
		effective := search.Effective()
		out = &effective
	}
	jsonResponse(w, 200, map[string]any{"data": out, "revision": rev})
	return nil
}
