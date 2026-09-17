package api

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"github.com/CarlFlo/tally/internal/torrent"
)

func (s *Server) torrentEpisodeTarget(ctx context.Context, profileID, episodeID string) (*torrent.EpisodeTarget, error) {
	if strings.TrimSpace(episodeID) == "" {
		return nil, nil
	}
	var showID, showName, premiered string
	var season, episode int
	err := s.DB.QueryRowContext(ctx, `SELECT e.show_id,s.name,s.premiered,e.season,e.number
		FROM episodes e
		JOIN shows s ON s.id=e.show_id
		JOIN profile_shows p ON p.show_id=e.show_id AND p.profile_id=?
		WHERE e.id=?`, profileID, episodeID).Scan(&showID, &showName, &premiered, &season, &episode)
	if err == sql.ErrNoRows {
		return nil, bad("episode is no longer in your followed shows")
	}
	if err != nil {
		return nil, err
	}
	target := &torrent.EpisodeTarget{ShowTitle: showName, Season: season, Episode: episode}
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
