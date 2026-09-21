package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/settings"
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
	if err := decode(r, &in); err != nil {
		return err
	}
	var err error
	in.Query, err = normalizedSearchQuery(in.Query)
	if err != nil {
		return err
	}
	enabled, err := s.torrentSearchEnabled(r.Context())
	if err != nil {
		return err
	}
	if !enabled {
		return bad("torrent search is disabled in Settings")
	}
	if in.MinSize < 0 || in.MaxSize < 0 || in.MinSeeders < 0 {
		return bad("filters cannot be negative")
	}
	if len(in.Include) > 500 || len(in.Exclude) > 500 {
		return bad("include and exclude filters must be 500 characters or fewer")
	}

	provider, err := s.jackett(r.Context())
	if err != nil {
		return err
	}
	if provider == nil {
		return bad("configure and enable Jackett in Settings before searching")
	}
	results, err := provider.Search(r.Context(), torrent.SearchQuery{
		Query: in.Query, MinSeeders: in.MinSeeders, MinSize: in.MinSize,
		MaxSize: in.MaxSize, Include: in.Include, Exclude: in.Exclude,
	})
	if err != nil {
		return remote(err)
	}
	if _, err = s.DB.ExecContext(r.Context(),
		"INSERT INTO torrent_search_history VALUES(?,?,?,?,?,?)",
		database.ID(), session.Profile, in.Query, "jackett", len(results), time.Now().Unix(),
	); err != nil {
		return err
	}

	// Manual discovery stays cheap. Episode resolution, confidence, bad-hash
	// checks, trust signals, and runtime-normalized sizing are resolved only
	// when the user expands a result or attempts to submit it.
	out := make([]map[string]any, 0, len(results))
	for _, result := range results {
		token := auth.Token()
		encoded, encodeErr := encodeTorrentSelection(torrentSelection{
			Result: result, Query: in.Query, EpisodeID: in.EpisodeID,
		})
		if encodeErr != nil {
			return encodeErr
		}
		s.selections.Store(token, selection{
			Profile: session.Profile,
			Data:    encoded,
			Expires: time.Now().Add(30 * time.Minute),
		})
		out = append(out, map[string]any{
			"id": token, "name": result.Name, "size": result.Size, "seeders": result.Seeders,
			"leechers": result.Leechers, "provider": result.Provider, "uploader": result.Uploader,
			"magnet": result.Magnet, "published": result.Published, "download_type": result.DownloadType,
			"sendable": result.Magnet != "" || result.URL != "",
		})
	}
	s.selections.prune()
	jsonResponse(w, http.StatusOK, map[string]any{"results": out, "warnings": []string{}})
	return nil
}

func (s *Server) resolveTorrentSelection(ctx context.Context, profileID string, payload torrentSelection) (torrentSelection, error) {
	if payload.Target == nil {
		target, err := s.torrentEpisodeTarget(ctx, profileID, payload.EpisodeID)
		if err != nil {
			return payload, err
		}
		if target == nil && strings.TrimSpace(payload.Query) != "" {
			target, err = s.torrentEpisodeTargetFromQuery(ctx, payload.Query)
			if err != nil {
				return payload, err
			}
		}
		payload.Target = target
	}
	if payload.Preliminary == nil && payload.Target != nil {
		assessment := torrent.EvaluateSearchCandidate(payload.Result, *payload.Target)
		payload.Preliminary = &assessment
	}
	return payload, nil
}

func (s *Server) torrentSearchEvaluation(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	token := r.PathValue("id")
	value, ok := s.selections.Load(token)
	if !ok {
		return bad("search result expired; search again")
	}
	selected := value.(selection)
	if selected.Profile != session.Profile || time.Now().After(selected.Expires) {
		s.selections.Delete(token)
		return bad("search result expired; search again")
	}
	payload, err := decodeTorrentSelection(selected.Data)
	if err != nil {
		return err
	}
	payload, err = s.resolveTorrentSelection(r.Context(), session.Profile, payload)
	if err != nil {
		return err
	}
	encoded, err := encodeTorrentSelection(payload)
	if err != nil {
		return err
	}
	selected.Data = encoded
	s.selections.Store(token, selected)

	var stored settings.TorrentAutomation
	if _, err := s.settingsStore().Load(r.Context(), "torrent_automation", &stored); err != nil {
		return err
	}
	config := stored.Effective()

	result := payload.Result
	parsed := torrent.ParseReleaseName(result.Name)
	if payload.Preliminary != nil {
		parsed = payload.Preliminary.Parsed
	}
	out := map[string]any{
		"parsed":      parsed,
		"preferences": torrent.AutomationPreferenceSignals(result, parsed, config),
	}

	knownHash := result.InfoHash
	if knownHash == "" {
		knownHash = torrent.MagnetInfoHash(result.Magnet)
	}
	if knownHash != "" {
		previouslyBad, checkErr := s.torrentAutomationStore().IsBadInfoHash(r.Context(), knownHash)
		if checkErr != nil {
			return checkErr
		}
		out["previously_bad"] = previouslyBad
	}

	if payload.Target != nil && payload.Target.RuntimeMinutes > 0 && result.Size > 0 {
		profile := s.manualTorrentMediaProfile(r.Context(), payload.Target)
		sizeProfile := torrent.EvaluateSizeProfile(result.Size, payload.Target.RuntimeMinutes, config, profile)
		out["size_profile"] = sizeProfile
		if sizeProfile.Known {
			out["mb_per_minute"] = sizeProfile.MBPerMinute
		}
	}
	if payload.Preliminary != nil {
		out["confidence"] = payload.Preliminary.Confidence
		out["verification"] = payload.Preliminary.Verification
		out["reasons"] = payload.Preliminary.Reasons
		out["hard_rejections"] = payload.Preliminary.HardRejections
	}

	jsonResponse(w, http.StatusOK, out)
	return nil
}

func (s *Server) manualTorrentMediaProfile(ctx context.Context, target *torrent.EpisodeTarget) string {
	if target == nil {
		return torrent.MediaProfileLive
	}
	refs := []struct {
		provider string
		id       string
	}{
		{"tvmaze", target.TVMazeID},
		{"tvdb", target.TVDBID},
		{"tmdb", target.TMDBID},
		{"imdb", target.IMDBID},
	}
	for _, ref := range refs {
		if ref.id == "" {
			continue
		}
		var showID string
		if err := s.DB.QueryRowContext(ctx,
			"SELECT internal_id FROM external_ids WHERE provider=? AND kind='show' AND external_id=?",
			ref.provider, ref.id,
		).Scan(&showID); err != nil {
			continue
		}
		profile, err := s.torrentAutomationStore().ShowMediaProfile(ctx, showID)
		if err == nil && profile.Effective == torrent.MediaProfileAnimated {
			return torrent.MediaProfileAnimated
		}
		return torrent.MediaProfileLive
	}
	return torrent.MediaProfileLive
}
