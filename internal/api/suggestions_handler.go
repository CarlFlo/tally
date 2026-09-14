package api

import (
	"context"
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
	"github.com/CarlFlo/mediaManager/internal/metadata"
)

func (s *Server) suggestions(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	out := []metadata.SearchResult{}
	if provider, ok := s.Metadata.Provider.(interface {
		SuggestShows(context.Context) ([]metadata.SearchResult, error)
	}); ok {
		var e error
		out, e = provider.SuggestShows(r.Context())
		if e != nil {
			return remote(e)
		}
	}
	return s.writeSearchResults(w, r, session, out)
}
