package api

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) searchShows(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	query := strings.Join(strings.Fields(r.URL.Query().Get("q")), " ")
	if utf8.RuneCountInString(query) < 2 || len(query) > 200 {
		return bad("search with 2–200 characters")
	}
	results, e := s.Metadata.Provider.SearchShows(r.Context(), query)
	if e != nil {
		return remote(e)
	}
	return s.writeSearchResults(w, r, session, results)
}
