package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/torrent"
)

func (s *Server) torrentSearch(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	var in struct {
		Query      string `json:"query"`
		EpisodeID  string `json:"episode_id"`
		MinSeeders int    `json:"min_seeders"`
		MinSize    int64  `json:"min_size"`
		MaxSize    int64  `json:"max_size"`
		Include    string `json:"include"`
		Exclude    string `json:"exclude"`
	}
	if e := decode(r, &in); e != nil {
		return e
	}
	in.Query = strings.Join(strings.Fields(in.Query), " ")
	if !s.torrentSearchEnabled(r.Context()) {
		return bad("torrent search is disabled in Settings")
	}
	if len(in.Query) < 2 || len(in.Query) > 200 {
		return bad("search with 2–200 characters")
	}
	if in.MinSize < 0 || in.MaxSize < 0 || in.MinSeeders < 0 {
		return bad("filters cannot be negative")
	}
	target, err := s.torrentEpisodeTarget(r.Context(), session.Profile, in.EpisodeID)
	if err != nil {
		return err
	}
	provider := s.jackett(r.Context())
	if provider == nil {
		return bad("configure and enable Jackett in Settings before searching")
	}
	results, err := provider.Search(r.Context(), torrent.SearchQuery{Query: in.Query, MinSeeders: in.MinSeeders, MinSize: in.MinSize, MaxSize: in.MaxSize, Include: in.Include, Exclude: in.Exclude})
	if err != nil {
		return remote(err)
	}
	if _, e := s.DB.ExecContext(r.Context(), "INSERT INTO torrent_search_history VALUES(?,?,?,?,?,?)", database.ID(), session.Profile, in.Query, "jackett", len(results), time.Now().Unix()); e != nil {
		return e
	}
	// Browser receives an opaque selection token. Provider URLs/API keys and
	// authoritative target data stay server-side inside the short-lived token.
	out := make([]map[string]any, 0, len(results))
	store := s.torrentAutomationStore()
	for _, result := range results {
		var preliminary *torrent.ReleaseAssessment
		previouslyBad := false
		if target != nil {
			assessment := torrent.EvaluateSearchCandidate(result, *target)
			preliminary = &assessment
			if result.InfoHash != "" {
				previouslyBad, err = store.IsBadInfoHash(r.Context(), result.InfoHash)
				if err != nil {
					return err
				}
			}
		}
		token := auth.Token()
		encoded, err := encodeTorrentSelection(torrentSelection{Result: result, Target: target, Preliminary: preliminary})
		if err != nil {
			return err
		}
		s.selections.Store(token, selection{Profile: session.Profile, Data: encoded, Expires: time.Now().Add(30 * time.Minute)})
		row := map[string]any{
			"id": token, "name": result.Name, "size": result.Size, "seeders": result.Seeders,
			"leechers": result.Leechers, "provider": result.Provider, "magnet": result.Magnet,
			"published": result.Published, "download_type": result.DownloadType,
			"sendable": result.Magnet != "" || result.URL != "",
		}
		if preliminary != nil {
			row["confidence"] = preliminary.Confidence
			row["verification"] = preliminary.Verification
			row["reasons"] = preliminary.Reasons
			row["hard_rejections"] = preliminary.HardRejections
			row["parsed"] = preliminary.Parsed
			row["previously_bad"] = previouslyBad
		}
		out = append(out, row)
	}
	s.selections.prune()
	jsonResponse(w, 200, map[string]any{"results": out, "warnings": []string{}})
	return nil
}
