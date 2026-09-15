package api

import (
	"net/http"
	"strconv"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/metadata"
)

func (s *Server) writeSearchResults(w http.ResponseWriter, r *http.Request, session auth.Session, results []metadata.SearchResult) error {
	rows, e := s.DB.Rows(r.Context(), "SELECT x.external_id FROM external_ids x JOIN profile_shows p ON p.show_id=x.internal_id WHERE x.provider='tvmaze' AND x.kind='show' AND p.profile_id=?", session.Profile)
	if e != nil {
		return e
	}
	ids := map[string]bool{}
	for _, row := range rows {
		ids[row["external_id"].(string)] = true
	}
	out := []map[string]any{}
	for _, result := range results {
		out = append(out, map[string]any{"show": result.Show, "followed": ids[strconv.Itoa(result.Show.ID)]})
	}
	jsonResponse(w, 200, out)
	return nil
}
