package api

import (
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) locales(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	if s.Locales == nil {
		return apiError{http.StatusServiceUnavailable, "localization is unavailable"}
	}
	jsonResponse(w, http.StatusOK, map[string]any{
		"revision": s.Locales.Revision(),
		"locales":  s.Locales.List(),
	})
	return nil
}

func (s *Server) localeCatalog(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	if s.Locales == nil {
		return apiError{http.StatusServiceUnavailable, "localization is unavailable"}
	}
	locale := r.PathValue("locale")
	catalog, ok := s.Locales.Catalog(locale)
	if !ok {
		return apiError{http.StatusConflict, "localization is unavailable"}
	}
	jsonResponse(w, http.StatusOK, catalog)
	return nil
}
