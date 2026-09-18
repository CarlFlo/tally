package torrent

import (
	"context"
	"database/sql"
	"time"

	"github.com/CarlFlo/tally/internal/settings"
)

func automationRetryDelay(config settings.TorrentAutomation, attempts int) time.Duration {
	minutes := config.RetryLaterMinutes
	switch {
	case attempts <= 1:
		minutes = config.RetryFirstMinutes
	case attempts == 2:
		minutes = config.RetrySecondMinutes
	}
	return time.Duration(minutes) * time.Minute
}

// claimEpisodeSearch durably advances the retry gate before any outbound
// discovery request. A crash, timeout or provider failure therefore cannot
// cause the next scheduler tick to immediately repeat the same search.
func (s *AutomationService) claimEpisodeSearch(ctx context.Context, episodeID string, now time.Time, config settings.TorrentAutomation) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	attempts := 0
	err = tx.QueryRowContext(ctx, "SELECT attempts FROM torrent_automation_episode_state WHERE episode_id=?", episodeID).Scan(&attempts)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	attempts++
	next := now.Add(automationRetryDelay(config, attempts)).Unix()
	if err == sql.ErrNoRows {
		_, err = tx.ExecContext(ctx, `INSERT INTO torrent_automation_episode_state(
			episode_id,attempts,next_search_at,last_search_at,updated_at
		) VALUES(?,?,?,?,?)`, episodeID, attempts, next, now.Unix(), now.Unix())
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE torrent_automation_episode_state
			SET attempts=?,next_search_at=?,last_search_at=?,updated_at=?
			WHERE episode_id=?`, attempts, next, now.Unix(), now.Unix(), episodeID)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

// postponeEpisodeSearch moves an existing retry gate forward without counting
// a second discovery request. It is used when later payload verification turns
// a previously accepted magnet into a retryable rejected attempt.
func (s *AutomationService) postponeEpisodeSearch(ctx context.Context, episodeID string, now time.Time, config settings.TorrentAutomation) error {
	var attempts int
	err := s.DB.QueryRowContext(ctx, "SELECT attempts FROM torrent_automation_episode_state WHERE episode_id=?", episodeID).Scan(&attempts)
	if err == sql.ErrNoRows {
		attempts = 1
		next := now.Add(automationRetryDelay(config, attempts)).Unix()
		_, err = s.DB.ExecContext(ctx, `INSERT INTO torrent_automation_episode_state(
			episode_id,attempts,next_search_at,last_search_at,updated_at
		) VALUES(?,?,?,?,?)`, episodeID, attempts, next, 0, now.Unix())
		return err
	}
	if err != nil {
		return err
	}
	if attempts < 1 {
		attempts = 1
	}
	next := now.Add(automationRetryDelay(config, attempts)).Unix()
	_, err = s.DB.ExecContext(ctx, `UPDATE torrent_automation_episode_state
		SET next_search_at=?,updated_at=? WHERE episode_id=?`, next, now.Unix(), episodeID)
	return err
}
