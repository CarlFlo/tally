package backup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func (s *Service) RestoreRecord(ctx context.Context, id string) (Manifest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var filename string
	var verified bool
	err := s.DB.QueryRowContext(ctx, "SELECT filename,verified FROM backup_records WHERE id=?", id).Scan(&filename, &verified)
	if errors.Is(err, sql.ErrNoRows) {
		return Manifest{}, ErrNotFound
	}
	if err != nil {
		return Manifest{}, err
	}
	if !verified {
		return Manifest{}, ErrUnverified
	}
	archive, err := s.archivePath(filename)
	if err != nil {
		return Manifest{}, err
	}
	return s.restoreLive(ctx, archive)
}

// RestoreLive validates and migrates an archive in staging before atomically
// replacing durable database state in the running process.
func (s *Service) RestoreLive(ctx context.Context, archive string) (Manifest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.restoreLive(ctx, archive)
}

func (s *Service) restoreLive(ctx context.Context, archive string) (Manifest, error) {
	stage, err := os.MkdirTemp(s.DataDir, ".live-restore-")
	if err != nil {
		return Manifest{}, err
	}
	defer os.RemoveAll(stage)
	manifest, err := Extract(ctx, archive, stage)
	if err != nil {
		return manifest, fmt.Errorf("validate backup: %w", err)
	}
	if err = prepareStagedDatabase(ctx, stage); err != nil {
		return manifest, fmt.Errorf("prepare backup: %w", err)
	}
	created, err := installRestoredAvatars(stage, s.DataDir)
	if err != nil {
		return manifest, fmt.Errorf("prepare avatars: %w", err)
	}
	if err = replaceDatabaseState(ctx, s.DB, filepath.Join(stage, "app.db")); err != nil {
		for _, path := range created {
			_ = os.Remove(path)
		}
		return manifest, fmt.Errorf("apply backup: %w", err)
	}
	return manifest, nil
}
