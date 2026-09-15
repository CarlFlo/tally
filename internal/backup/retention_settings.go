package backup

import (
	"context"
	"database/sql"
	"errors"

	"github.com/CarlFlo/tally/internal/settings"
)

func (s *Service) retentionCount(ctx context.Context) (int, error) {
	var value settings.Backups
	_, err := (settings.Store{DB: s.DB}).Load(ctx, "backups", &value)
	if errors.Is(err, sql.ErrNoRows) {
		if s.Keep > 0 {
			return s.Keep, nil
		}
		return 10, nil
	}
	if err != nil {
		return 0, err
	}
	if err = settings.ValidateBackups(value); err != nil {
		return 0, err
	}
	return value.Keep, nil
}
