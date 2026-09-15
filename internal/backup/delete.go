package backup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/CarlFlo/mediaManager/internal/database"
)

func (s *Service) DeleteRecord(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var filename string
	if err = tx.QueryRowContext(ctx, "SELECT filename FROM backup_records WHERE id=?", id).Scan(&filename); errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	if !validArchiveName(filename) {
		return fmt.Errorf("unsafe backup filename")
	}
	path := filepath.Join(s.Path, filename)
	tombstone := filepath.Join(s.Path, ".delete-"+database.ID())
	renamed := false
	info, statErr := os.Lstat(path)
	switch {
	case statErr == nil:
		if !info.Mode().IsRegular() {
			return fmt.Errorf("backup archive is not a regular file")
		}
		if err = os.Rename(path, tombstone); err != nil {
			return err
		}
		renamed = true
	case os.IsNotExist(statErr):
	default:
		return statErr
	}
	restoreFile := func() {
		if renamed {
			_ = os.Rename(tombstone, path)
		}
	}
	result, err := tx.ExecContext(ctx, "DELETE FROM backup_records WHERE id=?", id)
	if err != nil {
		restoreFile()
		return err
	}
	if count, countErr := result.RowsAffected(); countErr != nil || count != 1 {
		restoreFile()
		if countErr != nil {
			return countErr
		}
		return ErrNotFound
	}
	if err = tx.Commit(); err != nil {
		restoreFile()
		return err
	}
	if renamed {
		_ = os.Remove(tombstone)
	}
	return nil
}
