package jobs

import (
	"context"
)

func (s *Service) runBackup(ctx context.Context, trigger string) (result runResult, err error) {
	result.Attempt = 1

	backupKind := "manual"
	if trigger == "scheduled_refresh" {
		backupKind = "auto"
	}
	_, err = s.Backup.Create(ctx, backupKind)
	result.Candidates = 1
	if err == nil {
		result.Processed = 1
		result.Changes = 1
	}
	return result, err
}
