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
	if e := os.MkdirAll(s.Path, 0700); e != nil {
		return "", e
	}
	stage, e := os.MkdirTemp(s.Path, ".backup-")
	if e != nil {
		return "", e
	}
	defer os.RemoveAll(stage)
	snapshot := filepath.Join(stage, "app.db")
	if e = s.DB.Snapshot(ctx, snapshot); e != nil {
		return "", e
	}
	manifest, files, e := s.snapshotFiles(ctx, snapshot)
	if e != nil {
		return "", e
	}
	filename := fmt.Sprintf("tally-%s-%s-%s.zip", kind, time.Now().UTC().Format("20060102-150405"), database.ID()[:8])
	temp := filepath.Join(stage, filename)
	if e = writeArchive(ctx, temp, files, manifest); e != nil {
		return "", e
	}
	verify, e := os.MkdirTemp(stage, "verify-")
	if e != nil {
		return "", e
	}
	if _, e = Extract(ctx, temp, verify); e != nil {
		return "", fmt.Errorf("backup validation failed: %w", e)
	}
	final := filepath.Join(s.Path, filename)
	if e = os.Rename(temp, final); e != nil {
		return "", e
	}
	if kind == "auto" {
		e = s.Retain(ctx)
	}
	return filename, e
}
