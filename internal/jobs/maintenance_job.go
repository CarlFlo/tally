package jobs

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

func (s *Service) runMaintenance(ctx context.Context) (result runResult, err error) {
	result.Attempt = 1

	err = s.Control.Maintenance(ctx)
	if err != nil {
		return result, err
	}
	now := time.Now()
	_, err = s.DB.ExecContext(ctx, "DELETE FROM provider_requests WHERE created_at<?; DELETE FROM provider_request_aggregates WHERE day<?; DELETE FROM sessions WHERE expires_at<? OR last_seen<?; DELETE FROM torrent_search_history WHERE created_at<?; DELETE FROM torrent_send_history WHERE created_at<?; DELETE FROM job_runs WHERE status!='running' AND started_at<?", now.AddDate(0, 0, -s.Config.RawRetention).Unix(), now.AddDate(0, 0, -s.Config.AggregateRetention).Format("2006-01-02"), now.Unix(), now.Add(-s.Config.SessionIdle).Unix(), now.AddDate(0, 0, -30).Unix(), now.AddDate(0, 0, -90).Unix(), now.AddDate(0, 0, -365).Unix())
	if err == nil {
		files, _ := os.ReadDir(filepath.Join(s.Config.DataDir, "cache", "images"))
		for i, f := range files {
			if i >= s.Config.BatchSize {
				break
			}
			if ctx.Err() != nil {
				err = ctx.Err()
				return result, err
			}
			info, fileErr := f.Info()
			if fileErr == nil && !f.IsDir() && time.Since(info.ModTime()) > 30*24*time.Hour {
				_ = os.Remove(filepath.Join(s.Config.DataDir, "cache", "images", f.Name()))
				result.Changes++
			}
		}
	}
	if err == nil {
		err = s.DB.Checkpoint(ctx, false)
	}
	result.Processed = 1
	return result, err
}
