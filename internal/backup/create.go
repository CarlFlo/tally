package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CarlFlo/tally/internal/database"
)

func (s *Service) Create(ctx context.Context, kind string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.Path, 0700); err != nil {
		return "", err
	}
	stage, err := os.MkdirTemp(s.Path, ".backup-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	snapshot := filepath.Join(stage, "app.db")
	if err = s.DB.Snapshot(ctx, snapshot); err != nil {
		return "", err
	}
	manifest, files, err := s.snapshotFiles(ctx, snapshot)
	if err != nil {
		return "", err
	}
	filename := fmt.Sprintf("tally-%s-%s-%s.zip", kind, time.Now().UTC().Format("20060102-150405"), database.ID()[:8])
	temp := filepath.Join(stage, filename)
	if err = writeArchive(ctx, temp, files, manifest); err != nil {
		return "", err
	}
	verify, err := os.MkdirTemp(stage, "verify-")
	if err != nil {
		return "", err
	}
	if _, err = Extract(ctx, temp, verify); err != nil {
		return "", fmt.Errorf("backup validation failed: %w", err)
	}
	final := filepath.Join(s.Path, filename)
	if err = os.Rename(temp, final); err != nil {
		return "", err
	}
	if info, statErr := os.Stat(final); statErr == nil {
		s.rememberInspection(filename, info, manifest, nil)
	}
	if kind == "auto" {
		err = s.Retain(ctx)
	}
	return filename, err
}
