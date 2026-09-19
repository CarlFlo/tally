package torrent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/CarlFlo/tally/internal/database"
)

// EpisodeDownloadStore keeps the durable link between a Tally-owned torrent
// and the deployment-wide downloaded marker for its episode.
type EpisodeDownloadStore struct{ DB *database.Store }

func (s EpisodeDownloadStore) Track(ctx context.Context, infoHash, episodeID string) error {
	infoHash = normalizeInfoHash(infoHash)
	if infoHash == "" || strings.TrimSpace(episodeID) == "" {
		return fmt.Errorf("torrent episode tracking requires an infohash and episode")
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO torrent_episode_downloads(infohash,episode_id,created_at)
		VALUES(?,?,?) ON CONFLICT(infohash) DO UPDATE SET episode_id=excluded.episode_id`, infoHash, episodeID, time.Now().Unix())
	return err
}

func (s EpisodeDownloadStore) MarkCompleted(ctx context.Context, downloads []Download, completionPercent int) (int, error) {
	if completionPercent < 1 || completionPercent > 100 {
		return 0, fmt.Errorf("download completion percentage must be between 1 and 100")
	}
	hashes := make([]any, 0, len(downloads))
	threshold := float64(completionPercent) / 100
	for _, download := range downloads {
		hash := normalizeInfoHash(download.Hash)
		if download.Category == TallyCategory && hash != "" && download.Progress >= threshold {
			hashes = append(hashes, hash)
		}
	}
	if len(hashes) == 0 {
		return 0, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(hashes)), ",")
	result, err := s.DB.ExecContext(ctx, `UPDATE episodes SET downloaded=1
		WHERE downloaded=0
		AND ((airstamp<>'' AND julianday(airstamp)<=julianday('now')) OR (airstamp='' AND airdate<>'' AND airdate<date('now')))
		AND id IN (SELECT episode_id FROM torrent_episode_downloads WHERE infohash IN (`+placeholders+`))`, hashes...)
	if err != nil {
		return 0, err
	}
	changed, err := result.RowsAffected()
	return int(changed), err
}
