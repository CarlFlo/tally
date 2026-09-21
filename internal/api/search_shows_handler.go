package api

import (
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) searchShows(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	query, err := normalizedSearchQuery(r.URL.Query().Get("q"))
	if err != nil {
		return err
	}
	results, e := s.Metadata.Provider.SearchShows(r.Context(), query)
	if e != nil {
		return remote(e)
	}
	return s.writeSearchResults(w, r, session, results)
}
