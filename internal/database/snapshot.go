package database

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
)

func (s *Store) Snapshot(ctx context.Context, path string) error {
	if _, e := s.ExecContext(ctx, "VACUUM INTO '"+strings.ReplaceAll(filepath.ToSlash(path), "'", "''")+"'"); e != nil {
		return e
	}
	if e := os.Chmod(path, 0600); e != nil {
		return e
	}
	db, e := sql.Open("sqlite", filepath.ToSlash(path))
	if e != nil {
		return e
	}
	defer db.Close()
	return Validate(ctx, db)
}
