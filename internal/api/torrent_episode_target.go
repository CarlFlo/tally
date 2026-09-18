package api

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"github.com/CarlFlo/tally/internal/metadata"
	"github.com/CarlFlo/tally/internal/torrent"
)

func (s *Server) torrentEpisodeTarget(ctx context.Context, profileID, episodeID string) (*torrent.EpisodeTarget, error) {
	if strings.TrimSpace(episodeID) == "" {
		return nil, nil
	}
	var showID, showName, premiered string
	var season, episode, runtime int
	err := s.DB.QueryRowContext(ctx, `SELECT e.show_id,s.name,s.premiered,e.season,e.number,COALESCE(NULLIF(e.runtime,0),NULLIF(s.runtime,0),0)
		FROM episodes e
		JOIN shows s ON s.id=e.show_id
		JOIN profile_shows p ON p.show_id=e.show_id AND p.profile_id=?
		WHERE e.id=?`, profileID, episodeID).Scan(&showID, &showName, &premiered, &season, &episode, &runtime)
	if err == sql.ErrNoRows {
		return nil, bad("episode is no longer in your followed shows")
	}
	if err != nil {
		return nil, err
	}
	return s.buildTorrentEpisodeTarget(ctx, showID, showName, premiered, season, episode, runtime)
}

func (s *Server) torrentEpisodeTargetFromQuery(ctx context.Context, query string) (*torrent.EpisodeTarget, error) {
	parsed := torrent.ParseReleaseName(query)
	if parsed.Title == "" || parsed.Episode <= 0 || parsed.MultiEpisode || parsed.SeasonPack {
		return nil, nil
	}
	target, err := s.torrentEpisodeTargetFromLocalMetadata(ctx, parsed)
	if err != nil || target != nil {
		return target, err
	}
	return s.torrentEpisodeTargetFromProvider(ctx, parsed)
}

func (s *Server) torrentEpisodeTargetFromLocalMetadata(ctx context.Context, parsed torrent.ParsedRelease) (*torrent.EpisodeTarget, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT s.id,s.name,s.premiered,e.season,e.number,COALESCE(NULLIF(e.runtime,0),NULLIF(s.runtime,0),0)
		FROM episodes e
		JOIN shows s ON s.id=e.show_id
		WHERE e.season=? AND e.number=?`, parsed.Season, parsed.Episode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type match struct {
		showID, showName, premiered string
		season, episode, runtime    int
	}
	matches := []match{}
	for rows.Next() {
		var candidate match
		if err = rows.Scan(&candidate.showID, &candidate.showName, &candidate.premiered, &candidate.season, &candidate.episode, &candidate.runtime); err != nil {
			return nil, err
		}
		if torrentQueryTitleMatchesShow(parsed.Title, candidate.showName, candidate.premiered) {
			matches = append(matches, candidate)
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(matches) != 1 {
		return nil, nil
	}
	candidate := matches[0]
	return s.buildTorrentEpisodeTarget(ctx, candidate.showID, candidate.showName, candidate.premiered, candidate.season, candidate.episode, candidate.runtime)
}

func (s *Server) torrentEpisodeTargetFromProvider(ctx context.Context, parsed torrent.ParsedRelease) (*torrent.EpisodeTarget, error) {
	if s.Metadata == nil || s.Metadata.Provider == nil {
		return nil, nil
	}
	lookupCtx := metadata.WithInfo(ctx, metadata.Info{Trigger: "torrent_search"})
	results, err := s.Metadata.Provider.SearchShows(lookupCtx, parsed.Title)
	if err != nil {
		return nil, remote(err)
	}

	matches := make([]metadata.Show, 0, 1)
	seen := map[int]bool{}
	for _, result := range results {
		show := result.Show
		if show.ID <= 0 || seen[show.ID] || !torrentQueryTitleMatchesShow(parsed.Title, show.Name, show.Premiered) {
			continue
		}
		seen[show.ID] = true
		matches = append(matches, show)
	}
	if len(matches) != 1 {
		return nil, nil
	}
	show := matches[0]
	episodes, err := s.Metadata.Provider.GetEpisodes(lookupCtx, strconv.Itoa(show.ID))
	if err != nil {
		return nil, remote(err)
	}
	var matched *metadata.Episode
	for i := range episodes {
		episode := &episodes[i]
		if episode.Season != parsed.Season || episode.Number != parsed.Episode {
			continue
		}
		if matched != nil {
			return nil, nil
		}
		matched = episode
	}
	if matched == nil {
		return nil, nil
	}
	runtime := matched.Runtime
	if runtime <= 0 {
		runtime = show.Runtime
	}
	target := &torrent.EpisodeTarget{
		ShowTitle:      show.Name,
		Season:         matched.Season,
		Episode:        matched.Number,
		RuntimeMinutes: runtime,
		TVMazeID:       strconv.Itoa(show.ID),
		IMDBID:         show.Externals.IMDB,
	}
	if show.Externals.TVDB > 0 {
		target.TVDBID = strconv.Itoa(show.Externals.TVDB)
	}
	if len(show.Premiered) >= 4 {
		target.Year, _ = strconv.Atoi(show.Premiered[:4])
	}
	return target, nil
}

func torrentQueryTitleMatchesShow(queryTitle, showName, premiered string) bool {
	showTitle := torrent.ParseReleaseName(showName).Title
	if showTitle == "" {
		return false
	}
	if queryTitle == showTitle {
		return true
	}
	return len(premiered) >= 4 && queryTitle == showTitle+" "+premiered[:4]
}

func (s *Server) buildTorrentEpisodeTarget(ctx context.Context, showID, showName, premiered string, season, episode, runtime int) (*torrent.EpisodeTarget, error) {
	target := &torrent.EpisodeTarget{ShowTitle: showName, Season: season, Episode: episode, RuntimeMinutes: runtime}
	if len(premiered) >= 4 {
		target.Year, _ = strconv.Atoi(premiered[:4])
	}
	rows, err := s.DB.QueryContext(ctx, "SELECT provider,external_id FROM external_ids WHERE kind='show' AND internal_id=?", showID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var provider, externalID string
		if err = rows.Scan(&provider, &externalID); err != nil {
			return nil, err
		}
		switch strings.ToLower(provider) {
		case "tvmaze":
			target.TVMazeID = externalID
		case "tvdb":
			target.TVDBID = externalID
		case "tmdb":
			target.TMDBID = externalID
		case "imdb":
			target.IMDBID = externalID
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return target, nil
}
